/*
Copyright © 2024 Securae Backup
*/
package cmd

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/spf13/viper"
)

// Mock API Server
func mockAPIServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	return server
}

func TestInitCmdCreateConfigFile(t *testing.T) {
	server := mockAPIServer()
	defer server.Close()

	viper.Reset()
	viper.Set("api.url", server.URL)

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
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"--config", configFile, "init", "-t", "xxxxx"})
	RootCmd.Execute()

	// Check that confg file exists now
	_, err = os.Stat(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			t.Errorf("File %s should have been created", configFile)
		}
	}
}

func TestInitCmdDontOverrideEncryptionKey(t *testing.T) {
	server := mockAPIServer()
	defer server.Close()

	viper.Reset()
	viper.Set("api.url", server.URL)

	// Create a temporary and empty config file
	tmpDir := t.TempDir()
	f, err := os.CreateTemp(tmpDir, "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	key := "encryption-key-b64encoded: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx=\n"
	_, err = f.WriteString(key)
	if err != nil {
		t.Fatal(err)
	}

	// Call init command, using config file
	actual := new(bytes.Buffer)
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"--config", f.Name(), "init", "-t", "xxxxx"})
	RootCmd.Execute()

	// Check that encryption key is still there
	b, err := ioutil.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	keyExist, err := regexp.Match(key, b)
	if err != nil {
		t.Fatal(err)
	}
	if !keyExist {
		t.Errorf("Existent encryption key was modified by `init` command")
	}
	defer os.Remove(f.Name())
}
