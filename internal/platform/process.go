package platform

import (
	"os"
	"os/exec"
)

func StartDetached(executable string, args ...string) error {
	cmd := exec.Command(executable, args...)
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.Stdin = devNull
	if err := configureDetachedProcess(cmd); err != nil {
		_ = devNull.Close()
		return err
	}
	if err := cmd.Start(); err != nil {
		_ = devNull.Close()
		return err
	}
	if err := cmd.Process.Release(); err != nil {
		_ = devNull.Close()
		return err
	}
	return devNull.Close()
}

func CurrentExecutable() (string, error) {
	return os.Executable()
}
