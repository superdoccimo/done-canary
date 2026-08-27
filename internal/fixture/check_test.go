package fixture

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPolicyChecks(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "policy.json"), []byte("{\n  \"version\": 1,\n  \"routes\": {\n    \"fatal\": \"reject\",\n    \"ok\": \"accept\",\n    \"retryable\": \"retry\"\n  }\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "tests", "cases.json"), []byte("[{\"input\":\"retryable\",\"expected\":\"retry\"}]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckTest(repo); err != nil {
		t.Fatal(err)
	}
	if err := CheckBuild(repo); err != nil {
		t.Fatal(err)
	}
}

func TestBuildFakeFailure(t *testing.T) {
	t.Setenv("DONECANARY_FAKE_FAIL_BUILD", "1")
	if err := CheckBuild(t.TempDir()); err == nil || err.Error() != "deterministic fake build failure" {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestPolicyRejectsTrailingJSON(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"version":1,"routes":{"fatal":"reject","ok":"accept","retryable":"retry"}} {}`)
	if err := os.WriteFile(filepath.Join(repo, "src", "policy.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckRequiredChange(repo); err == nil {
		t.Fatal("expected trailing JSON rejection")
	}
}

func TestFixtureFileModesLimitExecutableBitToPreCommit(t *testing.T) {
	tests := []struct {
		name string
		want os.FileMode
	}{
		{filepath.FromSlash(".canary/hooks/pre-commit"), 0o700},
		{"AGENT_TASK.md", 0o600},
		{"AGENTS.md", 0o600},
		{"CLAUDE.md", 0o600},
		{"PROTECTED.md", 0o600},
		{filepath.FromSlash("src/policy.json"), 0o600},
		{filepath.FromSlash("tests/cases.json"), 0o600},
		{filepath.FromSlash(".canary/manifest.json"), 0o600},
	}
	for _, test := range tests {
		t.Run(filepath.ToSlash(test.name), func(t *testing.T) {
			if got := fixtureFileMode(test.name).Perm(); got != test.want {
				t.Fatalf("mode for %s: got %#o, want %#o", test.name, got, test.want)
			}
		})
	}
}

func TestChangedNamesParsesGitPorcelainPaths(t *testing.T) {
	// gitutil.MustOutput trims the combined command output, so an unstaged
	// status at the start of the stream loses its leading XY space.
	rawStatus := strings.Join([]string{
		" M src/policy.json",
		"?? OTHER.txt",
		"M  staged.txt",
		" M unstaged.txt",
		"MM staged-and-unstaged.txt",
		"R  old-name.txt -> new-name.txt",
		` M "quoted path.txt"`,
		" M path with spaces.txt",
		" M 素朴.txt",
		` M "\346\227\245\346\234\254\350\252\236.txt"`,
		"",
	}, "\n")

	got := changedNames("", strings.TrimSpace(rawStatus))
	want := []string{
		"OTHER.txt",
		"new-name.txt",
		"path with spaces.txt",
		"quoted path.txt",
		"src/policy.json",
		"staged-and-unstaged.txt",
		"staged.txt",
		"unstaged.txt",
		"日本語.txt",
		"素朴.txt",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("changedNames() = %#v, want %#v", got, want)
	}
}
