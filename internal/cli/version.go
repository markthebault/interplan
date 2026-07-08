package cli

import (
	"fmt"
	"io"
)

type VersionInfo struct {
	Version string
	Commit  string
	Date    string
}

var currentVersion = VersionInfo{Version: "dev", Commit: "unknown", Date: "unknown"}

func SetVersion(info VersionInfo) {
	if info.Version == "" {
		info.Version = "dev"
	}
	if info.Commit == "" {
		info.Commit = "unknown"
	}
	if info.Date == "" {
		info.Date = "unknown"
	}
	currentVersion = info
}

func writeVersion(stdout io.Writer) {
	fmt.Fprintf(stdout, "interplan %s\n", currentVersion.Version)
	if currentVersion.Commit != "unknown" {
		fmt.Fprintf(stdout, "commit %s\n", currentVersion.Commit)
	}
	if currentVersion.Date != "unknown" {
		fmt.Fprintf(stdout, "built %s\n", currentVersion.Date)
	}
}
