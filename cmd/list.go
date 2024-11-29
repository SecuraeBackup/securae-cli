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
		viper.BindPFlag(flagBackupId, cmd.Flags().Lookup(flagBackupId))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		api := viper.GetString("api.url")
		token := viper.GetString("api.token")

		backupId := viper.GetString(flagBackupId)
		if backupId == "" {
			return fmt.Errorf("A Backup ID must be specified.")
		}
		if !IsUUID(backupId) {
			return fmt.Errorf("Invalid Backup ID format.")
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
	listCmd.Flags().StringP(flagBackupId, flagShortBackupId, "", "A backup ID (`UUID` format) where your files will be stored. It can also be specified using the environment variable SECURAE_BACKUP_ID.")
}

func showBackupData(backup Backup) {
	text_bold := color.New(color.Bold, color.FgGreen).SprintFunc()
	text_wait := color.New(color.Bold, color.FgYellow).SprintFunc()
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
	if len(backup.Backupobjects) == 0 {
		fmt.Printf("%s\n", text_wait("No objects available in this bucket."))
	}
	for _, bo := range backup.Backupobjects {
		uploadDate, _ := time.ParseInLocation(time.RFC3339Nano, bo.CreatedAt, time.Local)
		if bo.Size > 0 {
			fmt.Printf("%s (%s)\n", text_bold(bo.Name), humanize.Bytes(bo.Size))
		} else {
			fmt.Printf("%s (replicating...)\n", text_wait(bo.Name))
		}
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
			return Backup{}, fmt.Errorf("Backup ID %s not found on this account.", backup_id)
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
