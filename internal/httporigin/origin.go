package httporigin

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

var (
	ErrCredentials = errors.New("origin must not include credentials")
	ErrInvalid     = errors.New("invalid HTTP origin")
)

type Origin struct {
	Value  string
	Scheme string
	Host   string
}

func Parse(value string) (Origin, error) {
	raw := strings.TrimSpace(value)
	parsed, err := url.Parse(raw)
	if parsed != nil && parsed.User != nil {
		return Origin{}, ErrCredentials
	}
	if err != nil || parsed == nil || parsed.Opaque != "" || parsed.Host == "" || parsed.Hostname() == "" || strings.HasSuffix(parsed.Host, ":") || unbracketedMultiColonHost(parsed.Host) || parsed.Path != "" || strings.ContainsAny(raw, "?#") {
		return Origin{}, ErrInvalid
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return Origin{}, ErrInvalid
	}
	if port := parsed.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return Origin{}, ErrInvalid
		}
	}
	host := strings.ToLower(parsed.Host)
	return Origin{Value: scheme + "://" + host, Scheme: scheme, Host: host}, nil
}

// ParseAuthority validates and canonicalizes an HTTP Host authority without
// accepting URL syntax, empty port delimiters, or unbracketed IPv6 literals.
func ParseAuthority(value string) (string, error) {
	raw := strings.TrimSpace(value)
	if raw == "" || raw != value {
		return "", ErrInvalid
	}
	origin, err := Parse("http://" + raw)
	if err != nil {
		return "", err
	}
	return origin.Host, nil
}

func unbracketedMultiColonHost(host string) bool {
	return strings.Count(host, ":") > 1 && !strings.HasPrefix(host, "[")
}
