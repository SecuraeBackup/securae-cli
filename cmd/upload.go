/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
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

		filename := args[0]
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		url := fmt.Sprintf("%s/backups/%s/preupload/", apiURL, backupId)
		filenameOnly := filepath.Base(filename)
		fi, _ := file.Stat()
		presignedURL, err := fetchPresignedURL(url, apiToken, []byte(fmt.Sprintf(`{"filename": "%s", "size": %d}`, filenameOnly, fi.Size())))
		if err != nil {
			return err
		}

		cmd.Printf("Uploading file %s... ", filename)
		err = uploadFile(presignedURL, encryptionKeyB64Encoded, file)
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
	uploadCmd.Flags().StringP(flagBackupId, flagShortBackupId, "", "A backup ID (`UUID` format) where your files will be stored. It can also be specified using the environment variable SECURAE_BACKUP_ID.")
}

func uploadFile(url string, encryptionKeyB64Encoded string, file *os.File) error {
	buffer := bytes.NewBuffer(nil)
	if _, err := io.Copy(buffer, file); err != nil {
		return err
	}

	tr := &http.Transport{
		TLSHandshakeTimeout:   5 * time.Second,
		IdleConnTimeout:       5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	client := &http.Client{Transport: tr}
	request, err := http.NewRequest(http.MethodPut, url, buffer)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "multipart/form-data")

	encryptionKeyMD5, _ := hashEncryptionKey(encryptionKeyB64Encoded)
	request.Header.Set("X-Amz-Server-Side-Encryption-Customer-Algorithm", "AES256")
	request.Header.Set("X-Amz-Server-Side-Encryption-Customer-Key", encryptionKeyB64Encoded)
	request.Header.Set("X-Amz-Server-Side-Encryption-Customer-Key-MD5", encryptionKeyMD5)

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

func hashEncryptionKey(encryptionKeyB64Encoded string) (string, error) {
	if encryptionKeyB64Encoded == "" {
		return "", fmt.Errorf("There's no encryption key to hash")
	}
	encryptionKey, err := base64.StdEncoding.DecodeString(encryptionKeyB64Encoded)
	if err != nil {
		return "", fmt.Errorf("error decoding base64 key: %v", err)
	}

	hash := md5.Sum(encryptionKey)
	hashBase64 := base64.StdEncoding.EncodeToString(hash[:])

	return hashBase64, nil

}
