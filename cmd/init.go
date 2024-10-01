/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gopkg.in/yaml.v3"
)

type AppConfiguration struct {
	ApiToken                string `yaml:"api-token"`
	EncryptionKeyB64encoded string `yaml:"encryption-key-b64encoded"`
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Securae's configuration",
	Run: func(cmd *cobra.Command, args []string) {

		client := &http.Client{
			Timeout: 1000 * time.Millisecond,
		}

		req, err := http.NewRequest("GET", apiEndpoint+"/users/me", nil)
		if err != nil {
			panic(err)
		}
		req.Header.Set("Authorization", "Token "+viper.GetViper().GetString("api-token"))

		resp, err := client.Do(req)
		if err != nil {
			log.Println("Network connexion error.")
			log.Fatal(err)
		} else {
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				if resp.StatusCode == 401 {
					cmd.Println("Error: This API token seems to be wrong.")
				} else {
					cmd.Println("Error: The API service is unavaliable or its URL has changed.")
				}
			} else {
				cmd.Println("Write token to file")
				config := AppConfiguration{
					ApiToken:                viper.GetViper().GetString("api-token"),
					EncryptionKeyB64encoded: "",
				}
				yamlFile, err := yaml.Marshal(&config)
				if err != nil {
					panic(err)
				}

				err = ioutil.WriteFile(cfgFile, yamlFile, 0600)
				if err != nil {
					panic(err)
				}

			}

			key := make([]byte, 32)
			_, err := rand.Read(key)
			if err != nil {
				panic(err)
			}
			cmd.Println("Write encryption key to file")
			config := AppConfiguration{
				ApiToken:                viper.GetViper().GetString("api-token"),
				EncryptionKeyB64encoded: base64.StdEncoding.EncodeToString(key),
			}
			yamlFile, err := yaml.Marshal(&config)
			if err != nil {
				panic(err)
			}

			err = ioutil.WriteFile(cfgFile, yamlFile, 0600)
			if err != nil {
				panic(err)
			}

			cmd.Println("A new encryption key was generated.")
			cmd.Println("\nWARNING: please save this encryption key in a safe place:")
			cmd.Println("\n" + config.EncryptionKeyB64encoded + "\n")
			cmd.Println("You will need it to recover your files in case of disaster.")
		}

	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("api-token", "t", "", "Your API token")
	viper.BindPFlag("api-token", initCmd.Flags().Lookup("api-token"))
}
