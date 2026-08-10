package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestParseArgsSelectsStdioHTTPAndDebugRPC(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		kind    commandKind
		address string
		method  string
	}{
		{name: "stdio", args: []string{"--config", "config.yaml"}, kind: commandServe},
		{name: "http", args: []string{"--config", "config.yaml", "--http", "127.0.0.1:9000"}, kind: commandServe, address: "127.0.0.1:9000"},
		{name: "debug", args: []string{"debug", "rpc", "--config", "config.yaml", "--device", "lab", "--method", "ping", "--params", "{}"}, kind: commandDebugRPC, method: "ping"},
	} {
		t.Run(test.name, func(t *testing.T) {
			options, err := parseArgs(test.args)
			if err != nil {
				t.Fatal(err)
			}
			if options.kind != test.kind || options.httpAddress != test.address || options.debugMethod != test.method || options.configPath != "config.yaml" {
				t.Fatalf("options = %+v", options)
			}
		})
	}
}

func TestParseArgsRejectsDeprecatedOrIncompleteCommands(t *testing.T) {
	for _, args := range [][]string{
		{"--sse", "127.0.0.1:9000"},
		{"debug", "rpc", "--device", "lab", "--method", "ping"},
		{"debug", "rpc", "--config", "config.yaml", "--method", "ping"},
		{"debug", "rpc", "--config", "config.yaml", "--device", "lab"},
		{"debug", "other"},
	} {
		if _, err := parseArgs(args); err == nil {
			t.Fatalf("parseArgs(%v) error = nil", args)
		}
	}
}

func TestRunVersionDoesNotRequireConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run(context.Background(), []string{"--version"}, strings.NewReader(""), &stdout, &stderr, nil); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout.String()) == "" || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
