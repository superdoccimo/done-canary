package runpath

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/superdoccimo/done-canary/internal/safepath"
)

func tempOutsideGit(t *testing.T) string {
	t.Helper()
	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := working
	for {
		if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("test checkout root not found")
		}
		root = parent
	}
	path, err := os.MkdirTemp(filepath.Dir(root), "done-canary-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(path) })
	return path
}

func TestCreateAndOpenOwnedUnicodePath(t *testing.T) {
	root := filepath.Join(tempOutsideGit(t), "path with spaces", "日本語")
	p, err := Create(root, time.Date(2026, 8, 10, 12, 34, 56, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if p.RunID[:16] != "20260810T123456Z" {
		t.Fatalf("unexpected run id %q", p.RunID)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if got, want := p.AgentWritableDir, filepath.Join(p.RunDir, "agent-writable"); got != want {
		t.Fatalf("agent-writable path: got %q, want %q", got, want)
	}
	if got, want := p.GitDir, filepath.Join(p.AgentWritableDir, "git-meta"); got != want {
		t.Fatalf("Git directory: got %q, want %q", got, want)
	}
	if got, want := p.GitTrace, filepath.Join(p.AgentWritableDir, "git-trace.jsonl"); got != want {
		t.Fatalf("Git trace path: got %q, want %q", got, want)
	}
	if got, want := p.HookEvidence, filepath.Join(p.AgentWritableDir, "hook-evidence.json"); got != want {
		t.Fatalf("hook evidence path: got %q, want %q", got, want)
	}
	for _, protected := range []string{p.Baseline, p.Metadata, p.Result, p.HTML, p.SVG} {
		if safepath.Within(p.AgentWritableDir, protected) {
			t.Fatalf("protected path is agent-writable: %q", protected)
		}
	}
	opened, err := OpenOwned(root, p.RunDir)
	if err != nil {
		t.Fatal(err)
	}
	if opened.RunDir != p.RunDir {
		t.Fatalf("got %q want %q", opened.RunDir, p.RunDir)
	}
}

func TestValidateRejectsPathSubstitution(t *testing.T) {
	p, err := Create(filepath.Join(tempOutsideGit(t), "owned"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	p.GitDir = filepath.Join(tempOutsideGit(t), "outside", "git-meta")
	if err := p.Validate(); err == nil {
		t.Fatal("expected substituted Git directory rejection")
	}
}

func TestOpenOwnedRejectsAgentWritableSymlink(t *testing.T) {
	root := filepath.Join(tempOutsideGit(t), "owned")
	p, err := Create(root, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p.AgentWritableDir); err != nil {
		t.Fatal(err)
	}
	outside := tempOutsideGit(t)
	if err := os.Symlink(outside, p.AgentWritableDir); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := OpenOwned(root, p.RunDir); err == nil {
		t.Fatal("expected agent-writable symlink rejection")
	}
}

func TestOpenOwnedRejectsOutside(t *testing.T) {
	base := tempOutsideGit(t)
	root := filepath.Join(base, "root")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenOwned(root, outside); err == nil {
		t.Fatal("expected outside path rejection")
	}
}

func TestCreateRejectsGitTree(t *testing.T) {
	root := tempOutsideGit(t)
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(filepath.Join(root, "data"), time.Now()); err == nil {
		t.Fatal("expected Git tree rejection")
	}
}
