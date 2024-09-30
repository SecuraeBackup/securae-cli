/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gopkg.in/yaml.v3"
)

const apiEndpoint = "http://localhost:8000/api/v1"

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
			log.Println("Response status:", resp.Status)

			scanner := bufio.NewScanner(resp.Body)
			for i := 0; scanner.Scan() && i < 5; i++ {
				log.Println(scanner.Text())
			}

			if err := scanner.Err(); err != nil {
				panic(err)
			}

			if resp.StatusCode != 200 {
				if resp.StatusCode == 401 {
					fmt.Println("Error: You're using an invalid API token.")
				} else {
					fmt.Println("Error: The API service is unavaliable or its URL has changed.")
				}
			} else {
				fmt.Println("Write token to file")
				config := AppConfiguration{
					ApiToken:                viper.GetViper().GetString("api-token"),
					EncryptionKeyB64encoded: "",
				}
				yamlFile, err := yaml.Marshal(&config)
				if err != nil {
					panic(err)
				}

				configFileName := "/home/pabluk/.config/securae.yaml"
				err = ioutil.WriteFile(configFileName, yamlFile, 0600)
				if err != nil {
					panic(err)
				}

			}

			key := make([]byte, 32)
			_, err := rand.Read(key)
			if err != nil {
				panic(err)
			}
			fmt.Println("Write encryption key to file")
			config := AppConfiguration{
				ApiToken:                viper.GetViper().GetString("api-token"),
				EncryptionKeyB64encoded: base64.StdEncoding.EncodeToString(key),
			}
			yamlFile, err := yaml.Marshal(&config)
			if err != nil {
				panic(err)
			}

			configFileName := "/home/pabluk/.config/securae.yaml"
			err = ioutil.WriteFile(configFileName, yamlFile, 0600)
			if err != nil {
				panic(err)
			}

			fmt.Println("A new encryption key was generated.\n")
			fmt.Println("WARNING: please save this encryption key in a safe place:")
			fmt.Println("\n" + config.EncryptionKeyB64encoded + "\n")
			fmt.Println("You will need it to recover your files in case of disaster.")
		}

	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("api-token", "t", "", "Your API token")
	viper.BindPFlag("api-token", initCmd.Flags().Lookup("api-token"))
}
