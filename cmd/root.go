/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const version = "0.1.8"
const apiEndpoint = "https://dashboard.securaebackup.com/api/v1"

const flagApiToken = "api-token"
const flagShortApiToken = "t"
const flagBackupId = "backup-id"
const flagShortBackupId = "b"

var cfgFile string

var rootCmd = &cobra.Command{
	Use:          "securae",
	Version:      version,
	Short:        "Securae Backup CLI",
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/securae.yaml)")
}

func initConfig() {
	var configFilename = "securae.yaml"

	viper.SetDefault("api.url", apiEndpoint)

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir, err := os.UserConfigDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName(configFilename)
		cfgFile = path.Join(configDir, configFilename)
		viper.SetConfigFile(cfgFile)
	}

	viper.SetEnvPrefix("securae")
	viper.SetEnvKeyReplacer(strings.NewReplacer(`-`, `_`))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(*fs.PathError); ok {
			// We can ignore, it will be written by `init` command.
		} else {
			log.Fatal(err)
		}
	}
}

func getBackupId() (string, error) {
	backupId := viper.GetString(flagBackupId)
	if backupId == "" {
		return "", fmt.Errorf("A Backup ID must be specified.")
	}
	if !IsUUID(backupId) {
		return "", fmt.Errorf("Invalid Backup ID format.")
	}
	return backupId, nil
}

func getEncryptionKey() (string, error) {
	encryptionKeyB64Encoded := viper.GetString("encryption-key-b64encoded")
	if encryptionKeyB64Encoded == "" {
		return "", fmt.Errorf("An encryption key is mandatory.")
	}
	return encryptionKeyB64Encoded, nil
}

func fetchPresignedURL(url string, token string, data []byte) (string, error) {
	tr := &http.Transport{
		TLSHandshakeTimeout:   5 * time.Second,
		IdleConnTimeout:       5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var objmap map[string]interface{}
	if err := json.Unmarshal(body, &objmap); err != nil {
		objmap = make(map[string]interface{})
	}

	if resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == http.StatusPaymentRequired {
			return "", fmt.Errorf(objmap["error"].(string))
		}
		return "", fmt.Errorf("Fetching presigned URL: %s", resp.Status)
	}

	return objmap["url"].(string), nil
}
