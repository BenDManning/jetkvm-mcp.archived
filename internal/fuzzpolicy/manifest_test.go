package fuzzpolicy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type target struct {
	Package string `json:"package"`
	Target  string `json:"target"`
}

func TestDiscoverFuzzTargetsIncludesOrdinaryTestFiles(t *testing.T) {
	root := t.TempDir()
	packageDirectory := filepath.Join(root, "internal", "example")
	if err := os.MkdirAll(packageDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	source := []byte("package example\n\nimport \"testing\"\n\nfunc FuzzOrdinary(f *testing.F) {}\n")
	if err := os.WriteFile(filepath.Join(packageDirectory, "ordinary_test.go"), source, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := discoverFuzzTargets(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []target{{Package: "./internal/example", Target: "FuzzOrdinary"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordinary test-file fuzz target not discovered: got %#v, want %#v", got, want)
	}
}

func TestFuzzTargetInventoryAndCorpusPolicy(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	manifestPath := filepath.Join(root, "testdata", "fuzz-targets.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest []target
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("decode target manifest: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("target manifest must contain exactly one JSON value: %v", err)
	}
	want := append([]target(nil), manifest...)
	sortTargets(want)
	if !reflect.DeepEqual(manifest, want) {
		t.Fatal("target manifest must be sorted by package and target")
	}
	seen := make(map[target]struct{}, len(manifest))
	for _, entry := range manifest {
		if entry.Package == "" || entry.Target == "" || !strings.HasPrefix(entry.Target, "Fuzz") {
			t.Fatalf("invalid manifest entry %#v", entry)
		}
		if _, duplicate := seen[entry]; duplicate {
			t.Fatalf("duplicate manifest entry %#v", entry)
		}
		seen[entry] = struct{}{}
	}

	found, err := discoverFuzzTargets(root)
	if err != nil {
		t.Fatal(err)
	}
	sortTargets(found)
	if !reflect.DeepEqual(found, manifest) {
		t.Fatalf("fuzz target inventory drifted\nmanifest: %#v\nsource:   %#v", manifest, found)
	}

	privatePattern := regexp.MustCompile(`(?i)(/workspace/|/home/|/Users/|[A-Z]:\\|-----BEGIN [A-Z ]*PRIVATE KEY-----|https?://(?:10\.|127\.|192\.168\.|172\.(?:1[6-9]|2[0-9]|3[01])\.))`)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.Contains(filepath.ToSlash(path), "/testdata/fuzz/") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > 64<<10 {
			return fmt.Errorf("non-regular or oversized fuzz corpus file: %s", path)
		}
		corpus, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(corpus, []byte("go test fuzz v1\n")) || privatePattern.Match(corpus) {
			return fmt.Errorf("nonportable or private fuzz corpus file: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func discoverFuzzTargets(root string) ([]target, error) {
	var found []target
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || !strings.HasPrefix(function.Name.Name, "Fuzz") {
				continue
			}
			relative, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return err
			}
			found = append(found, target{Package: "./" + filepath.ToSlash(relative), Target: function.Name.Name})
		}
		return nil
	})
	return found, err
}

func sortTargets(targets []target) {
	sort.Slice(targets, func(left, right int) bool {
		if targets[left].Package == targets[right].Package {
			return targets[left].Target < targets[right].Target
		}
		return targets[left].Package < targets[right].Package
	})
}
