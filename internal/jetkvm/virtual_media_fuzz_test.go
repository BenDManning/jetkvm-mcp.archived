package jetkvm

import (
	"net/url"
	"strings"
	"testing"
)

const maxFuzzMediaURLBytes = 4 << 10

func FuzzParseAllowedMediaURL(f *testing.F) {
	const allowedOrigin = "https://media.example.invalid"
	for _, seed := range []string{
		allowedOrigin + "/fixture.iso",
		allowedOrigin + ":443/fixture.iso?download=1#fragment",
		"https://user:password@media.example.invalid/fixture.iso",
		"https://other.example.invalid/fixture.iso",
		"http://media.example.invalid/fixture.iso",
		"not-a-url",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > maxFuzzMediaURLBytes {
			value = value[:maxFuzzMediaURLBytes]
		}
		if accepted, err := parseAllowedMediaURL(value, nil); err == nil || accepted != nil {
			t.Fatal("empty origin allowlist did not deny by default")
		}
		accepted, err := parseAllowedMediaURL(value, []string{allowedOrigin})
		if err != nil {
			return
		}
		if accepted == nil || accepted.User != nil || accepted.Opaque != "" || accepted.Scheme != "https" || accepted.Hostname() == "" {
			t.Fatalf("accepted unsafe media URL")
		}
		origin, err := normalizeMediaURLOrigin(accepted.Scheme + "://" + accepted.Host)
		if err != nil || origin != allowedOrigin {
			t.Fatalf("accepted URL outside exact configured origin")
		}
		roundTrip, err := parseAllowedMediaURL(accepted.String(), []string{allowedOrigin})
		if err != nil || roundTrip.String() != accepted.String() {
			t.Fatal("accepted media URL did not round-trip")
		}
		parsed, err := url.Parse(strings.TrimSpace(value))
		if err != nil || parsed.String() != accepted.String() {
			t.Fatal("accepted media URL changed during parsing")
		}
	})
}
