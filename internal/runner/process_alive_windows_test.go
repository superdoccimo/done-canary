//go:build windows

package runner

import (
	"os/exec"
	"strconv"
	"strings"
)

func processAlive(pid int) bool {
	output, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/FO", "CSV", "/NH").CombinedOutput()
	if err != nil {
		return false
	}
	text := strings.ToLower(string(output))
	return strings.Contains(text, `"`+strconv.Itoa(pid)+`"`) && !strings.Contains(text, "no tasks")
}
