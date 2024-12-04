/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var downloadCmd = &cobra.Command{
	Use:   "download [filename] [flags]",
	Short: "Download backup files",
	Long: `Download files using a backup ID (UUID format), as defined in the web UI.

If there is no filename argument, this command downloads the latest file from the backup.
`,
	Example: `# using --backup-id
securae download database-dump.tar.gz --backup-id=abcd1234-ab12-ab12-ab12-abcdef123456

# without specifying a filename it downloads the latest uploaded file
securae download --backup-id=abcd1234-ab12-ab12-ab12-abcdef123456

# download a file using an environment variable
export SECURAE_BACKUP_ID=abcd1234-ab12-ab12-ab12-abcdef123456
securae download database-dump.tar.gz`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MaximumNArgs(1)(cmd, args); err != nil {
			return fmt.Errorf("Only one filename must be specified.")
		}
		return nil
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag(flagBackupId, cmd.Flags().Lookup(flagBackupId))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := viper.GetString("api.url")
		apiToken := viper.GetString("api.token")

		backupId, err := getBackupId()
		if err != nil {
			return err
		}

		encryptionKeyB64Encoded, err := getEncryptionKey()
		if err != nil {
			return err
		}

		postData := []byte(fmt.Sprintf(`{}`))
		if len(args) > 0 {
			filename := args[0]
			filenameOnly := filepath.Base(filename)
			postData = []byte(fmt.Sprintf(`{"filename": "%s"}`, filenameOnly))
		}

		preDownloadURL := fmt.Sprintf("%s/backups/%s/predownload/", apiURL, backupId)
		presignedURL, err := fetchPresignedURL(preDownloadURL, apiToken, postData)
		if err != nil {
			return err
		}

		parsedURL, _ := url.Parse(presignedURL)
		fileToDownload := filepath.Base(parsedURL.Path)
		cmd.Printf("Downloading file %s... ", fileToDownload)
		err = downloadFile(presignedURL, encryptionKeyB64Encoded, fileToDownload)
		if err == nil {
			cmd.Printf("OK\n")
		} else {
			return err
		}
		return nil

	},
}

func init() {
	RootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().StringP(flagBackupId, flagShortBackupId, "", "A backup ID (`UUID` format) where your files will be stored. It can also be specified using the environment variable SECURAE_BACKUP_ID.")
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
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), "must provide the correct secret key") {
			msg := "the encryption key used to download the file does not match " +
				"the one used to upload it.\nPlease, verify the value of " +
				"`encryption-key-b64encoded` in your configuration file."
			return fmt.Errorf(msg)
		}
		return fmt.Errorf("status code: %s", resp.Status)
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
