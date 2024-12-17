/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var validateCmd = &cobra.Command{
	Use:   "validate [filename] [flags]",
	Short: "Validate backup files",
	Long: `Validate backup files verifying the encryption key and the integrity checksum.

If there is no filename argument, this command validates the latest file from the backup.
`,
	Example: `# using --backup-id
securae validate database-dump.tar.gz --backup-id=abcd1234-ab12-ab12-ab12-abcdef123456

# without specifying a filename it validates the latest uploaded file
securae validate --backup-id=abcd1234-ab12-ab12-ab12-abcdef123456

# validate a file using an environment variable
export SECURAE_BACKUP_ID=abcd1234-ab12-ab12-ab12-abcdef123456
securae validate database-dump.tar.gz`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MaximumNArgs(1)(cmd, args); err != nil {
			return fmt.Errorf("Only one filename must be specified.")
		}
		return nil
	},
	GroupID: "backup",
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

		postData := []byte(fmt.Sprintf(`{"include_checksum": true}`))
		if len(args) > 0 {
			filename := args[0]
			filenameOnly := filepath.Base(filename)
			postData = []byte(fmt.Sprintf(`{"filename": "%s", "include_checksum": true}`, filenameOnly))
		}

		metadataURL := fmt.Sprintf("%s/backups/%s/metadata/", apiURL, backupId)
		presignedURL, err := fetchPresignedURL(metadataURL, apiToken, postData)
		if err != nil {
			return err
		}

		parsedURL, _ := url.Parse(presignedURL)
		fileToDownload := filepath.Base(parsedURL.Path)
		cmd.Printf("[%s] Verifying encryption key... ", fileToDownload)
		checksumProvider, err := fetchChecksum(presignedURL, encryptionKeyB64Encoded)
		if err == nil {
			cmd.Printf("OK\n")
		} else {
			return err
		}

		preDownloadURL := fmt.Sprintf("%s/backups/%s/predownload/", apiURL, backupId)
		presignedURL, err = fetchPresignedURL(preDownloadURL, apiToken, postData)
		if err != nil {
			return err
		}

		tmpFile, err := os.CreateTemp("", "securae")
		if err != nil {
			return err
		}
		defer os.Remove(tmpFile.Name())

		cmd.Printf("[%s] Downloading file... ", fileToDownload)
		err = downloadFile(presignedURL, encryptionKeyB64Encoded, tmpFile.Name())
		if err == nil {
			cmd.Printf("OK\n")
		} else {
			return err
		}

		cmd.Printf("[%s] Calculating SHA-256 checksum... ", fileToDownload)
		checksum, err := ChecksumSHA256(tmpFile)
		if err == nil {
			cmd.Printf("OK\n")
		} else {
			return err
		}
		cmd.Printf("[%s] Verifying file integrity... ", fileToDownload)
		if checksum == checksumProvider {
			cmd.Printf("OK\n")
		} else if checksumProvider == "" {
			cmd.Printf("Error (file stored without checksum)\n")
		} else {
			cmd.Printf("Error\n")
		}

		return nil

	},
}

func init() {
	RootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringP(flagBackupId, flagShortBackupId, "", "A backup ID (`UUID` format) where your files were stored. It can also be specified using the environment variable SECURAE_BACKUP_ID.")
}

func fetchChecksum(url string, encryptionKeyB64Encoded string) (string, error) {
	tr := &http.Transport{
		TLSHandshakeTimeout:   5 * time.Second,
		IdleConnTimeout:       5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	encryptionKeyMD5, _ := hashEncryptionKey(encryptionKeyB64Encoded)
	req.Header.Set("X-Amz-Server-Side-Encryption-Customer-Algorithm", "AES256")
	req.Header.Set("X-Amz-Server-Side-Encryption-Customer-Key", encryptionKeyB64Encoded)
	req.Header.Set("X-Amz-Server-Side-Encryption-Customer-Key-MD5", encryptionKeyMD5)
	req.Header.Set("X-Amz-Checksum-Mode", "ENABLED")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			msg := "the encryption key used to upload the file does not match " +
				"the one used now.\nPlease, verify the value of " +
				"`encryption-key-b64encoded` in your configuration file."
			return "", fmt.Errorf(msg)
		}
		return "", fmt.Errorf("status code: %s", resp.Status)
	}

	checksum := resp.Header.Get("X-Amz-Checksum-Sha256")
	return checksum, nil
}
