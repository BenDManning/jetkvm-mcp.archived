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

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
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

func (manager *Manager) VirtualMedia(ctx context.Context, name string, request mcpserver.VirtualMediaRequest) (mcpserver.VirtualMediaResult, error) {
	device, err := manager.device(name)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	mediaLock := manager.mediaLocks[device.Name]
	mediaLock.Lock()
	defer mediaLock.Unlock()
	mode, err := firmwareMediaMode(request.Mode)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	publicMode, err := requestMode(mode)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	switch request.Operation {
	case mcpserver.VirtualMediaStatus:
		return manager.virtualMediaStatus(ctx, device)
	case mcpserver.VirtualMediaMountURL:
		source, err := url.Parse(strings.TrimSpace(request.Source))
		if err != nil || source.User != nil || source.Host == "" || source.Scheme != "http" && source.Scheme != "https" {
			return mcpserver.VirtualMediaResult{}, fmt.Errorf("%w: media URL", ErrUnsupportedInput)
		}
		err = manager.provider.WithSession(ctx, device, SessionProfileData, func(session Session) error {
			return session.Call(ctx, "mountWithHTTP", map[string]any{"url": source.String(), "mode": mode}, nil)
		})
		if err != nil {
			return mcpserver.VirtualMediaResult{}, err
		}
		return mcpserver.VirtualMediaResult{Device: device.Name, Operation: request.Operation, Mounted: true, Source: source.String(), Mode: publicMode, Status: "completed"}, nil
	case mcpserver.VirtualMediaUnmount:
		err = manager.provider.WithSession(ctx, device, SessionProfileData, func(session Session) error {
			return session.Call(ctx, "unmountImage", nil, nil)
		})
		if err != nil {
			return mcpserver.VirtualMediaResult{}, err
		}
		return mcpserver.VirtualMediaResult{Device: device.Name, Operation: request.Operation, Status: "completed"}, nil
	case mcpserver.VirtualMediaUpload, mcpserver.VirtualMediaMountFile:
		return manager.uploadLocalMedia(ctx, device, request, mode)
	default:
		return mcpserver.VirtualMediaResult{}, fmt.Errorf("%w: virtual media operation", ErrUnsupportedInput)
	}
}

func (manager *Manager) virtualMediaStatus(ctx context.Context, device DeviceConfig) (mcpserver.VirtualMediaResult, error) {
	var state *firmwareVirtualMediaState
	err := manager.provider.WithSession(ctx, device, SessionProfileData, func(session Session) error {
		return session.Call(ctx, "getVirtualMediaState", nil, &state)
	})
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	result := mcpserver.VirtualMediaResult{Device: device.Name, Operation: mcpserver.VirtualMediaStatus, Status: "observed"}
	if state == nil {
		return result, nil
	}
	result.Mounted = true
	result.Mode, err = requestMode(state.Mode)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	switch state.Source {
	case "Storage":
		result.Source = state.Filename
	case "HTTP":
		result.Source = state.URL
	default:
		return mcpserver.VirtualMediaResult{}, fmt.Errorf("%w: virtual media source", ErrInvalidResponse)
	}
	return result, nil
}

func (manager *Manager) uploadLocalMedia(ctx context.Context, device DeviceConfig, request mcpserver.VirtualMediaRequest, mode string) (mcpserver.VirtualMediaResult, error) {
	file, info, filename, err := openMediaFile(device.MediaDirectory, request.Source)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	defer file.Close()
	initialDigest, err := hashMediaFile(ctx, file, info.Size())
	if err != nil {
		if ctx.Err() != nil {
			return mcpserver.VirtualMediaResult{}, ctx.Err()
		}
		return mcpserver.VirtualMediaResult{}, fmt.Errorf("%w: read media file", ErrMediaPath)
	}

	err = manager.provider.WithSession(ctx, device, SessionProfileData, func(session Session) error {
		if err := discardIncompleteUpload(ctx, session, filename); err != nil {
			return err
		}
		var started firmwareUploadStart
		if err := session.Call(ctx, "startStorageFileUpload", map[string]any{"filename": filename, "size": info.Size()}, &started); err != nil {
			return err
		}
		if started.AlreadyUploadedBytes != 0 || !uploadIDPattern.MatchString(started.UploadID) {
			abortStartedUpload(session, started.UploadID, filename)
			return fmt.Errorf("%w: upload negotiation", ErrInvalidResponse)
		}
		uploadedHash := sha256.New()
		uploadErr := session.Upload(ctx, started.UploadID, io.TeeReader(io.LimitReader(file, info.Size()), uploadedHash), info.Size())
		if uploadErr != nil {
			bestEffortDeleteUploadArtifacts(session, filename)
			return uploadErr
		}
		if !bytes.Equal(uploadedHash.Sum(nil), initialDigest) {
			bestEffortDeleteUploadArtifacts(session, filename)
			return fmt.Errorf("%w: media file changed during upload", ErrMediaPath)
		}
		if err := verifyMediaFileUnchanged(ctx, device.MediaDirectory, request.Source, info, initialDigest); err != nil {
			bestEffortDeleteUploadArtifacts(session, filename)
			return err
		}
		if request.Operation == mcpserver.VirtualMediaMountFile {
			if err := session.Call(ctx, "mountWithStorage", map[string]any{"filename": filename, "mode": mode}, nil); err != nil {
				bestEffortDeleteUploadArtifacts(session, filename)
				return err
			}
		}
		return nil
	})
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	publicMode, err := requestMode(mode)
	if err != nil {
		return mcpserver.VirtualMediaResult{}, err
	}
	return mcpserver.VirtualMediaResult{
		Device: device.Name, Operation: request.Operation, Mounted: request.Operation == mcpserver.VirtualMediaMountFile,
		Source: filename, Mode: publicMode, Status: "completed",
	}, nil
}

func discardIncompleteUpload(ctx context.Context, session Session, filename string) error {
	var storage firmwareStorageFiles
	if err := session.Call(ctx, "listStorageFiles", nil, &storage); err != nil {
		return err
	}
	incomplete := filename + ".incomplete"
	for _, file := range storage.Files {
		if file.Filename == incomplete {
			return session.Call(ctx, "deleteStorageFile", map[string]any{"filename": incomplete}, nil)
		}
	}
	return nil
}

func abortStartedUpload(session Session, uploadID, filename string) {
	if uploadIDPattern.MatchString(uploadID) {
		ctx, cancel := context.WithTimeout(context.Background(), virtualMediaCleanupTimeout)
		_ = session.Upload(ctx, uploadID, strings.NewReader(""), 0)
		cancel()
	}
	bestEffortDeleteStorageFile(session, filename+".incomplete")
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

func bestEffortDeleteUploadArtifacts(session Session, filename string) {
	for _, artifact := range []string{filename + ".incomplete", filename} {
		bestEffortDeleteStorageFile(session, artifact)
	}
}

func bestEffortDeleteStorageFile(session Session, filename string) {
	ctx, cancel := context.WithTimeout(context.Background(), virtualMediaCleanupTimeout)
	defer cancel()
	_ = session.Call(ctx, "deleteStorageFile", map[string]any{"filename": filename}, nil)
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
