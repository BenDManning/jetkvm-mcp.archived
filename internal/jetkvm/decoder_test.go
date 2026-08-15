package jetkvm

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBuildFFmpegArgsUsesBoundedPipeOnlyPipeline(t *testing.T) {
	args, err := buildFFmpegArgs(800, 600)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, required := range []string{"-nostdin", "-protocol_whitelist pipe", "-f h264", "pipe:0", "-frames:v 1", "image2pipe", "pipe:1", "min(iw,800)", "min(ih,600)"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("args missing %q: %s", required, joined)
		}
	}
}

func TestCapturePNGBudgetAccommodatesMaximumRGBFrame(t *testing.T) {
	const minimum = 3840*2160*3 + 2160 + 1<<20
	if maxCapturePNGBytes < minimum {
		t.Fatalf("PNG budget = %d, want at least %d bytes", maxCapturePNGBytes, minimum)
	}
}

func TestFFmpegDecoderRunsLocalProcessAndValidatesPNG(t *testing.T) {
	directory := t.TempDir()
	pngPath := filepath.Join(directory, "fixture.png")
	pngData := testPNG(t, 2, 1)
	if err := os.WriteFile(pngPath, pngData, 0o600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(directory, "ffmpeg-fixture")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\n/bin/cat '"+pngPath+"'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	decoder, err := newFFmpegDecoder(executable, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	decoded, width, height, err := decoder.Decode(context.Background(), []byte{0, 0, 0, 1, 0x65}, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	if width != 2 || height != 1 || !bytes.Equal(decoded, pngData) {
		t.Fatalf("decoded=%d bytes %dx%d", len(decoded), width, height)
	}
}

func TestFFmpegDecoderRejectsMalformedOutput(t *testing.T) {
	directory := t.TempDir()
	executable := filepath.Join(directory, "ffmpeg-fixture")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf 'not a png'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	decoder, err := newFFmpegDecoder(executable, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := decoder.Decode(context.Background(), []byte{0, 0, 0, 1, 0x65}, 800, 600); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("error = %v", err)
	}
}

func TestFFmpegDecoderBoundsCleanupAfterCallerDeadline(t *testing.T) {
	directory := t.TempDir()
	executable := filepath.Join(directory, "ffmpeg-fixture")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nsleep 2 &\nwait\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	decoder, err := newFFmpegDecoder(executable, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	const budget = 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	started := time.Now()
	_, _, _, err = decoder.Decode(ctx, []byte{0, 0, 0, 1, 0x65}, 800, 600)
	elapsed := time.Since(started)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
	if elapsed > budget+ffmpegWaitDelay+150*time.Millisecond {
		t.Fatalf("decode elapsed = %v, want cleanup within caller budget plus WaitDelay", elapsed)
	}
}

func TestFFmpegDecoderMayCompleteBeforeCallerDeadline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-specific")
	}
	directory := t.TempDir()
	pngPath := filepath.Join(directory, "fixture.png")
	if err := os.WriteFile(pngPath, testPNG(t, 2, 1), 0o600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(directory, "ffmpeg-fixture")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nsleep 2.8\n/bin/cat '"+pngPath+"'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	decoder, err := newFFmpegDecoder(executable, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	decoded, width, height, err := decoder.Decode(ctx, []byte{0, 0, 0, 1, 0x65}, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	if width != 2 || height != 1 || len(decoded) == 0 {
		t.Fatalf("decoded=%d bytes %dx%d", len(decoded), width, height)
	}
}

func TestActiveHelperProcessCountTracksRunningDecoder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-specific")
	}
	directory := t.TempDir()
	executable := filepath.Join(directory, "ffmpeg-fixture")
	if err := os.WriteFile(executable, []byte("#!/usr/bin/python3\nimport time\ntime.sleep(30)\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	decoder, err := newFFmpegDecoder(executable, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, _, _, err := decoder.Decode(ctx, []byte{0, 0, 0, 1, 0x65}, 800, 600)
		done <- err
	}()
	deadline := time.After(time.Second)
	for ActiveHelperProcesses() != 1 {
		select {
		case <-deadline:
			cancel()
			t.Fatal("running helper was not counted")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("decode error=%v", err)
	}
	if count := ActiveHelperProcesses(); count != 0 {
		t.Fatalf("active helpers after cleanup=%d", count)
	}
}

func TestValidateDecodedPNGStopsWhenContextEndsBetweenReads(t *testing.T) {
	ctx := newCancelOnSecondReadContext()

	_, _, _, err := validateDecodedPNG(ctx, testPNG(t, 2, 1), 800, 600)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if ctx.calls < 2 {
		t.Fatalf("context read checks = %d, want at least 2", ctx.calls)
	}
}

type cancelOnSecondReadContext struct {
	calls int
}

func newCancelOnSecondReadContext() *cancelOnSecondReadContext {
	return &cancelOnSecondReadContext{}
}

func (ctx *cancelOnSecondReadContext) Deadline() (time.Time, bool) { return time.Time{}, false }

func (*cancelOnSecondReadContext) Done() <-chan struct{} { return nil }

func (ctx *cancelOnSecondReadContext) Err() error {
	ctx.calls++
	if ctx.calls >= 2 {
		return context.Canceled
	}
	return nil
}

func (*cancelOnSecondReadContext) Value(any) any { return nil }
