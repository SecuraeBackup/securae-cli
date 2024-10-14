/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var uploadCmd = &cobra.Command{
	Use:   "upload [filename] [flags]",
	Short: "Upload backup files",
	Long: `Upload files into the backup ID (UUID format) defined in the web UI.
For example:

  securae upload database-dump.tar.gz --backup-id=abcd1234-ab12-ab12-ab12-abcdef123456

Or you can also use an environment variable:

  export SECURAE_BACKUP_ID=abcd1234-ab12-ab12-ab12-abcdef123456
  securae upload database-dump.tar.gz

`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			return fmt.Errorf("A filename must be specified.")
		}
		if err := cobra.MaximumNArgs(1)(cmd, args); err != nil {
			return fmt.Errorf("Only one filename must be specified.")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := viper.GetString("api.url")
		apiToken := viper.GetString("api.token")

		backupId := viper.GetString("backup-id")
		if backupId == "" {
			return fmt.Errorf("A Backup ID must be specified.")
		}

		filename := args[0]
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		url := fmt.Sprintf("%s/backups/%s/preupload/", apiURL, backupId)
		filenameOnly := filepath.Base(filename)
		presignedURL, err := fetchPresignedURL(url, apiToken, []byte(fmt.Sprintf(`{"filename": "%s"}`, filenameOnly)))
		if err != nil {
			return err
		}

		cmd.Printf("Uploading file %s... ", filename)
		err = uploadFile(presignedURL, file)
		if err == nil {
			cmd.Printf("OK\n")
		} else {
			panic(err)
		}
		return nil

	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().StringP("backup-id", "b", "", "A backup ID (`UUID` format) where your files will be stored. It can also be specified using the environment variable SECURAE_BACKUP_ID.")
}

func fetchPresignedURL(url string, token string, data []byte) (string, error) {
	tr := &http.Transport{
		TLSHandshakeTimeout:   2 * time.Second,
		IdleConnTimeout:       2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("Error fetching presigned URL: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var objmap map[string]interface{}
	if err := json.Unmarshal(body, &objmap); err != nil {
		return "", err
	}
	return objmap["url"].(string), nil
}

func uploadFile(url string, file *os.File) error {
	buffer := bytes.NewBuffer(nil)
	if _, err := io.Copy(buffer, file); err != nil {
		return err
	}

	tr := &http.Transport{
		TLSHandshakeTimeout:   2 * time.Second,
		IdleConnTimeout:       2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
	}
	client := &http.Client{Transport: tr}
	request, err := http.NewRequest(http.MethodPut, url, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "multipart/form-data")

	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error uploading file: %s", resp.Status)
	}

	return nil
}
