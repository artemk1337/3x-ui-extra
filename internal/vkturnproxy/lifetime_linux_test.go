package vkturnproxy

import (
	"os/exec"
	"syscall"
	"testing"
)

func TestAttachChildLifetime(t *testing.T) {
	cmd := exec.Command("/bin/true")
	attachChildLifetime(cmd)
	if cmd.SysProcAttr == nil || cmd.SysProcAttr.Pdeathsig != syscall.SIGTERM {
		t.Fatalf("child parent-death signal = %+v", cmd.SysProcAttr)
	}
}
