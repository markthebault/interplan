package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersionPrintsBuildMetadata(t *testing.T) {
	t.Cleanup(func() {
		SetVersion(VersionInfo{})
	})
	SetVersion(VersionInfo{
		Version: "v1.2.3",
		Commit:  "abc123def456",
		Date:    "2026-07-08T11:22:33Z",
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"--version"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run --version: %v", err)
	}
	out := stdout.String()
	for _, required := range []string{"interplan v1.2.3", "commit abc123def456", "built 2026-07-08T11:22:33Z"} {
		if !strings.Contains(out, required) {
			t.Fatalf("version output missing %q: %s", required, out)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunVersionDefaultsToDev(t *testing.T) {
	t.Cleanup(func() {
		SetVersion(VersionInfo{})
	})
	SetVersion(VersionInfo{})

	var stdout bytes.Buffer
	if err := Run([]string{"version"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run version: %v", err)
	}
	if got := stdout.String(); got != "interplan dev\n" {
		t.Fatalf("version output = %q", got)
	}
}
