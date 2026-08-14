//go:build linux || darwin

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestPrepareCommandKillsDescendantProcessGroupOnCancellation(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	command := exec.CommandContext(ctx, "sh", "-c", `sleep 30 & echo $! > "$PID_FILE"; wait`)
	command.Env = append(os.Environ(), "PID_FILE="+pidPath)
	prepareCommand(command)
	started := time.Now()
	if err := command.Run(); err == nil {
		t.Fatal("cancelled process group returned no error")
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("process-group cancellation took %s", elapsed)
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		if processTerminated(pid) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("descendant process %d survived cancellation", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestProcessTerminatedTreatsZombieAsExited(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux exposes zombie state through procfs")
	}
	command := exec.Command("sh", "-c", "exit 0")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer command.Wait()
	statPath := filepath.Join("/proc", strconv.Itoa(command.Process.Pid), "stat")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(statPath)
		if err == nil && strings.Contains(string(data), ") Z ") {
			if !processTerminated(command.Process.Pid) {
				t.Fatal("zombie process was treated as running")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("child did not enter zombie state")
}

func TestRunningServerStopKillsDescendantThatIgnoresInterrupt(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "child.pid")
	command := exec.CommandContext(context.Background(), "sh", "-c", `trap 'exit 0' INT; (trap '' INT; sleep 30) & echo $! > "$PID_FILE"; wait`)
	command.Env = append(os.Environ(), "PID_FILE="+pidPath)
	prepareCommand(command)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	server := &runningServer{command: command, done: make(chan error, 1)}
	go func() { server.done <- command.Wait() }()
	var pid int
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(pidPath)
		if err == nil {
			pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("descendant PID was not recorded")
	}
	if err := server.stop(); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(time.Second)
	for {
		if processTerminated(pid) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("descendant process %d survived normal stop", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func processTerminated(pid int) bool {
	err := syscall.Kill(pid, 0)
	if errors.Is(err, syscall.ESRCH) {
		return true
	}
	if err != nil || runtime.GOOS != "linux" {
		return false
	}
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	if err != nil {
		return false
	}
	closingParen := strings.LastIndex(string(data), ") ")
	return closingParen >= 0 && len(data) > closingParen+2 && data[closingParen+2] == 'Z'
}

func TestRunningServerStopReturnsCachedErrorAfterWaitResultConsumed(t *testing.T) {
	command := exec.CommandContext(context.Background(), "sh", "-c", "exit 7")
	prepareCommand(command)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	server := &runningServer{command: command, done: make(chan error, 1)}
	go func() { server.done <- command.Wait() }()
	if err := server.stop(); err == nil {
		t.Fatal("nonzero server exit returned no error")
	}

	second := make(chan error, 1)
	go func() { second <- server.stop() }()
	select {
	case err := <-second:
		if err == nil {
			t.Fatal("cached nonzero server exit returned no error")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("second stop blocked after the server wait result was consumed")
	}
}
