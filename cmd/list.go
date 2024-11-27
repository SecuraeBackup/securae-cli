/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Backup struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Size      uint64 `json:"size"`
	Locations []struct {
		Region      string `json:"region"`
		CountryCode string `json:"country_code"`
		City        string `json:"city"`
	} `json:"locations"`
	Backupobjects []struct {
		Id     string `json:"id"`
		Name   string `json:"name"`
		Bucket struct {
			Region      string `json:"region"`
			CountryCode string `json:"country_code"`
			City        string `json:"city"`
		} `json:"bucket"`
		Size      uint64 `json:"size"`
		CreatedAt string `json:"created_at"`
	} `json:"backupobjects"`
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
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("backup-id", cmd.Flags().Lookup("backup-id"))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		api := viper.GetString("api.url")
		token := viper.GetString("api.token")

		backupId := viper.GetString("backup-id")
		if backupId == "" {
			return fmt.Errorf("A Backup ID must be specified.")
		}
		url := fmt.Sprintf("%s/backups/%s", api, backupId)
		data, err := fetchBackupData(url, token)
		if err != nil {
			return err
		}
		showBackupData(data)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("backup-id", "b", "", "A backup ID (`UUID` format) where your files will be stored. It can also be specified using the environment variable SECURAE_BACKUP_ID.")
}

func showBackupData(backup Backup) {
	text_bold := color.New(color.Bold, color.FgGreen).SprintFunc()
	text_uuid := color.New(color.Bold).SprintFunc()
	text_title := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s (%s)\n", text_bold(backup.Name), humanize.Bytes(backup.Size))
	fmt.Printf("Backup ID: %s\n", text_uuid(backup.Id))
	var locations []string
	for _, bucket := range backup.Locations {
		location := fmt.Sprintf("%s, %s", bucket.City, strings.ToUpper(bucket.CountryCode))
		locations = append(locations, location)
	}
	fmt.Printf("Storage location: %s\n", strings.Join(locations, " / "))
	fmt.Printf("\n%s\n-------\n", text_title("Objects"))
	for _, bo := range backup.Backupobjects {
		uploadDate, _ := time.ParseInLocation(time.RFC3339Nano, bo.CreatedAt, time.Local)
		fmt.Printf("%s (%s)\n", text_bold(bo.Name), humanize.Bytes(bo.Size))
		fmt.Printf("└─ Object ID: %s uploaded on %s in %s, %s\n", text_uuid(bo.Id), uploadDate.Format(time.RFC822Z), bo.Bucket.City, strings.ToUpper(bo.Bucket.CountryCode))
	}
}

func fetchBackupData(url string, token string) (Backup, error) {
	tr := &http.Transport{
		TLSHandshakeTimeout:   2 * time.Second,
		IdleConnTimeout:       2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Backup{}, err
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return Backup{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			parts := strings.Split(url, "/")
			backup_id := parts[len(parts)-1]
			return Backup{}, fmt.Errorf("Backup ID %s not found in this account or there are no objects available yet", backup_id)
		}
		return Backup{}, fmt.Errorf("Error fetching backup data: %s", resp.Status)
	}

	var backup = Backup{}
	err = json.NewDecoder(resp.Body).Decode(&backup)
	if err != nil {
		return Backup{}, err
	}
	return backup, nil
}
