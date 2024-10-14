/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Backup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Locations []struct {
		Provider    string `json:"provider"`
		Region      string `json:"region"`
		CountryCode string `json:"country_code"`
		City        string `json:"city"`
	} `json:"locations"`
	Backupobjects []string `json:"backupobjects"`
}

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List files in a backup",
	Long: `List files into the backup ID (UUID format) defined in the web UI.
For example:

  securae list --backup-id=abcd1234-ab12-ab12-ab12-abcdef123456

Or you can also use an environment variable:

  export SECURAE_BACKUP_ID=abcd1234-ab12-ab12-ab12-abcdef123456
  securae list

`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		api := viper.GetString("api.url")
		token := viper.GetString("api.token")
		backupId := viper.GetString("backup-id")
		if backupId == "" {
			fmt.Errorf("A Backup ID must be specified.")
		}
		url := fmt.Sprintf("%s/backups/%s", api, backupId)
		files, err := fetchFiles(url, token)
		if err != nil {
			fmt.Println(err)
		}
		for _, f := range files {
			cmd.Println(f)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("backup-id", "b", "", "A backup ID (`UUID` format) where your files will be stored. It can also be specified using the environment variable SECURAE_BACKUP_ID.")
}

func fetchFiles(url string, token string) ([]string, error) {
	tr := &http.Transport{
		TLSHandshakeTimeout:   2 * time.Second,
		IdleConnTimeout:       2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []string{}, err
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{}, fmt.Errorf("Error fetching files: %s", resp.Status)
	}

	var backup = Backup{}
	err = json.NewDecoder(resp.Body).Decode(&backup)
	if err != nil {
		return []string{}, err
	}
	files := []string{}
	for _, m := range backup.Backupobjects {
		files = append(files, m)
	}
	return files, nil
}
