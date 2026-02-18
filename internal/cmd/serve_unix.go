//go:build unix

package cmd

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr detaches the child from the parent session.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
