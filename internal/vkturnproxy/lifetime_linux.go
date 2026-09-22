package vkturnproxy

import (
	"os/exec"
	"syscall"
)

// The kernel terminates the proxy if the panel exits without running StopAll.
func attachChildLifetime(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
}
