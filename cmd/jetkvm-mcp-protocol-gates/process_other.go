//go:build !linux && !darwin

package main

import (
	"os"
	"os/exec"
	"time"
)

func prepareCommand(command *exec.Cmd) {
	command.WaitDelay = 5 * time.Second
}

func interruptCommand(command *exec.Cmd) error {
	if command.Process == nil {
		return os.ErrProcessDone
	}
	return command.Process.Signal(os.Interrupt)
}

func killCommand(command *exec.Cmd) error {
	if command.Process == nil {
		return os.ErrProcessDone
	}
	return command.Process.Kill()
}
