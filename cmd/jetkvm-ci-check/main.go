package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxGoSourceBytes = 4 << 20
	tidyTimeout      = 2 * time.Minute
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("jetkvm-ci-check", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "repository root")
	if err := flags.Parse(args); err != nil || flags.NArg() != 1 {
		return errors.New("usage: jetkvm-ci-check [--root DIR] format|tidy")
	}
	info, err := os.Lstat(*root)
	if err != nil || !info.IsDir() {
		return errors.New("repository root is not a directory")
	}
	switch flags.Arg(0) {
	case "format":
		return checkFormat(*root)
	case "tidy":
		return checkTidy(ctx, *root)
	default:
		return errors.New("usage: jetkvm-ci-check [--root DIR] format|tidy")
	}
}

func checkFormat(root string) error {
	unformatted := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("go source symlinks are not supported")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxGoSourceBytes {
			return errors.New("go source exceeds CI checker size bound")
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		formatted, err := format.Source(source)
		if err != nil || string(formatted) != string(source) {
			unformatted++
		}
		return nil
	})
	if err != nil {
		return errors.New("unable to inspect Go formatting")
	}
	if unformatted != 0 {
		return fmt.Errorf("%d Go source file(s) require gofmt", unformatted)
	}
	return nil
}

func checkTidy(ctx context.Context, root string) error {
	if err := requireRegularFile(filepath.Join(root, "go.mod"), true); err != nil {
		return errors.New("go.mod must be a regular file")
	}
	if err := requireRegularFile(filepath.Join(root, "go.sum"), false); err != nil {
		return errors.New("go.sum must be a regular file when present")
	}
	tidyCtx, cancel := context.WithTimeout(ctx, tidyTimeout)
	defer cancel()
	command := exec.CommandContext(tidyCtx, "go", "mod", "tidy", "-diff")
	command.Dir = root
	command.Env = append(os.Environ(), "GOWORK=off")
	configureCommandTermination(command)
	if err := command.Run(); err != nil {
		return errors.New("go.mod or go.sum is not tidy")
	}
	return nil
}

func requireRegularFile(path string, required bool) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) && !required {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("required module file is not regular")
	}
	return nil
}
