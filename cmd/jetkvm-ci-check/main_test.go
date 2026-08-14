package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestFormatGateAcceptsFormattedAndRejectsUnformattedSource(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package fixture\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), []string{"--root", root, "format"}); err != nil {
		t.Fatalf("formatted fixture rejected: %v", err)
	}
	if err := os.WriteFile(path, []byte("package fixture\nfunc main(){println(\"intentional failure\")}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), []string{"--root", root, "format"}); err == nil {
		t.Fatal("intentional unformatted fixture passed")
	}
}

func TestTidyGateAcceptsTidyAndRejectsStaleModule(t *testing.T) {
	root := t.TempDir()
	goMod := filepath.Join(root, "go.mod")
	if err := os.WriteFile(goMod, []byte("module fixture.invalid/ci-check\n\ngo 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), []string{"--root", root, "tidy"}); err != nil {
		t.Fatalf("tidy fixture rejected: %v", err)
	}
	if err := os.WriteFile(goMod, []byte("module fixture.invalid/ci-check\n\ngo 1.25.0\n\nrequire rsc.io/quote v1.5.2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), []string{"--root", root, "tidy"}); err == nil {
		t.Fatal("intentional stale module fixture passed")
	}
}

func TestTidyGateRejectsSymlinkedModuleFiles(t *testing.T) {
	for _, name := range []string{"go.mod", "go.sum"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			outside := filepath.Join(t.TempDir(), "outside")
			outsideContent := []byte("module fixture.invalid/outside\n\ngo 1.25.0\n")
			if name == "go.sum" {
				outsideContent = nil
			}
			if err := os.WriteFile(outside, outsideContent, 0o600); err != nil {
				t.Fatal(err)
			}
			if name == "go.sum" {
				if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture.invalid/ci-check\n\ngo 1.25.0\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(outside, filepath.Join(root, name)); err != nil {
				t.Fatal(err)
			}
			if err := run(context.Background(), []string{"--root", root, "tidy"}); err == nil {
				t.Fatalf("tidy accepted symlinked %s", name)
			}
		})
	}
}

func TestTidyGateTimeoutTerminatesDescendants(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group termination is Unix-specific")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture.invalid/timeout\n\ngo 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	shimDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "descendant-ran")
	shim := fmt.Sprintf("#!/bin/sh\n(sleep 1; printf leaked > %q) &\nsleep 30\n", marker)
	if err := os.WriteFile(filepath.Join(shimDir, "go"), []byte(shim), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := run(ctx, []string{"--root", root, "tidy"}); err == nil {
		t.Fatal("timed-out tidy command succeeded")
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("timed-out tidy descendant survived: %v", err)
	}
}

func TestCICheckRejectsUnknownModeAndNonDirectoryRoot(t *testing.T) {
	if err := run(context.Background(), []string{"unknown"}); err == nil {
		t.Fatal("unknown mode accepted")
	}
	if err := run(context.Background(), []string{"--root", filepath.Join(t.TempDir(), "missing"), "format"}); err == nil {
		t.Fatal("missing root accepted")
	}
}
