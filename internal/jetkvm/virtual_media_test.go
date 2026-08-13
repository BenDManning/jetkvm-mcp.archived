package jetkvm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

type concurrentMediaProvider struct {
	active atomic.Int32
	max    atomic.Int32
}

func (provider *concurrentMediaProvider) WithSession(_ context.Context, _ DeviceConfig, _ SessionProfile, operation func(Session) error) error {
	active := provider.active.Add(1)
	for current := provider.max.Load(); active > current && !provider.max.CompareAndSwap(current, active); current = provider.max.Load() {
	}
	defer provider.active.Add(-1)
	time.Sleep(20 * time.Millisecond)
	return operation(&fakeSession{results: map[string]any{
		"listStorageFiles":       map[string]any{"files": []any{}},
		"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
	}})
}

func TestManagerVirtualMediaStatusURLMountAndUnmount(t *testing.T) {
	for _, test := range []struct {
		name    string
		request mcpserver.VirtualMediaRequest
		results map[string]any
		want    []recordedCall
		mounted bool
	}{
		{name: "status", request: mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaStatus}, results: map[string]any{
			"getVirtualMediaState": map[string]any{"source": "HTTP", "mode": "CDROM", "url": "https://example.invalid/image.iso", "size": 1024},
		}, want: []recordedCall{{method: "getVirtualMediaState", params: nil}}, mounted: true},
		{name: "mount URL", request: mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaMountURL, Source: "https://example.invalid/image.iso"}, results: map[string]any{}, want: []recordedCall{{method: "mountWithHTTP", params: map[string]any{"url": "https://example.invalid/image.iso", "mode": "CDROM"}}}, mounted: true},
		{name: "unmount", request: mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaUnmount}, results: map[string]any{}, want: []recordedCall{{method: "unmountImage", params: nil}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			session := &fakeSession{results: test.results}
			manager := testManager(t, session)
			result, err := manager.VirtualMedia(context.Background(), "lab", test.request)
			if err != nil {
				t.Fatal(err)
			}
			if result.Mounted != test.mounted || !reflect.DeepEqual(session.calls, test.want) {
				t.Fatalf("result=%+v calls=%#v want=%#v", result, session.calls, test.want)
			}
		})
	}
}

func TestManagerVirtualMediaMountClassifiesConnectionFailureAsNotSent(t *testing.T) {
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &failingBeforeSessionProvider{err: ErrDeviceUnreachable})
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{
		Operation: mcpserver.VirtualMediaMountURL, Source: "https://media.invalid/install.iso",
	})
	assertToolOutcome(t, err, ToolOutcomeNotSent)
}

func TestManagerVirtualMediaRejectsURLCredentialsWithoutLeak(t *testing.T) {
	session := &fakeSession{results: map[string]any{}}
	manager := testManager(t, session)
	_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{
		Operation: mcpserver.VirtualMediaMountURL,
		Source:    "https://private-user:private-password@example.invalid/image.iso",
	})
	if !errors.Is(err, ErrUnsupportedInput) {
		t.Fatalf("error = %v, want unsupported input", err)
	}
	if errText := err.Error(); strings.Contains(errText, "private-user") || strings.Contains(errText, "private-password") {
		t.Fatalf("error leaked URL credentials: %v", err)
	}
	if len(session.calls) != 0 {
		t.Fatalf("device calls = %#v, want none", session.calls)
	}
}

func TestManagerVirtualMediaLocalPathFailureIsDefinitelyNotSent(t *testing.T) {
	manager := testManager(t, &fakeSession{results: map[string]any{}})
	_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{
		Operation: mcpserver.VirtualMediaUpload, Source: "missing.iso",
	})
	assertToolOutcome(t, err, ToolOutcomeNotSent)
}

func TestManagerVirtualMediaPriorCleanupMakesLaterNotSentUnknown(t *testing.T) {
	mediaDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(mediaDirectory, "install.iso"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{results: map[string]any{
		"listStorageFiles": map[string]any{"files": []map[string]any{{"filename": "install.iso.incomplete"}}},
	}}
	session.callHook = func(_ context.Context, method string, _ any) error {
		if method == "startStorageFileUpload" {
			return classifyOperationError(context.Canceled, ToolOutcomeNotSent)
		}
		return nil
	}
	manager := mediaManager(t, session, mediaDirectory)
	_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{
		Operation: mcpserver.VirtualMediaUpload, Source: "install.iso",
	})
	assertToolOutcome(t, err, ToolOutcomeUnknown)
}

func TestManagerVirtualMediaRejectsUnknownFirmwareMode(t *testing.T) {
	session := &fakeSession{results: map[string]any{
		"getVirtualMediaState": map[string]any{"source": "HTTP", "mode": "FutureMode", "url": "https://example.invalid/image.iso"},
	}}
	manager := testManager(t, session)
	_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaStatus})
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("error = %v, want invalid firmware response", err)
	}
}

func TestManagerVirtualMediaStatusRejectsUnknownFirmwareSourceWithoutLeak(t *testing.T) {
	const sentinel = "JETKVM-PRIVATE-FIRMWARE-SENTINEL-36bf27"
	session := &fakeSession{results: map[string]any{
		"getVirtualMediaState": map[string]any{
			"source": sentinel, "mode": "CDROM", "filename": sentinel + ".iso",
			"url": "https://media.invalid/" + sentinel + ".iso?token=" + sentinel,
		},
	}}
	manager := testManager(t, session)
	_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaStatus})
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("error = %v, want invalid firmware response", err)
	}
	if strings.Contains(err.Error(), sentinel) || strings.Contains(err.Error(), "media.invalid") {
		t.Fatalf("error leaked private firmware state: %v", err)
	}
}

func TestManagerVirtualMediaUploadsAndMountsConfiguredFile(t *testing.T) {
	mediaDirectory := t.TempDir()
	contents := []byte("small ISO fixture")
	if err := os.WriteFile(filepath.Join(mediaDirectory, "install.iso"), contents, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []mcpserver.VirtualMediaOperation{mcpserver.VirtualMediaUpload, mcpserver.VirtualMediaMountFile} {
		t.Run(string(operation), func(t *testing.T) {
			session := &fakeSession{results: map[string]any{
				"listStorageFiles":       map[string]any{"files": []any{}},
				"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
			}}
			manager := mediaManager(t, session, mediaDirectory)
			result, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: operation, Source: "install.iso"})
			if err != nil {
				t.Fatal(err)
			}
			if len(session.uploads) != 1 {
				t.Fatalf("uploads = %#v", session.uploads)
			}
			upload := session.uploads[0]
			if upload.id != "upload_12345678-1234-1234-1234-123456789abc" || upload.size != int64(len(contents)) || !bytes.Equal(upload.data, contents) {
				t.Fatalf("upload = %#v", upload)
			}
			wantCalls := []recordedCall{
				{method: "listStorageFiles", params: nil},
				{method: "startStorageFileUpload", params: map[string]any{"filename": "install.iso", "size": int64(len(contents))}},
			}
			if operation == mcpserver.VirtualMediaMountFile {
				wantCalls = append(wantCalls, recordedCall{method: "mountWithStorage", params: map[string]any{"filename": "install.iso", "mode": "CDROM"}})
			}
			if result.Status != "completed" || !reflect.DeepEqual(session.calls, wantCalls) {
				t.Fatalf("result=%+v calls=%#v want=%#v", result, session.calls, wantCalls)
			}
		})
	}
}

func TestManagerVirtualMediaDeletesUnverifiablePartialBeforeFreshUpload(t *testing.T) {
	for _, operation := range []mcpserver.VirtualMediaOperation{mcpserver.VirtualMediaUpload, mcpserver.VirtualMediaMountFile} {
		t.Run(string(operation), func(t *testing.T) {
			mediaDirectory := t.TempDir()
			contents := []byte("replacement ISO fixture")
			if err := os.WriteFile(filepath.Join(mediaDirectory, "install.iso"), contents, 0o600); err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{results: map[string]any{
				"listStorageFiles":       map[string]any{"files": []map[string]any{{"filename": "install.iso.incomplete", "size": 5}}},
				"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
			}}
			manager := mediaManager(t, session, mediaDirectory)
			result, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: operation, Source: "install.iso"})
			if err != nil {
				t.Fatal(err)
			}
			if len(session.uploads) != 1 || !bytes.Equal(session.uploads[0].data, contents) || session.uploads[0].size != int64(len(contents)) {
				t.Fatalf("uploads = %#v, want complete replacement", session.uploads)
			}
			wantCalls := []recordedCall{
				{method: "listStorageFiles", params: nil},
				{method: "deleteStorageFile", params: map[string]any{"filename": "install.iso.incomplete"}},
				{method: "startStorageFileUpload", params: map[string]any{"filename": "install.iso", "size": int64(len(contents))}},
			}
			if operation == mcpserver.VirtualMediaMountFile {
				wantCalls = append(wantCalls, recordedCall{method: "mountWithStorage", params: map[string]any{"filename": "install.iso", "mode": "CDROM"}})
			}
			if result.Status != "completed" || !reflect.DeepEqual(session.calls, wantCalls) {
				t.Fatalf("result=%+v calls=%#v want=%#v", result, session.calls, wantCalls)
			}
		})
	}
}

func TestManagerVirtualMediaRejectsUnexpectedResumeOffsetAndAbortsUpload(t *testing.T) {
	mediaDirectory := t.TempDir()
	contents := []byte("new ISO fixture")
	if err := os.WriteFile(filepath.Join(mediaDirectory, "install.iso"), contents, 0o600); err != nil {
		t.Fatal(err)
	}
	const uploadID = "upload_12345678-1234-1234-1234-123456789abc"
	session := &fakeSession{results: map[string]any{
		"listStorageFiles":       map[string]any{"files": []any{}},
		"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 5, "dataChannel": uploadID},
	}}
	manager := mediaManager(t, session, mediaDirectory)
	_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaUpload, Source: "install.iso"})
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("error = %v, want invalid resume response", err)
	}
	if len(session.uploads) != 1 || session.uploads[0].id != uploadID || session.uploads[0].size != 0 || len(session.uploads[0].data) != 0 {
		t.Fatalf("abort uploads = %#v, want one empty upload", session.uploads)
	}
	wantMethods := []string{"listStorageFiles", "startStorageFileUpload", "deleteStorageFile"}
	if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
		t.Fatalf("methods = %v, want %v", got, wantMethods)
	}
	if !reflect.DeepEqual(session.calls[2].params, map[string]any{"filename": "install.iso.incomplete"}) {
		t.Fatalf("cleanup params = %#v", session.calls[2].params)
	}
}

func TestManagerVirtualMediaRejectsMalformedNegotiationWithoutMounting(t *testing.T) {
	const validUploadID = "upload_12345678-1234-1234-1234-123456789abc"
	for _, test := range []struct {
		name      string
		offset    int64
		uploadID  string
		wantAbort bool
	}{
		{name: "malformed upload ID", offset: 0, uploadID: "not-an-upload-id"},
		{name: "negative offset", offset: -1, uploadID: validUploadID, wantAbort: true},
		{name: "oversized offset", offset: 9, uploadID: validUploadID, wantAbort: true},
	} {
		for _, operation := range []mcpserver.VirtualMediaOperation{mcpserver.VirtualMediaUpload, mcpserver.VirtualMediaMountFile} {
			t.Run(test.name+"/"+string(operation), func(t *testing.T) {
				mediaDirectory := t.TempDir()
				if err := os.WriteFile(filepath.Join(mediaDirectory, "install.iso"), []byte("12345678"), 0o600); err != nil {
					t.Fatal(err)
				}
				session := &fakeSession{results: map[string]any{
					"listStorageFiles":       map[string]any{"files": []any{}},
					"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": test.offset, "dataChannel": test.uploadID},
				}}
				manager := mediaManager(t, session, mediaDirectory)
				_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: operation, Source: "install.iso"})
				if !errors.Is(err, ErrInvalidResponse) {
					t.Fatalf("error = %v, want invalid negotiation", err)
				}
				wantMethods := []string{"listStorageFiles", "startStorageFileUpload", "deleteStorageFile"}
				if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
					t.Fatalf("methods = %v, want %v", got, wantMethods)
				}
				if gotAbort := len(session.uploads) == 1 && session.uploads[0].size == 0; gotAbort != test.wantAbort {
					t.Fatalf("abort uploads = %#v, wantAbort=%v", session.uploads, test.wantAbort)
				}
			})
		}
	}
}

func TestManagerVirtualMediaPendingReleaseTimeoutDoesNotCancelPartialDeletion(t *testing.T) {
	mediaDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(mediaDirectory, "install.iso"), []byte("release timeout fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	partialDeleteSawFreshContext := false
	session := &fakeSession{results: map[string]any{
		"listStorageFiles":       map[string]any{"files": []any{}},
		"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 1, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
	}}
	session.uploadFunc = func(ctx context.Context, _ string, _ io.Reader, _ int64) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("pending-upload release has no deadline")
			return errors.New("missing cleanup deadline")
		}
		<-ctx.Done()
		return ctx.Err()
	}
	session.callHook = func(ctx context.Context, method string, params any) error {
		if method == "deleteStorageFile" && reflect.DeepEqual(params, map[string]any{"filename": "install.iso.incomplete"}) {
			partialDeleteSawFreshContext = ctx.Err() == nil
		}
		return nil
	}
	manager := mediaManager(t, session, mediaDirectory)
	_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaMountFile, Source: "install.iso"})
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("error = %v, want invalid negotiation", err)
	}
	if !partialDeleteSawFreshContext {
		t.Fatal("partial deletion did not receive an independent cleanup context")
	}
}

func TestManagerVirtualMediaPartialDeleteTimeoutDoesNotCancelCompletedDeletion(t *testing.T) {
	mediaDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(mediaDirectory, "install.iso"), []byte("delete timeout fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	uploadErr := errors.New("upload interrupted")
	completedDeleteSawFreshContext := false
	session := &fakeSession{
		results: map[string]any{
			"listStorageFiles":       map[string]any{"files": []any{}},
			"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
		},
		uploadErr: uploadErr,
	}
	session.callHook = func(ctx context.Context, method string, params any) error {
		if method != "deleteStorageFile" {
			return nil
		}
		switch {
		case reflect.DeepEqual(params, map[string]any{"filename": "install.iso.incomplete"}):
			if _, ok := ctx.Deadline(); !ok {
				return errors.New("partial deletion has no cleanup deadline")
			}
			<-ctx.Done()
			return ctx.Err()
		case reflect.DeepEqual(params, map[string]any{"filename": "install.iso"}):
			completedDeleteSawFreshContext = ctx.Err() == nil
		}
		return nil
	}
	manager := mediaManager(t, session, mediaDirectory)
	_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaUpload, Source: "install.iso"})
	if !errors.Is(err, uploadErr) {
		t.Fatalf("error = %v, want upload failure", err)
	}
	if !completedDeleteSawFreshContext {
		t.Fatal("completed-file deletion did not receive an independent cleanup context")
	}
}

func TestManagerVirtualMediaDeletesPartialAndCompletedArtifactsAfterUploadFailure(t *testing.T) {
	for _, operation := range []mcpserver.VirtualMediaOperation{mcpserver.VirtualMediaUpload, mcpserver.VirtualMediaMountFile} {
		t.Run(string(operation), func(t *testing.T) {
			mediaDirectory := t.TempDir()
			contents := []byte("upload failure fixture")
			if err := os.WriteFile(filepath.Join(mediaDirectory, "install.iso"), contents, 0o600); err != nil {
				t.Fatal(err)
			}
			uploadErr := errors.New("upload interrupted")
			session := &fakeSession{
				results: map[string]any{
					"listStorageFiles":       map[string]any{"files": []any{}},
					"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
				},
				uploadErr: uploadErr,
			}
			manager := mediaManager(t, session, mediaDirectory)
			_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: operation, Source: "install.iso"})
			if !errors.Is(err, uploadErr) {
				t.Fatalf("error = %v, want upload failure", err)
			}
			wantMethods := []string{"listStorageFiles", "startStorageFileUpload", "deleteStorageFile", "deleteStorageFile"}
			if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
				t.Fatalf("methods = %v, want %v", got, wantMethods)
			}
			wantCleanup := []any{
				map[string]any{"filename": "install.iso.incomplete"},
				map[string]any{"filename": "install.iso"},
			}
			if !reflect.DeepEqual([]any{session.calls[2].params, session.calls[3].params}, wantCleanup) {
				t.Fatalf("cleanup params = %#v, want %#v", []any{session.calls[2].params, session.calls[3].params}, wantCleanup)
			}
		})
	}
}

func TestManagerVirtualMediaDeletesUploadedArtifactsAfterMountFailure(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "RPC failure", err: errors.New("mount failed")},
		{name: "cancellation", err: context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			mediaDirectory := t.TempDir()
			if err := os.WriteFile(filepath.Join(mediaDirectory, "install.iso"), []byte("mount failure fixture"), 0o600); err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{
				results: map[string]any{
					"listStorageFiles":       map[string]any{"files": []any{}},
					"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
				},
				err: map[string]error{"mountWithStorage": test.err},
			}
			manager := mediaManager(t, session, mediaDirectory)
			_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaMountFile, Source: "install.iso"})
			if !errors.Is(err, test.err) {
				t.Fatalf("error = %v, want %v", err, test.err)
			}
			wantMethods := []string{"listStorageFiles", "startStorageFileUpload", "mountWithStorage", "deleteStorageFile", "deleteStorageFile"}
			if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
				t.Fatalf("methods = %v, want %v", got, wantMethods)
			}
		})
	}
}

func TestManagerVirtualMediaDeletesChangedUploadBeforeMountOrReturn(t *testing.T) {
	for _, operation := range []mcpserver.VirtualMediaOperation{mcpserver.VirtualMediaMountFile, mcpserver.VirtualMediaUpload} {
		t.Run(string(operation), func(t *testing.T) {
			mediaDirectory := t.TempDir()
			mediaPath := filepath.Join(mediaDirectory, "install.iso")
			if err := os.WriteFile(mediaPath, []byte("original media"), 0o600); err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{results: map[string]any{
				"listStorageFiles":       map[string]any{"files": []any{}},
				"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
			}}
			session.uploadHook = func() {
				if err := os.WriteFile(mediaPath, []byte("changed media contents"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			manager := mediaManager(t, session, mediaDirectory)
			_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: operation, Source: "install.iso"})
			if !errors.Is(err, ErrMediaPath) {
				t.Fatalf("error = %v, want changed-media failure", err)
			}
			wantMethods := []string{"listStorageFiles", "startStorageFileUpload", "deleteStorageFile", "deleteStorageFile"}
			if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
				t.Fatalf("methods = %v, want %v", got, wantMethods)
			}
			wantCleanup := []any{
				map[string]any{"filename": "install.iso.incomplete"},
				map[string]any{"filename": "install.iso"},
			}
			if !reflect.DeepEqual([]any{session.calls[2].params, session.calls[3].params}, wantCleanup) {
				t.Fatalf("cleanup params = %#v, want %#v", []any{session.calls[2].params, session.calls[3].params}, wantCleanup)
			}
		})
	}
}

func TestManagerVirtualMediaDetectsAtomicSourceReplacementBeforeMountOrReturn(t *testing.T) {
	for _, operation := range []mcpserver.VirtualMediaOperation{mcpserver.VirtualMediaMountFile, mcpserver.VirtualMediaUpload} {
		t.Run(string(operation), func(t *testing.T) {
			mediaDirectory := t.TempDir()
			mediaPath := filepath.Join(mediaDirectory, "install.iso")
			if err := os.WriteFile(mediaPath, []byte("original media"), 0o600); err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{results: map[string]any{
				"listStorageFiles":       map[string]any{"files": []any{}},
				"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
			}}
			session.uploadHook = func() {
				replacement := filepath.Join(mediaDirectory, "replacement.iso")
				if err := os.WriteFile(replacement, []byte("replacement data"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(replacement, mediaPath); err != nil {
					t.Fatal(err)
				}
			}
			manager := mediaManager(t, session, mediaDirectory)
			_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: operation, Source: "install.iso"})
			if !errors.Is(err, ErrMediaPath) {
				t.Fatalf("error = %v, want replaced-media failure", err)
			}
			wantMethods := []string{"listStorageFiles", "startStorageFileUpload", "deleteStorageFile", "deleteStorageFile"}
			if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
				t.Fatalf("methods = %v, want %v", got, wantMethods)
			}
		})
	}
}

func TestManagerVirtualMediaDetectsSameSizeRewriteWithRestoredModTime(t *testing.T) {
	for _, operation := range []mcpserver.VirtualMediaOperation{mcpserver.VirtualMediaMountFile, mcpserver.VirtualMediaUpload} {
		t.Run(string(operation), func(t *testing.T) {
			mediaDirectory := t.TempDir()
			mediaPath := filepath.Join(mediaDirectory, "install.iso")
			if err := os.WriteFile(mediaPath, []byte("AAAAAAAA"), 0o600); err != nil {
				t.Fatal(err)
			}
			original, err := os.Stat(mediaPath)
			if err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{results: map[string]any{
				"listStorageFiles":       map[string]any{"files": []any{}},
				"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
			}}
			session.uploadHook = func() {
				if err := os.WriteFile(mediaPath, []byte("BBBBBBBB"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(mediaPath, original.ModTime(), original.ModTime()); err != nil {
					t.Fatal(err)
				}
			}
			manager := mediaManager(t, session, mediaDirectory)
			_, err = manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: operation, Source: "install.iso"})
			if !errors.Is(err, ErrMediaPath) {
				t.Fatalf("error = %v, want content-change failure", err)
			}
			wantMethods := []string{"listStorageFiles", "startStorageFileUpload", "deleteStorageFile", "deleteStorageFile"}
			if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
				t.Fatalf("methods = %v, want %v", got, wantMethods)
			}
		})
	}
}

func TestManagerVirtualMediaConfinesLocalPaths(t *testing.T) {
	parent := t.TempDir()
	mediaDirectory := filepath.Join(parent, "media")
	if err := os.Mkdir(mediaDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "outside.iso"), []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "outside.iso"), filepath.Join(mediaDirectory, "escape.iso")); err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{results: map[string]any{}}
	manager := mediaManager(t, session, mediaDirectory)
	for _, source := range []string{"../outside.iso", filepath.Join(parent, "outside.iso"), "escape.iso"} {
		if _, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaUpload, Source: source}); !errors.Is(err, ErrMediaPath) {
			t.Fatalf("source %q error = %v", source, err)
		}
	}
	if len(session.calls) != 0 || len(session.uploads) != 0 {
		t.Fatalf("calls=%#v uploads=%#v", session.calls, session.uploads)
	}
}

func TestManagerVirtualMediaRejectsEmptyLocalFileBeforeDeviceCall(t *testing.T) {
	for _, operation := range []mcpserver.VirtualMediaOperation{mcpserver.VirtualMediaUpload, mcpserver.VirtualMediaMountFile} {
		t.Run(string(operation), func(t *testing.T) {
			mediaDirectory := t.TempDir()
			if err := os.WriteFile(filepath.Join(mediaDirectory, "empty.iso"), nil, 0o600); err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{results: map[string]any{}}
			manager := mediaManager(t, session, mediaDirectory)
			_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: operation, Source: "empty.iso"})
			if !errors.Is(err, ErrMediaPath) {
				t.Fatalf("error = %v, want empty-media rejection", err)
			}
			if len(session.calls) != 0 || len(session.uploads) != 0 {
				t.Fatalf("calls=%#v uploads=%#v, want no device activity", session.calls, session.uploads)
			}
		})
	}
}

func TestManagerVirtualMediaHonorsCancellationBeforeLocalHashing(t *testing.T) {
	mediaDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(mediaDirectory, "install.iso"), []byte("cancellation fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{results: map[string]any{
		"listStorageFiles":       map[string]any{"files": []any{}},
		"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
	}}
	manager := mediaManager(t, session, mediaDirectory)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := manager.VirtualMedia(ctx, "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaUpload, Source: "install.iso"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want cancellation", err)
	}
	if len(session.calls) != 0 || len(session.uploads) != 0 {
		t.Fatalf("calls=%#v uploads=%#v, want no device activity", session.calls, session.uploads)
	}
}

func TestHashMediaReaderHonorsCancellationDuringActivePass(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := &gatedHashReader{
		started: make(chan struct{}),
		release: make(chan struct{}),
		data:    []byte("first hash chunk"),
	}
	result := make(chan error, 1)
	go func() {
		_, err := hashMediaReader(ctx, reader, int64(len(reader.data)+1))
		result <- err
	}()
	<-reader.started
	cancel()
	close(reader.release)
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("hash error = %v, want cancellation", err)
	}
}

func TestManagerVirtualMediaPostUploadCancellationCleansArtifacts(t *testing.T) {
	mediaDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(mediaDirectory, "install.iso"), []byte("post-upload cancellation fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	session := &fakeSession{results: map[string]any{
		"listStorageFiles":       map[string]any{"files": []any{}},
		"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
	}}
	session.uploadHook = cancel
	manager := mediaManager(t, session, mediaDirectory)
	_, err := manager.VirtualMedia(ctx, "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaMountFile, Source: "install.iso"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want cancellation", err)
	}
	wantMethods := []string{"listStorageFiles", "startStorageFileUpload", "deleteStorageFile", "deleteStorageFile"}
	if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
		t.Fatalf("methods = %v, want %v", got, wantMethods)
	}
}

func TestManagerVirtualMediaRejectsTamperedUploadStreamEvenWhenPathIsRestored(t *testing.T) {
	mediaDirectory := t.TempDir()
	mediaPath := filepath.Join(mediaDirectory, "install.iso")
	originalContents := []byte("AAAAAAAA")
	if err := os.WriteFile(mediaPath, originalContents, 0o600); err != nil {
		t.Fatal(err)
	}
	originalInfo, err := os.Stat(mediaPath)
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{results: map[string]any{
		"listStorageFiles":       map[string]any{"files": []any{}},
		"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
	}}
	session.uploadFunc = func(_ context.Context, _ string, reader io.Reader, _ int64) error {
		if err := os.WriteFile(mediaPath, []byte("BBBBBBBB"), 0o600); err != nil {
			t.Fatal(err)
		}
		uploaded, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(uploaded, []byte("BBBBBBBB")) {
			t.Fatalf("uploaded bytes = %q, want tampered stream", uploaded)
		}
		if err := os.WriteFile(mediaPath, originalContents, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(mediaPath, originalInfo.ModTime(), originalInfo.ModTime()); err != nil {
			t.Fatal(err)
		}
		return nil
	}
	manager := mediaManager(t, session, mediaDirectory)
	_, err = manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaMountFile, Source: "install.iso"})
	if !errors.Is(err, ErrMediaPath) {
		t.Fatalf("error = %v, want uploaded-stream integrity failure", err)
	}
	wantMethods := []string{"listStorageFiles", "startStorageFileUpload", "deleteStorageFile", "deleteStorageFile"}
	if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
		t.Fatalf("methods = %v, want %v", got, wantMethods)
	}
}

type gatedHashReader struct {
	started chan struct{}
	release chan struct{}
	data    []byte
	sent    bool
}

func (reader *gatedHashReader) Read(buffer []byte) (int, error) {
	if reader.sent {
		return 0, io.EOF
	}
	reader.sent = true
	close(reader.started)
	<-reader.release
	return copy(buffer, reader.data), nil
}

func TestManagerVirtualMediaSerializesOperationsPerDevice(t *testing.T) {
	mediaDirectory := t.TempDir()
	for _, name := range []string{"one.iso", "two.iso"} {
		if err := os.WriteFile(filepath.Join(mediaDirectory, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	provider := &concurrentMediaProvider{}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base, MediaDirectory: mediaDirectory}}, provider)
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	errorsSeen := make(chan error, 2)
	for _, source := range []string{"one.iso", "two.iso"} {
		source := source
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaUpload, Source: source})
			errorsSeen <- err
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal(err)
		}
	}
	if maximum := provider.max.Load(); maximum != 1 {
		t.Fatalf("concurrent virtual-media sessions = %d, want 1", maximum)
	}
}

func mediaManager(t *testing.T, session *fakeSession, directory string) *Manager {
	t.Helper()
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base, MediaDirectory: directory}}, &fakeProvider{session: session})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}
