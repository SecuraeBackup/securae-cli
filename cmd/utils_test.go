package cmd

import (
	"net/http"
	"testing"
)

func TestCheckCLIVersionHeaders(t *testing.T) {
	tests := []struct {
		name                string
		minSupportedVersion string
		latestVersion       string
		currentVersion      string
		expectErr           bool
	}{
		{"CLI Outdated", "0.1.10", "0.1.10", "0.1.9", true},
		{"CLI Up-to-date", "0.1.10", "0.1.10", "0.1.10", false},
		{"CLI Version Ahead", "0.1.10", "0.1.11", "0.1.11", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testHeaders := http.Header{
				"X-Securae-Cli-Latest-Version":        {test.latestVersion},
				"X-Securae-Cli-Min-Supported-Version": {test.minSupportedVersion},
			}
			err := CheckCLIVersionHeaders(testHeaders, test.currentVersion)
			if test.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect error but got: %q", err.Error())
				}
			}
		})
	}
}

func TestCheckCLIVersionNoHeaders(t *testing.T) {
	currentVersion := "0.1.10"
	testHeaders := http.Header{}

	err := CheckCLIVersionHeaders(testHeaders, currentVersion)
	if err != nil {
		t.Errorf("Did not expect error but got: %q", err.Error())
	}
}
