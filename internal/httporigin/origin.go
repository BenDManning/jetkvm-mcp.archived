package httporigin

import (
	"errors"
	"net/netip"
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
	host := escapeZone(strings.ToLower(parsed.Host))
	return Origin{Value: scheme + "://" + host, Scheme: scheme, Host: host}, nil
}

// ParseEffective canonicalizes an HTTP(S) origin by effective port. It is used
// for authorization where omitted and explicit default ports name the same
// network endpoint. Parse remains available when the serialized authority is
// intentionally significant.
func ParseEffective(value string) (Origin, error) {
	origin, err := Parse(value)
	if err != nil {
		return Origin{}, err
	}
	parsed, err := url.Parse(origin.Value)
	if err != nil {
		return Origin{}, ErrInvalid
	}
	port := parsed.Port()
	if port == "" {
		host := canonicalEffectiveHostname(parsed.Hostname())
		host = serializeHostname(host)
		return Origin{Value: origin.Scheme + "://" + host, Scheme: origin.Scheme, Host: host}, nil
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return Origin{}, ErrInvalid
	}
	if origin.Scheme != "http" || portNumber != 80 {
		if origin.Scheme != "https" || portNumber != 443 {
			host := canonicalEffectiveHostname(parsed.Hostname())
			host = serializeHostname(host)
			host = host + ":" + strconv.Itoa(portNumber)
			return Origin{Value: origin.Scheme + "://" + host, Scheme: origin.Scheme, Host: host}, nil
		}
	}
	host := canonicalEffectiveHostname(parsed.Hostname())
	host = serializeHostname(host)
	return Origin{Value: origin.Scheme + "://" + host, Scheme: origin.Scheme, Host: host}, nil
}

func canonicalEffectiveHostname(host string) string {
	address, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	return address.String()
}

func serializeHostname(host string) string {
	if strings.Contains(host, ":") {
		return "[" + escapeZone(host) + "]"
	}
	return host
}

func escapeZone(value string) string {
	return strings.ReplaceAll(value, "%", "%25")
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
