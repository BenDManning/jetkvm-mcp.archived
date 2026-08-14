package httporigin

import (
	"net/url"
	"strings"
	"testing"
)

const maxFuzzOriginBytes = 4 << 10

func FuzzParseOrigin(f *testing.F) {
	for _, seed := range []string{
		"https://example.invalid",
		"HTTP://EXAMPLE.INVALID:80",
		"https://[2001:db8::1]:443",
		"http://127.0.0.1:8080",
		"https://user:password@example.invalid",
		"https://example.invalid/path",
		"https://example.invalid?query",
		"https://example.invalid#fragment",
		" https://example.invalid ",
		"example.invalid",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > maxFuzzOriginBytes {
			value = value[:maxFuzzOriginBytes]
		}
		origin, err := Parse(value)
		if err != nil {
			return
		}
		parsed, parseErr := url.Parse(origin.Value)
		if parseErr != nil || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || origin.Scheme == "" || origin.Host == "" {
			t.Fatalf("accepted unsafe canonical origin %q", origin.Value)
		}
		roundTrip, err := Parse(origin.Value)
		if err != nil || roundTrip != origin {
			t.Fatalf("canonical origin did not round-trip")
		}
		effective, err := ParseEffective(value)
		if err != nil {
			t.Fatalf("effective parse rejected an accepted origin")
		}
		effectiveRoundTrip, err := ParseEffective(effective.Value)
		if err != nil || effectiveRoundTrip != effective {
			t.Fatalf("effective origin did not round-trip")
		}
	})
}

func FuzzParseAuthority(f *testing.F) {
	for _, seed := range []string{
		"example.invalid",
		"EXAMPLE.INVALID:443",
		"127.0.0.1:8080",
		"[2001:db8::1]:8443",
		"2001:db8::1",
		"example.invalid:",
		"user@example.invalid",
		" example.invalid",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > maxFuzzOriginBytes {
			value = value[:maxFuzzOriginBytes]
		}
		authority, err := ParseAuthority(value)
		if err != nil {
			return
		}
		if strings.ContainsAny(authority, "@/?#") || strings.TrimSpace(authority) != authority || authority == "" {
			t.Fatalf("accepted unsafe canonical authority %q", authority)
		}
		roundTrip, err := ParseAuthority(authority)
		if err != nil || roundTrip != authority {
			t.Fatalf("canonical authority did not round-trip")
		}
		origin, err := Parse("http://" + authority)
		if err != nil || origin.Host != authority {
			t.Fatalf("authority and origin canonicalization differ")
		}
	})
}
