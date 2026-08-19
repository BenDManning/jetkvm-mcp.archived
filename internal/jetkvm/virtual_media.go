package jetkvm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/httporigin"
	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
	"github.com/BenDManning/jetkvm-mcp/internal/telemetry"
)

const (
	maxVirtualMediaBytes       int64 = 64 << 30
	virtualMediaCleanupTimeout       = 2 * time.Second
)

var uploadIDPattern = regexp.MustCompile(`^upload_[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type firmwareVirtualMediaState struct {
	Source   string `json:"source"`
	Mode     string `json:"mode"`
	Filename string `json:"filename,omitempty"`
	URL      string `json:"url,omitempty"`
	Size     int64  `json:"size"`
}

type firmwareUploadStart struct {
	AlreadyUploadedBytes int64  `json:"alreadyUploadedBytes"`
	UploadID             string `json:"dataChannel"`
}

type firmwareStorageFiles struct {
	Files []firmwareStorageFile `json:"files"`
}

type firmwareStorageFile struct {
	Filename string `json:"filename"`
}

type preparedLocalMedia struct {
	file          *os.File
	info          os.FileInfo
	filename      string
	initialDigest []byte
}

func (manager *Manager) VirtualMedia(ctx context.Context, name string, request mcpserver.VirtualMediaRequest) (mcpserver.VirtualMediaResult, error) {
	device, err := manager.resolveDevice(ctx, name)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	mode, err := firmwareMediaMode(request.Mode)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, classifyOperationError(err, ToolOutcomeNotSent)
	}
	publicMode, err := requestMode(mode)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	var sourceURL *url.URL
	var localMedia *preparedLocalMedia
	defer func() {
		if localMedia != nil {
			_ = localMedia.file.Close()
		}
	}()
	switch request.Operation {
	case mcpserver.VirtualMediaStatus, mcpserver.VirtualMediaMountURL, mcpserver.VirtualMediaUnmount, mcpserver.VirtualMediaUpload, mcpserver.VirtualMediaMountFile:
	default:
		return mcpserver.VirtualMediaResult{}, classifyOperationError(fmt.Errorf("%w: virtual media operation", ErrUnsupportedInput), ToolOutcomeNotSent)
	}
	mutating := request.Operation != mcpserver.VirtualMediaStatus
	var result mcpserver.VirtualMediaResult
	err = manager.withPreparedOperation(ctx, device, mutating, false, func() error {
		switch request.Operation {
		case mcpserver.VirtualMediaMountURL:
			sourceURL, err = parseAllowedMediaURL(request.Source, device.MediaURLAllowedOrigins)
			if err != nil {
				return classifyOperationError(err, ToolOutcomeNotSent)
			}
		case mcpserver.VirtualMediaUpload, mcpserver.VirtualMediaMountFile:
			localMedia, err = prepareLocalMedia(ctx, device, request.Source)
			return err
		}
		return nil
	}, func(operationCtx context.Context, session Session) error {
		var operationErr error
		result, operationErr = manager.virtualMedia(operationCtx, session, device, request, mode, publicMode, sourceURL, localMedia)
		return operationErr
	})
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	return result, nil
}

func (manager *Manager) virtualMedia(ctx context.Context, session Session, device DeviceConfig, request mcpserver.VirtualMediaRequest, mode, publicMode string, sourceURL *url.URL, localMedia *preparedLocalMedia) (mcpserver.VirtualMediaResult, error) {
	switch request.Operation {
	case mcpserver.VirtualMediaStatus:
		return manager.virtualMediaStatus(ctx, session, device)
	case mcpserver.VirtualMediaMountURL:
		err := session.Call(ctx, "mountWithHTTP", map[string]any{"url": sourceURL.String(), "mode": mode}, nil)
		if err != nil {
			return mcpserver.VirtualMediaResult{}, err
		}
		return mcpserver.VirtualMediaResult{Device: device.Name, Operation: request.Operation, Mounted: true, SourceType: mcpserver.VirtualMediaSourceHTTP, Mode: publicMode, Status: mcpserver.ResultStatusCompleted}, nil
	case mcpserver.VirtualMediaUnmount:
		err := session.Call(ctx, "unmountImage", nil, nil)
		if err != nil {
			return mcpserver.VirtualMediaResult{}, err
		}
		return mcpserver.VirtualMediaResult{Device: device.Name, Operation: request.Operation, Status: mcpserver.ResultStatusCompleted}, nil
	case mcpserver.VirtualMediaUpload, mcpserver.VirtualMediaMountFile:
		return manager.uploadLocalMedia(ctx, session, device, request, mode, localMedia)
	default:
		return mcpserver.VirtualMediaResult{}, classifyOperationError(fmt.Errorf("%w: virtual media operation", ErrUnsupportedInput), ToolOutcomeNotSent)
	}
}

func (manager *Manager) virtualMediaStatus(ctx context.Context, session Session, device DeviceConfig) (mcpserver.VirtualMediaResult, error) {
	var state *firmwareVirtualMediaState
	err := session.Call(ctx, "getVirtualMediaState", nil, &state)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, classifyReadFailure(err)
	}
	projected, err := publicVirtualMediaState(state)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	return mcpserver.VirtualMediaResult{
		Device: device.Name, Operation: mcpserver.VirtualMediaStatus,
		Mounted: projected.Mounted, SourceType: projected.SourceType,
		Mode: projected.Mode, Status: mcpserver.ResultStatusObserved,
	}, nil
}

func publicVirtualMediaState(state *firmwareVirtualMediaState) (*mcpserver.VirtualMediaState, error) {
	if state == nil {
		return &mcpserver.VirtualMediaState{Mounted: false}, nil
	}
	mode, err := requestMode(state.Mode)
	if err != nil {
		return nil, err
	}
	result := &mcpserver.VirtualMediaState{Mounted: true, Mode: mode}
	switch state.Source {
	case "Storage":
		result.SourceType = mcpserver.VirtualMediaSourceStorage
	case "HTTP":
		result.SourceType = mcpserver.VirtualMediaSourceHTTP
	default:
		return nil, fmt.Errorf("%w: virtual media source", ErrInvalidResponse)
	}
	return result, nil
}

func prepareLocalMedia(ctx context.Context, device DeviceConfig, source string) (*preparedLocalMedia, error) {
	file, info, filename, err := openMediaFile(device.MediaDirectory, source)
	if err != nil {
		return nil, classifyOperationError(err, ToolOutcomeNotSent)
	}
	initialDigest, err := hashMediaFile(ctx, file, info.Size())
	if err != nil {
		_ = file.Close()
		if ctx.Err() != nil {
			return nil, classifyOperationError(ctx.Err(), ToolOutcomeNotSent)
		}
		return nil, classifyOperationError(fmt.Errorf("%w: read media file", ErrMediaPath), ToolOutcomeNotSent)
	}
	return &preparedLocalMedia{file: file, info: info, filename: filename, initialDigest: initialDigest}, nil
}

func (manager *Manager) uploadLocalMedia(ctx context.Context, session Session, device DeviceConfig, request mcpserver.VirtualMediaRequest, mode string, prepared *preparedLocalMedia) (mcpserver.VirtualMediaResult, error) {
	file, info, filename, initialDigest := prepared.file, prepared.info, prepared.filename, prepared.initialDigest
	mutated := false
	err := func() error {
		var err error
		mutated, err = discardExistingUploadArtifacts(ctx, session, filename)
		if err != nil {
			return mutationSequenceError(err, mutated)
		}
		var started firmwareUploadStart
		if err := session.Call(ctx, "startStorageFileUpload", map[string]any{"filename": filename, "size": info.Size()}, &started); err != nil {
			return mutationSequenceError(err, mutated)
		}
		mutated = true
		if started.AlreadyUploadedBytes != 0 || !uploadIDPattern.MatchString(started.UploadID) {
			abortStartedUpload(ctx, session, started.UploadID, filename)
			return classifyOperationError(fmt.Errorf("%w: upload negotiation", ErrInvalidResponse), ToolOutcomeUnknown)
		}
		uploadedHash := sha256.New()
		uploadErr := session.Upload(ctx, started.UploadID, io.TeeReader(io.LimitReader(file, info.Size()), uploadedHash), info.Size())
		if uploadErr != nil {
			bestEffortDeleteUploadArtifacts(ctx, session, filename)
			return mutationSequenceError(uploadErr, true)
		}
		if !bytes.Equal(uploadedHash.Sum(nil), initialDigest) {
			bestEffortDeleteUploadArtifacts(ctx, session, filename)
			return classifyOperationError(fmt.Errorf("%w: media file changed during upload", ErrMediaPath), ToolOutcomeUnknown)
		}
		if err := verifyMediaFileUnchanged(ctx, device.MediaDirectory, request.Source, info, initialDigest); err != nil {
			bestEffortDeleteUploadArtifacts(ctx, session, filename)
			return mutationSequenceError(err, true)
		}
		if request.Operation == mcpserver.VirtualMediaMountFile {
			if err := session.Call(ctx, "mountWithStorage", map[string]any{"filename": filename, "mode": mode}, nil); err != nil {
				bestEffortDeleteUploadArtifacts(ctx, session, filename)
				return mutationSequenceError(err, true)
			}
		}
		return nil
	}()
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	publicMode, err := requestMode(mode)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	return mcpserver.VirtualMediaResult{
		Device: device.Name, Operation: request.Operation, Mounted: request.Operation == mcpserver.VirtualMediaMountFile,
		SourceType: mcpserver.VirtualMediaSourceStorage, Mode: publicMode, Status: mcpserver.ResultStatusCompleted,
	}, nil
}

func discardExistingUploadArtifacts(ctx context.Context, session Session, filename string) (mutated bool, returnErr error) {
	cleanupStage := telemetry.BeginStage(ctx, telemetry.StageCleanup)
	defer func() { finishTelemetryStage(cleanupStage, returnErr) }()
	var storage firmwareStorageFiles
	if err := session.Call(ctx, "listStorageFiles", nil, &storage); err != nil {
		return false, classifyOperationError(err, ToolOutcomeNotSent)
	}
	present := make(map[string]bool, len(storage.Files))
	for _, file := range storage.Files {
		present[file.Filename] = true
	}
	for _, artifact := range []string{filename + ".incomplete", filename} {
		if !present[artifact] {
			continue
		}
		if err := session.Call(ctx, "deleteStorageFile", map[string]any{"filename": artifact}, nil); err != nil {
			return mutated, err
		}
		mutated = true
	}
	return mutated, nil
}

func abortStartedUpload(operationCtx context.Context, session Session, uploadID, filename string) {
	if uploadIDPattern.MatchString(uploadID) {
		ctx, cancel := detachedCleanupContext(operationCtx)
		_ = session.Upload(ctx, uploadID, strings.NewReader(""), 0)
		cancel()
	}
	bestEffortDeleteStorageFile(operationCtx, session, filename+".incomplete")
}

func verifyMediaFileUnchanged(ctx context.Context, directory, source string, before os.FileInfo, expectedDigest []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, after, _, err := openMediaFile(directory, source)
	if err != nil {
		return fmt.Errorf("%w: media file changed during upload", ErrMediaPath)
	}
	defer file.Close()
	if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return fmt.Errorf("%w: media file changed during upload", ErrMediaPath)
	}
	digest, err := hashMediaFile(ctx, file, after.Size())
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil || !bytes.Equal(digest, expectedDigest) {
		return fmt.Errorf("%w: media file changed during upload", ErrMediaPath)
	}
	final, err := file.Stat()
	if err != nil || !os.SameFile(after, final) || after.Size() != final.Size() || !after.ModTime().Equal(final.ModTime()) {
		return fmt.Errorf("%w: media file changed during upload", ErrMediaPath)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	current, currentInfo, _, err := openMediaFile(directory, source)
	if err != nil {
		return fmt.Errorf("%w: media file changed during upload", ErrMediaPath)
	}
	current.Close()
	if !os.SameFile(after, currentInfo) || after.Size() != currentInfo.Size() || !after.ModTime().Equal(currentInfo.ModTime()) {
		return fmt.Errorf("%w: media file changed during upload", ErrMediaPath)
	}
	return nil
}

func hashMediaFile(ctx context.Context, file *os.File, size int64) ([]byte, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	digest, err := hashMediaReader(ctx, file, size)
	if err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return digest, nil
}

func hashMediaReader(ctx context.Context, reader io.Reader, size int64) ([]byte, error) {
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(&contextReader{ctx: ctx, reader: reader}, size+1))
	if err != nil {
		return nil, err
	}
	if written != size {
		return nil, fmt.Errorf("media size changed while hashing")
	}
	return hash.Sum(nil), nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader *contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}

func bestEffortDeleteUploadArtifacts(operationCtx context.Context, session Session, filename string) {
	for _, artifact := range []string{filename + ".incomplete", filename} {
		bestEffortDeleteStorageFile(operationCtx, session, artifact)
	}
}

func bestEffortDeleteStorageFile(operationCtx context.Context, session Session, filename string) {
	ctx, cancel := detachedCleanupContext(operationCtx)
	defer cancel()
	_ = session.Call(ctx, "deleteStorageFile", map[string]any{"filename": filename}, nil)
}

func detachedCleanupContext(operationCtx context.Context) (context.Context, context.CancelFunc) {
	if operationCtx == nil {
		operationCtx = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(operationCtx), virtualMediaCleanupTimeout)
}

func openMediaFile(directory, source string) (*os.File, os.FileInfo, string, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, nil, "", ErrMediaDirectoryNotConfigured
	}
	if filepath.IsAbs(source) {
		return nil, nil, "", fmt.Errorf("%w: absolute path", ErrMediaPath)
	}
	clean := filepath.Clean(strings.TrimSpace(source))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, nil, "", fmt.Errorf("%w: traversal", ErrMediaPath)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, nil, "", fmt.Errorf("%w: media directory", ErrMediaPath)
	}
	defer root.Close()
	file, err := root.Open(clean)
	if err != nil {
		return nil, nil, "", fmt.Errorf("%w: open media file", ErrMediaPath)
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxVirtualMediaBytes {
		file.Close()
		return nil, nil, "", fmt.Errorf("%w: regular media file required", ErrMediaPath)
	}
	return file, info, filepath.Base(clean), nil
}

func firmwareMediaMode(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", "read_only":
		return "CDROM", nil
	case "read_write":
		return "Disk", nil
	default:
		return "", fmt.Errorf("%w: media mode", ErrUnsupportedInput)
	}
}

func requestMode(value string) (string, error) {
	switch value {
	case "CDROM":
		return "read_only", nil
	case "Disk":
		return "read_write", nil
	default:
		return "", fmt.Errorf("%w: virtual media mode", ErrInvalidResponse)
	}
}

func normalizeMediaURLAllowedOrigins(values []string) ([]string, error) {
	origins := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		origin, err := normalizeMediaURLOrigin(value)
		if err != nil || strings.Contains(origin, "*") {
			return nil, fmt.Errorf("invalid media URL origin")
		}
		if _, duplicate := seen[origin]; duplicate {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	return origins, nil
}

func mediaURLOriginAllowed(source *url.URL, allowed []string) bool {
	if source == nil {
		return false
	}
	origin, err := normalizeMediaURLOrigin(strings.ToLower(source.Scheme) + "://" + source.Host)
	if err != nil {
		return false
	}
	for _, candidate := range allowed {
		if origin == candidate {
			return true
		}
	}
	return false
}

func normalizeMediaURLOrigin(value string) (string, error) {
	origin, err := httporigin.ParseEffective(value)
	return origin.Value, err
}

func parseAllowedMediaURL(value string, allowed []string) (*url.URL, error) {
	if len(allowed) == 0 {
		return nil, fmt.Errorf("%w: media URL origin", ErrUnsupportedInput)
	}
	source, err := url.Parse(strings.TrimSpace(value))
	if err != nil || source == nil || source.Opaque != "" || source.User != nil || source.Host == "" || source.Hostname() == "" || source.Scheme != "http" && source.Scheme != "https" || !mediaURLOriginAllowed(source, allowed) {
		return nil, fmt.Errorf("%w: media URL", ErrUnsupportedInput)
	}
	return source, nil
}
