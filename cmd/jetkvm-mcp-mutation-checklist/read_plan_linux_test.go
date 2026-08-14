//go:build linux

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"golang.org/x/sys/unix"
)

func TestReadPlanRejectsSymlinkExchangeRace(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "plan.json")
	alternate := filepath.Join(directory, "alternate")
	validPath := filepath.Join(directory, "valid.json")
	valid, err := json.Marshal(completePlan())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(validPath, valid, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(validPath, alternate); err != nil {
		t.Fatal(err)
	}

	stop := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = unix.Renameat2(unix.AT_FDCWD, path, unix.AT_FDCWD, alternate, unix.RENAME_EXCHANGE)
				runtime.Gosched()
			}
		}
	}()
	defer func() {
		close(stop)
		wait.Wait()
	}()

	for range 10_000 {
		if _, err := readPlan(path); err == nil {
			t.Fatal("readPlan accepted the valid target of a raced symlink")
		}
	}
}
