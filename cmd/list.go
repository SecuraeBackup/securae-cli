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
	Short: "List backups or files in a backup",
	Long:  `List backups or files into a backup using its ID (UUID format).`,
	Example: `# list all the backups
securae list

# list files into a backup
securae list --backup-id=abcd1234-ab12-ab12-ab12-abcdef123456

# list files using an environment variable
export SECURAE_BACKUP_ID=abcd1234-ab12-ab12-ab12-abcdef123456
securae list`,
	Args:    cobra.NoArgs,
	GroupID: "backup",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag(flagBackupId, cmd.Flags().Lookup(flagBackupId))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		api := viper.GetString("api.url")
		token := viper.GetString("api.token")

		backupId := viper.GetString(flagBackupId)
		if backupId == "" {
			url := fmt.Sprintf("%s/backups", api)
			data, err := fetchBackups(url, token)
			if err != nil {
				return err
			}
			showBackups(data)
		} else {
			if !IsUUID(backupId) {
				return fmt.Errorf("Invalid Backup ID format.")
			}
			url := fmt.Sprintf("%s/backups/%s", api, backupId)
			data, err := fetchBackupData(url, token)
			if err != nil {
				return err
			}
			showBackupData(data, true)
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP(flagBackupId, flagShortBackupId, "", "A backup ID (`UUID` format) where your files will be stored. It can also be specified using the environment variable SECURAE_BACKUP_ID.")
}

func showBackupData(backup Backup, showMissing bool) {
	textBold := color.New(color.Bold, color.FgGreen).SprintFunc()
	textWait := color.New(color.Bold, color.FgYellow).SprintFunc()
	textUUID := color.New(color.Bold).SprintFunc()
	textTitle := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s (%s)\n", textBold(backup.Name), humanize.Bytes(backup.Size))
	fmt.Printf("Backup ID: %s\n", textUUID(backup.Id))
	var locations []string
	for _, bucket := range backup.Locations {
		location := fmt.Sprintf("%s, %s", bucket.City, strings.ToUpper(bucket.CountryCode))
		locations = append(locations, location)
	}
	if len(locations) > 0 {
		fmt.Printf("Storage location: %s\n", strings.Join(locations, " / "))
	}
	if len(backup.Backupobjects) > 0 {
		fmt.Printf("\n%s\n-------\n", textTitle("Objects"))
		for _, bo := range backup.Backupobjects {
			uploadDate, _ := time.ParseInLocation(time.RFC3339Nano, bo.CreatedAt, time.Local)
			if bo.Size > 0 {
				fmt.Printf("%s (%s)\n", textBold(bo.Name), humanize.Bytes(bo.Size))
			} else {
				fmt.Printf("%s (replicating...)\n", textWait(bo.Name))
			}
			fmt.Printf("└─ Object ID: %s uploaded on %s in %s, %s\n", textUUID(bo.Id), uploadDate.Format(time.RFC822Z), bo.Bucket.City, strings.ToUpper(bo.Bucket.CountryCode))
		}
	} else {
		if showMissing {
			fmt.Printf("\n%s\n-------\n", textTitle("Objects"))
			fmt.Printf("%s\n", textWait("No objects available in this bucket."))
		} else {
			fmt.Printf("\n")
		}
	}
}

func showBackups(backups []Backup) {
	for _, backup := range backups {
		showBackupData(backup, false)
	}
}

func fetchBackupData(url string, token string) (Backup, error) {
	tr := &http.Transport{
		TLSHandshakeTimeout:   5 * time.Second,
		IdleConnTimeout:       5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Backup{}, err
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return Backup{}, err
	}
	defer resp.Body.Close()

	err = CheckCLIVersionHeaders(resp.Header, version)
	if err != nil {
		return Backup{}, err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			parts := strings.Split(url, "/")
			backupId := parts[len(parts)-1]
			return Backup{}, fmt.Errorf("Backup ID %s not found on this account.", backupId)
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

func fetchBackups(url string, token string) ([]Backup, error) {
	tr := &http.Transport{
		TLSHandshakeTimeout:   5 * time.Second,
		IdleConnTimeout:       5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []Backup{}, err
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return []Backup{}, err
	}
	defer resp.Body.Close()

	err = CheckCLIVersionHeaders(resp.Header, version)
	if err != nil {
		return []Backup{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return []Backup{}, fmt.Errorf("Error fetching backup data: %s", resp.Status)
	}

	var backups = []Backup{}
	err = json.NewDecoder(resp.Body).Decode(&backups)
	if err != nil {
		return []Backup{}, err
	}
	return backups, nil
}
