package jetkvm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestConnectedSessionUploadUsesAuthenticatedBoundedHTTP(t *testing.T) {
	payload := []byte("remaining upload bytes")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/storage/upload" || request.URL.Query().Get("uploadId") != "upload_12345678-1234-1234-1234-123456789abc" {
			t.Errorf("request URL = %s", request.URL)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		cookie, err := request.Cookie("session")
		if err != nil || cookie.Value != "ok" {
			t.Errorf("cookie = %v err=%v", cookie, err)
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
		}
		if request.ContentLength != int64(len(payload)) || request.Header.Get("Content-Type") != "application/octet-stream" || !bytes.Equal(body, payload) {
			t.Errorf("length=%d type=%q body=%q", request.ContentLength, request.Header.Get("Content-Type"), body)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	base, _ := url.Parse(server.URL)
	jar, _ := cookiejar.New(nil)
	jar.SetCookies(base, []*http.Cookie{{Name: "session", Value: "ok", Path: "/"}})
	session := &connectedSession{httpClient: &http.Client{Jar: jar}, baseURL: *base}
	if err := session.Upload(context.Background(), "upload_12345678-1234-1234-1234-123456789abc", bytes.NewReader(payload), int64(len(payload))); err != nil {
		t.Fatal(err)
	}
}

func TestConnectedSessionUploadRejectsInvalidIDAndHTTPFailure(t *testing.T) {
	session := &connectedSession{}
	if err := session.Upload(context.Background(), "bad", bytes.NewReader(nil), 0); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("invalid ID error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte("private server detail"))
	}))
	defer server.Close()
	base, _ := url.Parse(server.URL)
	session = &connectedSession{httpClient: server.Client(), baseURL: *base}
	if err := session.Upload(context.Background(), "upload_12345678-1234-1234-1234-123456789abc", bytes.NewReader(nil), 0); !errors.Is(err, ErrProtocol) {
		t.Fatalf("HTTP error = %v", err)
	}
}
