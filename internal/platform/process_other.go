//go:build !darwin && !linux && !windows

package platform

import "os/exec"

func configureDetachedProcess(cmd *exec.Cmd) error {
	return nil
}
