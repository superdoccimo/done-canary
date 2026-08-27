package safepath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJoinRejectsTraversalAndAbsolute(t *testing.T) {
	root := t.TempDir()
	if _, err := Join(root, "..", "escape"); err == nil {
		t.Fatal("expected traversal rejection")
	}
	absolute := filepath.Join(filepath.VolumeName(root)+string(filepath.Separator), "absolute")
	if _, err := Join(root, absolute); err == nil {
		t.Fatal("expected absolute component rejection")
	}
}

func TestJoinSupportsSpacesAndUnicode(t *testing.T) {
	root := filepath.Join(t.TempDir(), "space root", "日本語")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Join(root, "runs", "試験 run")
	if err != nil {
		t.Fatal(err)
	}
	if !Within(root, got) {
		t.Fatalf("%q should be within %q", got, root)
	}
}

func TestJoinRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Join(root, "link", "file"); err == nil {
		t.Fatal("expected symlink rejection")
	}
}
