package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
)

const expectedRepository = "https://github.com/jetkvm/kvm"

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

var expectedSurfaces = []driftSurface{
	{Name: "auth", Paths: []string{"web.go", "ui/src/routes/devices.$id.tsx"}},
	{Name: "signaling", Paths: []string{"web.go", "ui/src/routes/devices.$id.tsx"}},
	{Name: "rpc", Paths: []string{"jsonrpc.go", "ota.go"}},
	{Name: "video", Paths: []string{"video.go"}},
	{Name: "hid", Paths: []string{"usb.go"}},
	{Name: "virtual_media", Paths: []string{"usb_mass_storage.go", "ui/src/routes/devices.$id.mount.tsx"}},
}

type driftManifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	Repository    string         `json:"repository"`
	PinnedCommit  string         `json:"pinnedCommit"`
	Surfaces      []driftSurface `json:"surfaces"`
}

type driftSurface struct {
	Name  string   `json:"name"`
	Paths []string `json:"paths"`
}

type driftReport struct {
	SchemaVersion   int              `json:"schemaVersion"`
	Repository      string           `json:"repository"`
	PinnedCommit    string           `json:"pinnedCommit"`
	TargetCommit    string           `json:"targetCommit"`
	Result          string           `json:"result"`
	ChangedSurfaces []changedSurface `json:"changedSurfaces"`
}

type changedSurface struct {
	Name  string   `json:"name"`
	Paths []string `json:"paths"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("jetkvm-mcp-upstream-drift", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	upstream := flags.String("upstream", "", "local JetKVM upstream Git checkout")
	manifestPath := flags.String("manifest", "docs/compatibility/jetkvm-upstream-surfaces.json", "reviewed surface manifest")
	target := flags.String("target", "HEAD", "upstream target commit or ref")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || strings.TrimSpace(*upstream) == "" || strings.TrimSpace(*manifestPath) == "" || strings.TrimSpace(*target) == "" {
		fmt.Fprintln(stderr, "jetkvm-upstream-drift: invalid arguments")
		return 2
	}

	manifest, err := loadManifest(*manifestPath)
	if err != nil {
		fmt.Fprintln(stderr, "jetkvm-upstream-drift: invalid manifest")
		return 2
	}
	pin, err := resolveCommit(*upstream, manifest.PinnedCommit)
	if err != nil || pin != manifest.PinnedCommit {
		fmt.Fprintln(stderr, "jetkvm-upstream-drift: reviewed pin is unavailable")
		return 2
	}
	targetCommit, err := resolveCommit(*upstream, *target)
	if err != nil {
		fmt.Fprintln(stderr, "jetkvm-upstream-drift: target ref is unavailable")
		return 2
	}

	paths := uniqueManifestPaths(manifest)
	changed, err := gitChangedPaths(*upstream, pin, targetCommit, paths)
	if err != nil {
		fmt.Fprintln(stderr, "jetkvm-upstream-drift: comparison failed")
		return 2
	}
	changedSet := make(map[string]bool, len(changed))
	for _, name := range changed {
		changedSet[name] = true
	}
	report := driftReport{
		SchemaVersion: 1, Repository: manifest.Repository, PinnedCommit: pin,
		TargetCommit: targetCommit, Result: "no_drift", ChangedSurfaces: []changedSurface{},
	}
	for _, surface := range manifest.Surfaces {
		var surfacePaths []string
		for _, name := range surface.Paths {
			if changedSet[name] {
				surfacePaths = append(surfacePaths, name)
			}
		}
		if len(surfacePaths) > 0 {
			sort.Strings(surfacePaths)
			report.ChangedSurfaces = append(report.ChangedSurfaces, changedSurface{Name: surface.Name, Paths: surfacePaths})
		}
	}
	if len(report.ChangedSurfaces) > 0 {
		report.Result = "drift"
	}
	if err := json.NewEncoder(stdout).Encode(report); err != nil {
		fmt.Fprintln(stderr, "jetkvm-upstream-drift: report failed")
		return 2
	}
	if report.Result == "drift" {
		return 1
	}
	return 0
}

func loadManifest(name string) (driftManifest, error) {
	file, err := os.Open(name)
	if err != nil {
		return driftManifest{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 1<<20))
	decoder.DisallowUnknownFields()
	var manifest driftManifest
	if err := decoder.Decode(&manifest); err != nil {
		return driftManifest{}, err
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return driftManifest{}, err
	}
	if manifest.SchemaVersion != 1 || manifest.Repository != expectedRepository || !commitPattern.MatchString(manifest.PinnedCommit) {
		return driftManifest{}, errors.New("invalid manifest header")
	}
	if len(manifest.Surfaces) != len(expectedSurfaces) {
		return driftManifest{}, errors.New("invalid surface count")
	}
	for i, surface := range manifest.Surfaces {
		if surface.Name != expectedSurfaces[i].Name || !slices.Equal(surface.Paths, expectedSurfaces[i].Paths) {
			return driftManifest{}, errors.New("invalid surface")
		}
		seen := make(map[string]bool)
		for _, name := range surface.Paths {
			if !validUpstreamPath(name) || seen[name] {
				return driftManifest{}, errors.New("invalid surface path")
			}
			seen[name] = true
		}
	}
	return manifest, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func validUpstreamPath(name string) bool {
	return name != "" && !strings.HasPrefix(name, "-") && !strings.HasPrefix(name, ":") &&
		!strings.ContainsAny(name, "*?[") && !strings.ContainsRune(name, '\x00') &&
		path.Clean(name) == name && !path.IsAbs(name) && name != "." && !strings.HasPrefix(name, "../")
}

func uniqueManifestPaths(manifest driftManifest) []string {
	seen := make(map[string]bool)
	var paths []string
	for _, surface := range manifest.Surfaces {
		for _, name := range surface.Paths {
			if !seen[name] {
				seen[name] = true
				paths = append(paths, name)
			}
		}
	}
	sort.Strings(paths)
	return paths
}

func resolveCommit(repository, ref string) (string, error) {
	command := exec.Command("git", "-C", repository, "rev-parse", "--verify", ref+"^{commit}")
	command.Env = append(os.Environ(), "LC_ALL=C")
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	commit := strings.TrimSpace(string(output))
	if !commitPattern.MatchString(commit) {
		return "", errors.New("invalid commit")
	}
	return commit, nil
}

func gitChangedPaths(repository, pin, target string, paths []string) ([]string, error) {
	arguments := []string{"-C", repository, "diff", "--name-only", "-z", "--diff-filter=ACDMRTUXB", pin + ".." + target, "--"}
	arguments = append(arguments, paths...)
	command := exec.Command("git", arguments...)
	command.Env = append(os.Environ(), "LC_ALL=C", "GIT_LITERAL_PATHSPECS=1")
	output, err := command.Output()
	if err != nil {
		return nil, err
	}
	parts := bytes.Split(output, []byte{0})
	changed := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) > 0 {
			changed = append(changed, string(part))
		}
	}
	sort.Strings(changed)
	return changed, nil
}
