//go:build linux || darwin

package main

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func prepareCommand(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.WaitDelay = 5 * time.Second
	command.Cancel = func() error {
		return signalCommandGroup(command, syscall.SIGKILL)
	}
}

func interruptCommand(command *exec.Cmd) error {
	return signalCommandGroup(command, syscall.SIGINT)
}

func killCommand(command *exec.Cmd) error {
	return signalCommandGroup(command, syscall.SIGKILL)
}

func signalCommandGroup(command *exec.Cmd, signal syscall.Signal) error {
	if command.Process == nil {
		return os.ErrProcessDone
	}
	if err := syscall.Kill(-command.Process.Pid, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
