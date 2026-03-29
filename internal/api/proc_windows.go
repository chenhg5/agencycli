//go:build windows

package api

import (
	"fmt"
	"os/exec"
)

func setProcGroup(cmd *exec.Cmd) {}

func killProcessGroup(pid int) {
	_ = exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(pid)).Run()
}
