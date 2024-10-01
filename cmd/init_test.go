/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"bytes"
	"os"
	"testing"
)

func TestInitCmdCreateConfigFile(t *testing.T) {
	// Set path for a temporary config file
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"

	// Check that config file doesn't exist
	_, err := os.Stat(configFile)
	if err == nil {
		t.Fatalf("File %s should not exist yet.", configFile)
	}

	// Call init command, it should create the config file
	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"--config", configFile, "init", "-t", "xxxxx"})
	rootCmd.Execute()

	// Check that confg file exists now
	_, err = os.Stat(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			t.Errorf("File %s should have been created", configFile)
		}
	}

}

func TestInitCmdDontOverrideConfigFile(t *testing.T) {
	// Create a temporary and empty config file
	tmpDir := t.TempDir()
	f, err := os.CreateTemp(tmpDir, "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}

	// Get config file modification time
	fileInfo, err := os.Stat(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	mTimeBefore := fileInfo.ModTime()

	// Call init command, using config file
	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"--config", f.Name(), "init", "-t", "xxxxx"})
	rootCmd.Execute()

	// Get config file modification time
	fileInfo, err = os.Stat(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	mTimeAfter := fileInfo.ModTime()

	// Check that confg file wasn't modified
	if mTimeBefore != mTimeAfter {
		t.Errorf("Existing file %s has been modified by the `init` command", f.Name())
	}
	defer os.Remove(f.Name())
}
