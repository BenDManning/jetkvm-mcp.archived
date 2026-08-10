package httporigin

import (
	"errors"
	"testing"
)

func TestParseAcceptsCanonicalHTTPOrigins(t *testing.T) {
	for _, test := range []struct {
		input string
		want  Origin
	}{
		{input: "http://localhost:8080", want: Origin{Value: "http://localhost:8080", Scheme: "http", Host: "localhost:8080"}},
		{input: "https://MCP.EXAMPLE.INVALID", want: Origin{Value: "https://mcp.example.invalid", Scheme: "https", Host: "mcp.example.invalid"}},
		{input: "https://[::1]:8443", want: Origin{Value: "https://[::1]:8443", Scheme: "https", Host: "[::1]:8443"}},
		{input: "https://mcp.example.invalid:1", want: Origin{Value: "https://mcp.example.invalid:1", Scheme: "https", Host: "mcp.example.invalid:1"}},
		{input: "https://mcp.example.invalid:65535", want: Origin{Value: "https://mcp.example.invalid:65535", Scheme: "https", Host: "mcp.example.invalid:65535"}},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, err := Parse(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("origin = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseRejectsNonOrigins(t *testing.T) {
	for _, input := range []string{
		"",
		"mcp.example.invalid",
		"ftp://mcp.example.invalid",
		"https://",
		"https://user:password@mcp.example.invalid",
		"https://mcp.example.invalid/",
		"https://mcp.example.invalid/path",
		"https://mcp.example.invalid?",
		"https://mcp.example.invalid?query",
		"https://mcp.example.invalid#",
		"https://mcp.example.invalid#fragment",
		"https://mcp.example.invalid:0",
		"https://mcp.example.invalid:65536",
		"https://mcp.example.invalid:invalid",
		"https://mcp.example.invalid:+80",
		"https://mcp.example.invalid:-1",
		"https://mcp.example.invalid:",
		"https://[::1]:",
		"http://::1",
	} {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input)
			if err == nil {
				t.Fatal("invalid origin accepted")
			}
			if input == "https://user:password@mcp.example.invalid" && !errors.Is(err, ErrCredentials) {
				t.Fatalf("error = %v, want credential classification", err)
			}
		})
	}
}

func TestParseAuthority(t *testing.T) {
	for input, want := range map[string]string{
		"localhost":       "localhost",
		"LOCALHOST:8080":  "localhost:8080",
		"127.0.0.1":       "127.0.0.1",
		"127.0.0.1:65535": "127.0.0.1:65535",
		"[::1]":           "[::1]",
		"[::1]:1":         "[::1]:1",
	} {
		t.Run("accept_"+input, func(t *testing.T) {
			got, err := ParseAuthority(input)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("authority = %q, want %q", got, want)
			}
		})
	}
	for _, input := range []string{
		"",
		" localhost",
		"localhost ",
		"localhost:",
		"127.0.0.1:",
		"[::1]:",
		"::1",
		"user@localhost",
		"localhost/",
		"localhost?",
		"localhost:0",
		"localhost:65536",
	} {
		t.Run("reject_"+input, func(t *testing.T) {
			if _, err := ParseAuthority(input); err == nil {
				t.Fatal("invalid authority accepted")
			}
		})
	}
}
