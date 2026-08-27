//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

func configureCommand(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessTree(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}
