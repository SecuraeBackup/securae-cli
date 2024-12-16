package cmd

import (
	"net/http"
	"os"
	"testing"
)

func TestChecksumSHA256(t *testing.T) {
	testContent := "Securae Backup"
	// Generated using:
	// $ echo -n "Securae Backup" | openssl dgst -binary -sha256 | base64
	expectedChecksum := "sksu4ehJX6Lp+89fjDr+l2j6vxQ2ZR82wZ2UL/LPwlU="

	tempFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = tempFile.WriteString(testContent)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek to beginning of temp file: %v", err)
	}

	checksum, err := ChecksumSHA256(tempFile)
	if err != nil {
		t.Fatalf("Error calculating checksum: %v", err)
	}

	if checksum != expectedChecksum {
		t.Errorf("Checksum mismatch: got %s, want %s", checksum, expectedChecksum)
	}
}

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
