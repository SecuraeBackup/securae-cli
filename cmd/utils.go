package cmd

import (
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/mod/semver"
)

func CheckCLIVersionHeaders(headers http.Header, ownVersion string) error {
	var latestVersion, minSupportedVersion string

	minSupportedVersionHeaders, ok := headers["X-Securae-Cli-Min-Supported-Version"]
	if !ok {
		// Skip check when there's no header
		return nil
	}
	for _, value := range minSupportedVersionHeaders {
		minSupportedVersion = value
	}

	latestVersionHeaders, ok := headers["X-Securae-Cli-Latest-Version"]
	for _, value := range latestVersionHeaders {
		latestVersion = value
	}

	cmp := semver.Compare("v"+ownVersion, "v"+minSupportedVersion)
	if cmp < 0 {
		msg := fmt.Sprintf("Your CLI version (%s) is outdated.\n"+
			"A newer version (%s) is available. Please update to the latest version to ensure optimal performance and compatibility.\n"+
			"For update instructions, visit: https://docs.securaebackup.com/", ownVersion, latestVersion)
		return errors.New(msg)
	}

	return nil
}
