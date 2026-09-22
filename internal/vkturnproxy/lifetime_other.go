//go:build !linux

package vkturnproxy

import "os/exec"

func attachChildLifetime(_ *exec.Cmd) {}
