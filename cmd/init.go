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
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var initCmd = &cobra.Command{
	Use:   "init [flags]",
	Short: "Initialize Securae's configuration",
	Args:  cobra.NoArgs,
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

		resp, err := client.Do(req)
		if err != nil {
			return errors.Join(err, fmt.Errorf("Please verify that %s is reachable from this device.", apiEndpoint))
		} else {
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				if resp.StatusCode == 401 {
					return fmt.Errorf("There was an authentication issue, please check the API token in the configuration.")
				} else {
					return fmt.Errorf("The API service is unavailable. Please, try again in a few minutes.")
				}
			} else {
				viper.WriteConfig()
				if err != nil {
					return err
				}
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
				if err != nil {
					return err
				}

				cmd.Println("A new encryption key was generated:")
				cmd.Println("\n" + keyEncoded + "\n")
				cmd.Println("WARNING: Please save this encryption key in a safe place. You will need it to test your backups or to recover your files in case of disaster.")
			}
		}
		return nil

	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("api-token", "t", "", "Your API token")
	viper.BindPFlag("api.token", initCmd.Flags().Lookup("api-token"))
}
