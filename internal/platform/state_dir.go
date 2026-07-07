package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

func StateFile() (string, error) {
	var dir string
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, "Library", "Application Support", "interplan")
	case "windows":
		dir = os.Getenv("LocalAppData")
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			dir = filepath.Join(home, "AppData", "Local")
		}
		dir = filepath.Join(dir, "interplan")
	default:
		dir = os.Getenv("XDG_STATE_HOME")
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			dir = filepath.Join(home, ".local", "state")
		}
		dir = filepath.Join(dir, "interplan")
	}
	return filepath.Join(dir, "state.json"), nil
}
