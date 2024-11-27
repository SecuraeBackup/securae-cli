/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"io/fs"
	"log"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const version = "0.1.6"
const apiEndpoint = "https://dashboard.securaebackup.com/api/v1"

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
