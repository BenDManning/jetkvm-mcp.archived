//go:build linux

package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	if err := unix.Renameat2(unix.AT_FDCWD, path, unix.AT_FDCWD, alternate, unix.RENAME_EXCHANGE); err != nil {
		if errors.Is(err, unix.ENOSYS) || errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.EINVAL) {
			t.Skipf("RENAME_EXCHANGE unsupported: %v", err)
		}
		t.Fatalf("initial RENAME_EXCHANGE failed: %v", err)
	}

	stop := make(chan struct{})
	var wait sync.WaitGroup
	var stopOnce sync.Once
	var exchanges atomic.Int64
	var exchangeErr atomic.Value
	exchanges.Store(1)
	wait.Add(1)
	go func() {
		defer wait.Done()
		for {
			select {
			case <-stop:
				return
			default:
				if err := unix.Renameat2(unix.AT_FDCWD, path, unix.AT_FDCWD, alternate, unix.RENAME_EXCHANGE); err != nil {
					exchangeErr.Store(err)
					return
				}
				exchanges.Add(1)
				runtime.Gosched()
			}
		}
	}()
	stopExchanges := func() {
		stopOnce.Do(func() {
			close(stop)
			wait.Wait()
		})
	}
	defer stopExchanges()

	for range 10_000 {
		if _, err := readPlan(path); err == nil {
			t.Fatal("readPlan accepted the valid target of a raced symlink")
		}
	}
	stopExchanges()
	if err, ok := exchangeErr.Load().(error); ok {
		t.Fatalf("RENAME_EXCHANGE failed during race: %v", err)
	}
	if exchanges.Load() < 2 {
		t.Fatalf("symlink exchange did not race with reads; exchanges = %d", exchanges.Load())
	}
}

func TestReadPlanRejectsFIFOWithoutBlocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.fifo")
	if err := unix.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := readPlan(path)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("readPlan accepted a FIFO")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("readPlan blocked while opening a FIFO")
	}
}
