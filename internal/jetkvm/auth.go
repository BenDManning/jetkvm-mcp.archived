package jetkvm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxHTTPBody int64 = 1 << 20

type deviceIdentity struct {
	AuthMode     string `json:"authMode"`
	DeviceID     string `json:"deviceId"`
	LoopbackOnly bool   `json:"loopbackOnly"`
}

type loginRequest struct {
	Password string `json:"password"`
}

func authenticate(ctx context.Context, client *http.Client, baseURL url.URL, password string) (deviceIdentity, error) {
	identity, status, err := probeDevice(ctx, client, baseURL)
	if err != nil {
		return deviceIdentity{}, err
	}
	if status == http.StatusOK {
		return identity, nil
	}
	if status != http.StatusUnauthorized {
		return deviceIdentity{}, fmt.Errorf("%w: device probe returned HTTP %d", ErrDeviceUnreachable, status)
	}
	if password == "" {
		return deviceIdentity{}, ErrAuthentication
	}

	payload, err := json.Marshal(loginRequest{Password: password})
	if err != nil {
		return deviceIdentity{}, fmt.Errorf("%w: encode login", ErrProtocol)
	}
	loginURL := endpoint(baseURL, "/auth/login-local")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL.String(), bytes.NewReader(payload))
	if err != nil {
		return deviceIdentity{}, fmt.Errorf("%w: create login request", ErrProtocol)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return deviceIdentity{}, fmt.Errorf("%w: login request", ErrDeviceUnreachable)
	}
	_, readErr := readBounded(response.Body, maxHTTPBody)
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		return deviceIdentity{}, fmt.Errorf("%w: invalid login response", ErrAuthentication)
	}
	if response.StatusCode != http.StatusOK {
		return deviceIdentity{}, fmt.Errorf("%w: login returned HTTP %d", ErrAuthentication, response.StatusCode)
	}

	identity, status, err = probeDevice(ctx, client, baseURL)
	if err != nil {
		return deviceIdentity{}, fmt.Errorf("%w: post-login probe: %w", ErrAuthentication, err)
	}
	if status != http.StatusOK {
		return deviceIdentity{}, fmt.Errorf("%w: post-login probe returned HTTP %d", ErrAuthentication, status)
	}
	return identity, nil
}

func probeDevice(ctx context.Context, client *http.Client, baseURL url.URL) (deviceIdentity, int, error) {
	requestURL := endpoint(baseURL, "/device")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return deviceIdentity{}, 0, fmt.Errorf("%w: create device probe", ErrProtocol)
	}
	response, err := client.Do(request)
	if err != nil {
		return deviceIdentity{}, 0, fmt.Errorf("%w: device probe", ErrDeviceUnreachable)
	}
	body, readErr := readBounded(response.Body, maxHTTPBody)
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		return deviceIdentity{}, response.StatusCode, fmt.Errorf("%w: device probe body", ErrInvalidResponse)
	}
	if response.StatusCode != http.StatusOK {
		return deviceIdentity{}, response.StatusCode, nil
	}
	var identity deviceIdentity
	if err := json.Unmarshal(body, &identity); err != nil || strings.TrimSpace(identity.DeviceID) == "" {
		return deviceIdentity{}, response.StatusCode, fmt.Errorf("%w: device identity", ErrInvalidResponse)
	}
	return identity, response.StatusCode, nil
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	if limit < 0 {
		return nil, errors.New("negative read limit")
	}
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, io.ErrUnexpectedEOF
	}
	return body, nil
}

func endpoint(base url.URL, suffix string) url.URL {
	result := base
	result.Path = strings.TrimRight(base.Path, "/") + suffix
	result.RawPath = ""
	result.RawQuery = ""
	result.Fragment = ""
	return result
}
