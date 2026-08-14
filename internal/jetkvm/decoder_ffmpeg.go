package jetkvm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

const (
	maxAnnexBBytes  = 4 << 20
	maxFFmpegStderr = 16 << 10
)

var errDecoderOutputLimit = errors.New("decoder output limit exceeded")

type ffmpegDecoder struct {
	executable string
	timeout    time.Duration
}

func NewFFmpegDecoder() (Decoder, error) {
	executable, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, ErrDecoderUnavailable
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return nil, ErrDecoderUnavailable
	}
	return newFFmpegDecoder(executable, 15*time.Second)
}

func newFFmpegDecoder(executable string, timeout time.Duration) (Decoder, error) {
	if !filepath.IsAbs(executable) {
		return nil, errors.New("FFmpeg executable must be absolute")
	}
	info, err := os.Stat(executable)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return nil, ErrDecoderUnavailable
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &ffmpegDecoder{executable: filepath.Clean(executable), timeout: timeout}, nil
}

func (decoder *ffmpegDecoder) Decode(ctx context.Context, annexB []byte, maxWidth, maxHeight int) ([]byte, int, int, error) {
	if decoder == nil || decoder.executable == "" {
		return nil, 0, 0, ErrDecoderUnavailable
	}
	if len(annexB) == 0 || len(annexB) > maxAnnexBBytes {
		return nil, 0, 0, fmt.Errorf("%w: H.264 frame size", ErrInvalidResponse)
	}
	args, err := buildFFmpegArgs(maxWidth, maxHeight)
	if err != nil {
		return nil, 0, 0, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	decodeCtx, cancel := context.WithTimeout(ctx, decoder.timeout)
	defer cancel()
	command := exec.CommandContext(decodeCtx, decoder.executable, args...)
	command.Env = []string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"}
	command.Stdin = bytes.NewReader(annexB)
	stdout := &boundedBuffer{limit: maxCapturePNGBytes}
	stderr := &boundedBuffer{limit: maxFFmpegStderr, discardOverflow: true}
	command.Stdout = stdout
	command.Stderr = stderr
	command.WaitDelay = 250 * time.Millisecond
	if err := command.Run(); err != nil {
		if errors.Is(decodeCtx.Err(), context.Canceled) || errors.Is(decodeCtx.Err(), context.DeadlineExceeded) {
			return nil, 0, 0, decodeCtx.Err()
		}
		if errors.Is(stdout.err, errDecoderOutputLimit) {
			return nil, 0, 0, errDecoderOutputLimit
		}
		return nil, 0, 0, ErrInvalidResponse
	}
	return validateDecodedPNG(decodeCtx, stdout.Bytes(), maxWidth, maxHeight)
}

func buildFFmpegArgs(maxWidth, maxHeight int) ([]string, error) {
	if maxWidth < 1 || maxWidth > 3840 || maxHeight < 1 || maxHeight > 2160 {
		return nil, fmt.Errorf("%w: capture bounds", ErrUnsupportedInput)
	}
	filter := "scale=w='min(iw," + strconv.Itoa(maxWidth) + ")':h='min(ih," + strconv.Itoa(maxHeight) + ")':force_original_aspect_ratio=decrease,setsar=1"
	return []string{
		"-nostdin", "-hide_banner", "-loglevel", "error", "-filter_threads", "1", "-protocol_whitelist", "pipe",
		"-f", "h264", "-probesize", "4194304", "-analyzeduration", "1000000", "-max_alloc", "67108864",
		"-max_pixels", "8294400", "-threads:v", "1", "-i", "pipe:0", "-map", "0:v:0", "-frames:v", "1",
		"-vf", filter, "-threads:v", "1", "-pix_fmt", "rgb24", "-c:v", "png", "-compression_level", "3",
		"-f", "image2pipe", "pipe:1",
	}, nil
}

func validateDecodedPNG(ctx context.Context, data []byte, maxWidth, maxHeight int) ([]byte, int, int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, 0, err
	}
	if len(data) == 0 || len(data) > maxCapturePNGBytes {
		return nil, 0, 0, ErrInvalidResponse
	}
	config, err := png.DecodeConfig(&contextReader{ctx: ctx, reader: bytes.NewReader(data)})
	if contextErr := ctx.Err(); contextErr != nil {
		return nil, 0, 0, contextErr
	}
	if err != nil || config.Width < 1 || config.Width > maxWidth || config.Height < 1 || config.Height > maxHeight || int64(config.Width)*int64(config.Height) > 8_294_400 {
		return nil, 0, 0, ErrInvalidResponse
	}
	if _, err := png.Decode(&contextReader{ctx: ctx, reader: bytes.NewReader(data)}); err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, 0, 0, contextErr
		}
		return nil, 0, 0, ErrInvalidResponse
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, 0, err
	}
	return append([]byte(nil), data...), config.Width, config.Height, nil
}

type boundedBuffer struct {
	bytes.Buffer
	limit           int
	discardOverflow bool
	err             error
}

func (buffer *boundedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := buffer.limit - buffer.Len()
	if remaining > 0 {
		toWrite := data
		if len(toWrite) > remaining {
			toWrite = toWrite[:remaining]
		}
		_, _ = buffer.Buffer.Write(toWrite)
	}
	if original > remaining && !buffer.discardOverflow {
		buffer.err = errDecoderOutputLimit
		return 0, buffer.err
	}
	return original, nil
}
