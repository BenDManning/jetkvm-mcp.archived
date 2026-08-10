package jetkvm

import (
	"context"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func TestAuthenticateNoPasswordAndPasswordModes(t *testing.T) {
	t.Run("no password", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if request.Method != http.MethodGet || request.URL.Path != "/device" {
				t.Fatalf("request = %s %s", request.Method, request.URL.Path)
			}
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write([]byte(`{"authMode":"noPassword","deviceId":"device-1","loopbackOnly":true}`))
		}))
		defer server.Close()

		identity, err := authenticate(context.Background(), clientWithCookies(t), mustURL(t, server.URL), "")
		if err != nil {
			t.Fatal(err)
		}
		if identity.DeviceID != "device-1" || identity.AuthMode != "noPassword" || !identity.LoopbackOnly {
			t.Fatalf("identity = %+v", identity)
		}
	})

	t.Run("password", func(t *testing.T) {
		const password = "correct horse battery staple"
		var mu sync.Mutex
		loggedIn := false
		loginCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			switch request.URL.Path {
			case "/device":
				cookie, _ := request.Cookie("session")
				if !loggedIn || cookie == nil || cookie.Value != "ok" {
					http.Error(response, "unauthorized", http.StatusUnauthorized)
					return
				}
				_, _ = response.Write([]byte(`{"authMode":"password","deviceId":"device-2"}`))
			case "/auth/login-local":
				loginCalls++
				if request.Method != http.MethodPost || request.Header.Get("Content-Type") != "application/json" {
					t.Fatalf("login request = %s content-type=%q", request.Method, request.Header.Get("Content-Type"))
				}
				body := make([]byte, request.ContentLength)
				_, _ = request.Body.Read(body)
				if string(body) != `{"password":"`+password+`"}` {
					t.Fatalf("login body = %q", body)
				}
				loggedIn = true
				http.SetCookie(response, &http.Cookie{Name: "session", Value: "ok", Path: "/"})
				response.WriteHeader(http.StatusOK)
			default:
				http.NotFound(response, request)
			}
		}))
		defer server.Close()

		identity, err := authenticate(context.Background(), clientWithCookies(t), mustURL(t, server.URL), password)
		if err != nil {
			t.Fatal(err)
		}
		if identity.DeviceID != "device-2" || loginCalls != 1 {
			t.Fatalf("identity=%+v loginCalls=%d", identity, loginCalls)
		}
	})
}

func TestAuthenticateRejectsMissingPasswordAndInvalidDeviceResponse(t *testing.T) {
	t.Run("missing password", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			http.Error(response, "unauthorized", http.StatusUnauthorized)
		}))
		defer server.Close()
		_, err := authenticate(context.Background(), clientWithCookies(t), mustURL(t, server.URL), "")
		if !errors.Is(err, ErrAuthentication) {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("invalid response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			_, _ = response.Write([]byte(`{"deviceId":""}`))
		}))
		defer server.Close()
		_, err := authenticate(context.Background(), clientWithCookies(t), mustURL(t, server.URL), "unused")
		if !errors.Is(err, ErrInvalidResponse) {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("bounded response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			_, _ = response.Write([]byte(strings.Repeat("x", int(maxHTTPBody)+1)))
		}))
		defer server.Close()
		_, err := authenticate(context.Background(), clientWithCookies(t), mustURL(t, server.URL), "unused")
		if !errors.Is(err, ErrInvalidResponse) {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestAuthenticateNeverForwardsPasswordThroughRedirects(t *testing.T) {
	var redirected bool
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected = true
	}))
	defer target.Close()
	device := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/device":
			writer.WriteHeader(http.StatusUnauthorized)
		case "/auth/login-local":
			http.Redirect(writer, request, target.URL, http.StatusTemporaryRedirect)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer device.Close()
	base, _ := url.Parse(device.URL)
	client, err := newDeviceHTTPClient(DeviceConfig{BaseURL: *base})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authenticate(context.Background(), client, *base, "test password"); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("error = %v, want ErrAuthentication", err)
	}
	if redirected {
		t.Fatal("login password was forwarded through an HTTP redirect")
	}
}

func clientWithCookies(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar}
}

func mustURL(t *testing.T, raw string) url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return *parsed
}
