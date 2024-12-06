/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:     "init [flags]",
	Short:   "Initialize Securae's configuration",
	Long:    `Validate your API token, generate an encryption key, and store all this information in a configuration file.`,
	Example: `securae init --api-token xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`,
	Args:    cobra.NoArgs,
	GroupID: "setup",
	RunE: func(cmd *cobra.Command, args []string) error {

		client := &http.Client{
			Timeout: 1000 * time.Millisecond,
		}

		api := viper.GetString("api.url")
		token := viper.GetString("api.token")
		req, err := http.NewRequest("GET", api+"/users/me", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Token "+token)
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			return errors.Join(err, fmt.Errorf("Please verify that %s is reachable from this device.", apiEndpoint))
		} else {
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if resp.StatusCode == http.StatusUnauthorized {
					return fmt.Errorf("There was an authentication issue, please check the API token in the configuration.")
				} else {
					return fmt.Errorf("The API service is unavailable. Please, try again in a few minutes.")
				}
			} else {
				viper.WriteConfig()
			}

			if viper.GetString("encryption-key-b64encoded") == "" {
				key := make([]byte, 32)
				_, err := rand.Read(key)
				if err != nil {
					return err
				}
				keyEncoded := base64.StdEncoding.EncodeToString(key)
				viper.Set("encryption-key-b64encoded", keyEncoded)
				viper.WriteConfig()

				cmd.Println("A new encryption key was generated:")
				cmd.Println("\n" + keyEncoded + "\n")
				cmd.Println("WARNING: Please save this encryption key in a safe place. You will need it to test your backups or to recover your files in case of disaster.")
			}

			// Because using `AutomaticEnv()` the env var `SECURAE_BACKUP_ID` is
			// saved to the config file but it's not needed so it must be removed.
			configFileName := viper.ConfigFileUsed()
			if err := removeYAMLKey(configFileName, flagBackupId); err != nil {
				return err
			}
		}
		return nil

	},
}

func init() {
	RootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP(flagApiToken, flagShortApiToken, "", "Your API token")
	viper.BindPFlag("api.token", initCmd.Flags().Lookup(flagApiToken))
}

func removeYAMLKey(filename string, key string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	var yamlContent map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlContent); err != nil {
		return fmt.Errorf("error parsing YAML: %w", err)
	}

	if _, exists := yamlContent[key]; exists {
		delete(yamlContent, key)
	} else {
		return nil
	}
	updatedData, err := yaml.Marshal(yamlContent)
	if err != nil {
		return fmt.Errorf("error encoding updated YAML: %w", err)
	}

	if err := os.WriteFile(filename, updatedData, 0644); err != nil {
		return fmt.Errorf("error writing updated file: %w", err)
	}
	return nil
}

func IsUUID(s string) bool {
	// Regular expression for UUID v4
	uuidV4Regex := `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[89abAB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`
	re := regexp.MustCompile(uuidV4Regex)

	return re.MatchString(s)
}
