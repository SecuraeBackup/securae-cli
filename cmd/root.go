/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const version = "0.1.0"

var cfgFile string
var apiEndpoint = "http://localhost:8000/api/v1"

var rootCmd = &cobra.Command{
	Use:     "securae",
	Version: version,
	Short:   "Securae Backup CLI",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/securae.yaml)")
}

func initConfig() {
	var configFilename = "securae.yaml"

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir, err := os.UserConfigDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName(configFilename)
		cfgFile = path.Join(configDir, configFilename)
	}

	viper.SetEnvPrefix("securae")
	viper.SetEnvKeyReplacer(strings.NewReplacer(`-`, `_`))
	viper.AutomaticEnv()
}
