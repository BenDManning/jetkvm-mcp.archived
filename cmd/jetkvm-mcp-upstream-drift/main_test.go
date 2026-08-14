package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunReportsNoDriftAtPinnedCommitAndIgnoresUnrelatedChanges(t *testing.T) {
	repo, pin := newUpstreamFixture(t)
	manifest := writeManifest(t, pin)
	commitFile(t, repo, "unrelated.md", "changed\n", "unrelated change")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"--upstream", repo, "--manifest", manifest, "--target", "HEAD"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run exit=%d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var report driftReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Result != "no_drift" || report.PinnedCommit != pin || report.TargetCommit == "" || len(report.ChangedSurfaces) != 0 {
		t.Fatalf("report = %#v", report)
	}
	if strings.Contains(stdout.String(), repo) {
		t.Fatalf("report exposed local checkout path: %s", stdout.String())
	}
}

func TestRunFailsWithChangedReviewedSurfaces(t *testing.T) {
	repo, pin := newUpstreamFixture(t)
	manifest := writeManifest(t, pin)
	commitFile(t, repo, "web.go", "package main\n// changed\n", "auth drift")
	commitFile(t, repo, "usb.go", "package main\n// changed\n", "hid drift")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"--upstream", repo, "--manifest", manifest, "--target", "HEAD"}, &stdout, &stderr); code != 1 {
		t.Fatalf("run exit=%d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var report driftReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Result != "drift" || len(report.ChangedSurfaces) != 3 ||
		report.ChangedSurfaces[0].Name != "auth" || !equalStrings(report.ChangedSurfaces[0].Paths, []string{"web.go"}) ||
		report.ChangedSurfaces[1].Name != "signaling" || !equalStrings(report.ChangedSurfaces[1].Paths, []string{"web.go"}) ||
		report.ChangedSurfaces[2].Name != "hid" || !equalStrings(report.ChangedSurfaces[2].Paths, []string{"usb.go"}) {
		t.Fatalf("report = %#v", report)
	}
}

func TestRunRejectsMissingPinAndMalformedManifestWithoutEchoingLocalPath(t *testing.T) {
	repo, pin := newUpstreamFixture(t)
	manifest := writeManifest(t, strings.Repeat("f", 40))
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--upstream", repo, "--manifest", manifest}, &stdout, &stderr); code != 2 {
		t.Fatalf("missing pin exit=%d stderr=%q", code, stderr.String())
	}
	if strings.Contains(stderr.String(), repo) {
		t.Fatalf("error exposed local checkout path: %q", stderr.String())
	}

	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte(`{"schemaVersion":1,"pinnedCommit":"`+pin+`","repository":"https://github.com/jetkvm/kvm","surfaces":[],"unexpected":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"--upstream", repo, "--manifest", bad}, &stdout, &stderr); code != 2 {
		t.Fatalf("malformed manifest exit=%d stderr=%q", code, stderr.String())
	}
}

func TestValidUpstreamPathRejectsGitPathspecMagic(t *testing.T) {
	for _, name := range []string{"*", ":(glob)**", ":(top)web.go", ":!web.go"} {
		if validUpstreamPath(name) {
			t.Fatalf("accepted Git pathspec magic %q", name)
		}
	}
}

func TestRepositoryManifestUsesExactReviewedSurfaces(t *testing.T) {
	manifest, err := loadManifest(filepath.Join("..", "..", "docs", "compatibility", "jetkvm-upstream-surfaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Surfaces) != len(expectedSurfaces) {
		t.Fatalf("surface count = %d", len(manifest.Surfaces))
	}
}

func newUpstreamFixture(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	git(t, repo, "init", "-q", "-b", "main")
	git(t, repo, "config", "user.name", "Test")
	git(t, repo, "config", "user.email", "test@example.invalid")
	for _, path := range []string{"web.go", "jsonrpc.go", "ota.go", "video.go", "usb.go", "usb_mass_storage.go", "ui/src/routes/devices.$id.tsx", "ui/src/routes/devices.$id.mount.tsx"} {
		full := filepath.Join(repo, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("fixture\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "pin")
	return repo, strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
}

func commitFile(t *testing.T, repo, name, contents, message string) {
	t.Helper()
	full := filepath.Join(repo, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", name)
	git(t, repo, "commit", "-q", "-m", message)
}

func writeManifest(t *testing.T, pin string) string {
	t.Helper()
	manifest := map[string]any{
		"schemaVersion": 1,
		"repository":    "https://github.com/jetkvm/kvm",
		"pinnedCommit":  pin,
		"surfaces": []map[string]any{
			{"name": "auth", "paths": []string{"web.go", "ui/src/routes/devices.$id.tsx"}},
			{"name": "signaling", "paths": []string{"web.go", "ui/src/routes/devices.$id.tsx"}},
			{"name": "rpc", "paths": []string{"jsonrpc.go", "ota.go"}},
			{"name": "video", "paths": []string{"video.go"}},
			{"name": "hid", "paths": []string{"usb.go"}},
			{"name": "virtual_media", "paths": []string{"usb_mass_storage.go", "ui/src/routes/devices.$id.mount.tsx"}},
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func git(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
	return string(output)
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
