//go:build !windows

package runner

import "syscall"

func processAlive(pid int) bool { return syscall.Kill(pid, 0) == nil }
