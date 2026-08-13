// jetkvm-mcp-validate performs a deliberately read-only validation of a real
// JetKVM through the jetkvm-mcp stdio interface.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	statusTool      = "jetkvm_get_status"
	captureTool     = "jetkvm_capture_screen"
	mediaStatusTool = "jetkvm_get_virtual_media_status"
	captureTimeout  = 30 * time.Second
)

type options struct {
	binary string
	config string
	device string
}

type captureMetadata struct {
	Width     int `json:"width"`
	Height    int `json:"height"`
	SizeBytes int `json:"size_bytes"`
}

type validationReport struct {
	Result  string           `json:"result"`
	Checks  []string         `json:"checks,omitempty"`
	Failed  string           `json:"failed,omitempty"`
	Capture *captureMetadata `json:"capture,omitempty"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		emit(validationReport{Result: "fail", Failed: "arguments"})
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	report := runValidation(ctx, opts)
	emit(report)
	if report.Result != "pass" {
		os.Exit(1)
	}
}

func emit(report validationReport) {
	// Reports intentionally exclude paths, device names, status values, child
	// output, image bytes, and error strings.
	_ = json.NewEncoder(os.Stdout).Encode(report)
}

func parseArgs(args []string) (options, error) {
	flags := flag.NewFlagSet("jetkvm-mcp-validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	binary := flags.String("binary", "", "path to the jetkvm-mcp binary")
	config := flags.String("config", "", "path to its configuration")
	device := flags.String("device", "", "configured device name")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return options{}, errors.New("invalid arguments")
	}
	if strings.TrimSpace(*binary) == "" || strings.TrimSpace(*config) == "" || strings.TrimSpace(*device) == "" {
		return options{}, errors.New("binary, config, and device are required")
	}
	return options{binary: *binary, config: *config, device: *device}, nil
}

func runValidation(ctx context.Context, opts options) validationReport {
	checks := make([]string, 0, 3)
	fail := func(stage string) validationReport {
		return validationReport{Result: "fail", Checks: checks, Failed: stage}
	}

	command := exec.CommandContext(ctx, opts.binary, "--config", opts.config)
	// A real device or server can put private diagnostic details on stderr.
	// Leaving Stderr nil makes os/exec discard them rather than retaining or
	// forwarding them from this sanitizing runner.
	client := mcp.NewClient(&mcp.Implementation{Name: "jetkvm-mcp-read-only-validator", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		return fail("connect")
	}
	defer session.Close()

	listed, err := session.ListTools(ctx, nil)
	if err != nil || validateReadOnlyTools(listed.Tools) != nil {
		return fail("tools_list")
	}
	checks = append(checks, "tools_list")

	status, err := session.CallTool(ctx, &mcp.CallToolParams{Name: statusTool, Arguments: map[string]any{"device": opts.device}})
	if err != nil || status.IsError {
		return fail("status")
	}
	statusObject, ok := status.StructuredContent.(map[string]any)
	if !ok || validateStatus(statusObject, opts.device) != nil {
		return fail("status")
	}
	checks = append(checks, "status")

	captureCtx, cancelCapture := context.WithTimeout(ctx, captureTimeout)
	defer cancelCapture()
	capture, err := session.CallTool(captureCtx, &mcp.CallToolParams{Name: captureTool, Arguments: map[string]any{"device": opts.device}})
	if err != nil || capture.IsError {
		return fail("capture")
	}
	metadata, err := decodeCapture(capture, opts.device)
	if err != nil {
		return fail("capture")
	}
	checks = append(checks, "capture")
	return validationReport{Result: "pass", Checks: checks, Capture: &metadata}
}

func validateReadOnlyTools(tools []*mcp.Tool) error {
	byName := make(map[string]*mcp.Tool, len(tools))
	for _, tool := range tools {
		byName[tool.Name] = tool
	}
	for _, name := range []string{statusTool, captureTool, mediaStatusTool} {
		tool := byName[name]
		if tool == nil || tool.Annotations == nil || !tool.Annotations.ReadOnlyHint ||
			tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint ||
			!tool.Annotations.IdempotentHint || tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
			return errors.New("required read-only tool or annotations missing")
		}
	}
	return nil
}

func validateStatus(status map[string]any, device string) error {
	if value, ok := status["device"].(string); !ok || value == "" || value != device {
		return errors.New("invalid device field")
	}
	if connected, ok := status["connected"].(bool); !ok || !connected {
		return errors.New("invalid connected field")
	}
	for _, name := range []string{"applicationVersion", "systemVersion", "activeExtension", "usbState"} {
		if value, exists := status[name]; exists {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("invalid %s field", name)
			}
		}
	}
	if value, exists := status["virtualMedia"]; exists {
		media, ok := value.(map[string]any)
		if !ok || validateVirtualMediaState(media) != nil {
			return errors.New("invalid virtualMedia field")
		}
	}
	for _, name := range []string{"atxPowerOn", "dcPowerOn", "videoReady", "usbWakeAttached"} {
		if value, exists := status[name]; exists {
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("invalid %s field", name)
			}
		}
	}
	for _, name := range []string{"dcVoltage", "videoWidth", "videoHeight", "videoFPS"} {
		if value, exists := status[name]; exists {
			if _, ok := value.(float64); !ok {
				return fmt.Errorf("invalid %s field", name)
			}
		}
	}
	if value, exists := status["warnings"]; exists {
		warnings, ok := value.([]any)
		if !ok {
			return errors.New("invalid warnings field")
		}
		for _, warning := range warnings {
			if _, ok := warning.(string); !ok {
				return errors.New("invalid warnings item")
			}
		}
	}
	return nil
}

func validateVirtualMediaState(media map[string]any) error {
	mounted, ok := media["mounted"].(bool)
	if !ok {
		return errors.New("invalid mounted field")
	}
	allowed := map[string]bool{"mounted": true, "sourceType": true, "mode": true}
	for name := range media {
		if !allowed[name] {
			return errors.New("unexpected virtual media field")
		}
	}
	if !mounted {
		if len(media) != 1 {
			return errors.New("unmounted media must not describe a source")
		}
		return nil
	}
	sourceType, sourceOK := media["sourceType"].(string)
	mode, modeOK := media["mode"].(string)
	if !sourceOK || sourceType != "http" && sourceType != "storage" || !modeOK || mode != "read_only" && mode != "read_write" {
		return errors.New("invalid mounted media state")
	}
	return nil
}

func decodeCapture(result *mcp.CallToolResult, device string) (captureMetadata, error) {
	object, ok := result.StructuredContent.(map[string]any)
	if !ok {
		return captureMetadata{}, errors.New("invalid capture metadata")
	}
	if value, ok := object["device"].(string); !ok || value == "" || value != device {
		return captureMetadata{}, errors.New("invalid capture device")
	}
	if value, ok := object["mimeType"].(string); !ok || value != "image/png" {
		return captureMetadata{}, errors.New("invalid capture MIME type")
	}
	capturedAt, ok := object["capturedAt"].(string)
	if !ok {
		return captureMetadata{}, errors.New("invalid capture timestamp")
	}
	if _, err := time.Parse(time.RFC3339Nano, capturedAt); err != nil {
		return captureMetadata{}, errors.New("invalid capture timestamp")
	}
	width, widthOK := positiveInt(object["width"])
	height, heightOK := positiveInt(object["height"])
	size, sizeOK := positiveInt(object["sizeBytes"])
	if !widthOK || !heightOK || !sizeOK {
		return captureMetadata{}, errors.New("invalid capture dimensions")
	}

	var pngData []byte
	for _, content := range result.Content {
		imageContent, ok := content.(*mcp.ImageContent)
		if !ok {
			continue
		}
		if pngData != nil || imageContent.MIMEType != "image/png" {
			return captureMetadata{}, errors.New("invalid image content")
		}
		pngData = imageContent.Data
	}
	if len(pngData) != size {
		return captureMetadata{}, errors.New("capture size mismatch")
	}
	defer clear(pngData)
	decoded, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return captureMetadata{}, errors.New("PNG decode failed")
	}
	bounds := decoded.Bounds()
	if bounds.Dx() != width || bounds.Dy() != height {
		return captureMetadata{}, errors.New("PNG dimensions mismatch")
	}
	return captureMetadata{Width: width, Height: height, SizeBytes: size}, nil
}

func positiveInt(value any) (int, bool) {
	number, ok := value.(float64)
	integer := int(number)
	return integer, ok && number > 0 && float64(integer) == number
}
