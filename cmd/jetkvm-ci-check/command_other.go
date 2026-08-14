//go:build !unix

package main

import (
	"os/exec"
	"time"
)

const commandWaitDelay = 250 * time.Millisecond

func configureCommandTermination(command *exec.Cmd) {
	command.WaitDelay = commandWaitDelay
}
