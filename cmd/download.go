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

var downloadCmd = &cobra.Command{
	Use:   "download [filename] [flags]",
	Short: "download backup files",
	Long: `download files using a backup ID (UUID format), as defined in the web UI.
For example:

  securae download database-dump.tar.gz --backup-id=abcd1234-ab12-ab12-ab12-abcdef123456

Or you can also use an environment variable:

  export SECURAE_BACKUP_ID=abcd1234-ab12-ab12-ab12-abcdef123456
  securae download database-dump.tar.gz

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

		encryptionKeyB64Encoded := viper.GetString("encryption-key-b64encoded")
		if encryptionKeyB64Encoded == "" {
			return fmt.Errorf("An encryption key is mandatory.")
		}

		filename := args[0]

		url := fmt.Sprintf("%s/backups/%s/predownload/", apiURL, backupId)
		filenameOnly := filepath.Base(filename)
		presignedURL, err := fetchPresignedDownloadURL(url, apiToken, []byte(fmt.Sprintf(`{"filename": "%s"}`, filenameOnly)))
		if err != nil {
			return err
		}

		cmd.Printf("downloading file %s... ", filename)
		err = downloadFile(presignedURL, encryptionKeyB64Encoded, filename)
		if err == nil {
			cmd.Printf("OK\n")
		} else {
			panic(err)
		}
		return nil

	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().StringP("backup-id", "b", "", "A backup ID (`UUID` format) where your files will be stored. It can also be specified using the environment variable SECURAE_BACKUP_ID.")
	viper.BindPFlag("backup-id", downloadCmd.Flags().Lookup("backup-id"))
}

func fetchPresignedDownloadURL(url string, token string, data []byte) (string, error) {
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

func downloadFile(url, encryptionKeyB64Encoded, filename string) error {
	tr := &http.Transport{
		TLSHandshakeTimeout:   2 * time.Second,
		IdleConnTimeout:       2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	encryptionKeyMD5, _ := hashEncryptionKey(encryptionKeyB64Encoded)
	req.Header.Set("X-Amz-Server-Side-Encryption-Customer-Algorithm", "AES256")
	req.Header.Set("X-Amz-Server-Side-Encryption-Customer-Key", encryptionKeyB64Encoded)
	req.Header.Set("X-Amz-Server-Side-Encryption-Customer-Key-MD5", encryptionKeyMD5)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error downloading file: %s", resp.Status)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create the file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write content to file: %v", err)
	}

	return nil
}