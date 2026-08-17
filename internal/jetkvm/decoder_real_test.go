package jetkvm

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestRealFFmpegH264ToPNG(t *testing.T) {
	if os.Getenv("JETKVM_TEST_REAL_FFMPEG") != "1" {
		t.Skip("real FFmpeg integration is owned by the race-enabled test lane")
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	generateCtx, cancelGenerate := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelGenerate()
	command := exec.CommandContext(generateCtx, ffmpeg,
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "color=c=blue:s=64x48:d=0.1",
		"-frames:v", "1", "-c:v", "libx264", "-preset", "ultrafast", "-tune", "zerolatency",
		"-f", "h264", "pipe:1",
	)
	var h264 bytes.Buffer
	command.Stdout = &h264
	if err := command.Run(); err != nil {
		t.Fatalf("generate H.264 fixture: %v", err)
	}
	decoder, err := newFFmpegDecoder(ffmpeg, 15*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	decodeCtx, cancelDecode := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelDecode()
	pngData, width, height, err := decoder.Decode(decodeCtx, h264.Bytes(), 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	if width != 64 || height != 48 || len(pngData) == 0 {
		t.Fatalf("decoded=%d bytes %dx%d", len(pngData), width, height)
	}
}
