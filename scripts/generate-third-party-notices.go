//go:build ignore

// generate-third-party-notices emits the license and notice files belonging to
// the external modules compiled into a named Go package.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type listedPackage struct {
	Module *listedModule
}

type listedModule struct {
	Path    string
	Version string
	Dir     string
	Main    bool
	Replace *listedModule
}

type moduleNotice struct {
	path    string
	version string
	dir     string
	files   []string
}

func main() {
	checkPath := flag.String("check", "", "fail unless this file matches generated notices")
	outputPath := flag.String("output", "", "write generated notices to this file")
	flag.Parse()
	if flag.NArg() != 1 || (*checkPath != "" && *outputPath != "") {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/generate-third-party-notices.go [-check FILE | -output FILE] PACKAGE")
		os.Exit(2)
	}

	generated, err := generate(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *checkPath != "" {
		current, err := os.ReadFile(*checkPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !bytes.Equal(current, generated) {
			fmt.Fprintf(os.Stderr, "%s is stale; regenerate it with -output %s\n", *checkPath, *checkPath)
			os.Exit(1)
		}
		return
	}
	if *outputPath != "" {
		if err := os.WriteFile(*outputPath, generated, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if _, err := os.Stdout.Write(generated); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(packageName string) ([]byte, error) {
	modules := make(map[string]moduleNotice)
	for _, arch := range []string{"amd64", "arm64"} {
		command := exec.Command("go", "list", "-deps", "-json", packageName)
		command.Env = append(os.Environ(), "GOWORK=off", "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
		output, err := command.Output()
		if err != nil {
			return nil, fmt.Errorf("list linux/%s release package closure: %w", arch, err)
		}

		decoder := json.NewDecoder(bytes.NewReader(output))
		for {
			var pkg listedPackage
			if err := decoder.Decode(&pkg); err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("decode linux/%s package closure: %w", arch, err)
			}
			if pkg.Module == nil || pkg.Module.Main {
				continue
			}
			module := pkg.Module
			source := module
			if module.Replace != nil {
				source = module.Replace
			}
			key := module.Path + "@" + module.Version
			modules[key] = moduleNotice{path: module.Path, version: module.Version, dir: source.Dir}
		}
	}

	keys := make([]string, 0, len(modules))
	for key := range modules {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		module := modules[key]
		entries, err := os.ReadDir(module.dir)
		if err != nil {
			return nil, fmt.Errorf("read %s license directory: %w", module.path, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !isNoticeFile(entry.Name()) {
				continue
			}
			module.files = append(module.files, entry.Name())
		}
		sort.Strings(module.files)
		if len(module.files) == 0 {
			return nil, fmt.Errorf("external module %s has no top-level license or notice file", key)
		}
		modules[key] = module
	}

	var result bytes.Buffer
	result.WriteString("# Third-party licenses and notices\n\n")
	result.WriteString("This file is generated from the union of external Go module package closures compiled into the Linux amd64 and arm64 `./cmd/jetkvm-mcp` binaries. Regenerate it with:\n\n")
	result.WriteString("```sh\n")
	result.WriteString("go run ./scripts/generate-third-party-notices.go -output THIRD_PARTY_NOTICES.md ./cmd/jetkvm-mcp\n")
	result.WriteString("```\n")
	for _, key := range keys {
		module := modules[key]
		fmt.Fprintf(&result, "\n## %s %s\n", module.path, module.version)
		for _, name := range module.files {
			contents, err := os.ReadFile(filepath.Join(module.dir, name))
			if err != nil {
				return nil, fmt.Errorf("read %s %s: %w", module.path, name, err)
			}
			fmt.Fprintf(&result, "\n### %s\n\n```text\n", name)
			result.Write(bytes.TrimSpace(contents))
			result.WriteString("\n```\n")
		}
	}

	return result.Bytes(), nil
}

func isNoticeFile(name string) bool {
	upper := strings.ToUpper(name)
	return strings.HasPrefix(upper, "LICENSE") || strings.HasPrefix(upper, "COPYING") || strings.HasPrefix(upper, "NOTICE")
}
