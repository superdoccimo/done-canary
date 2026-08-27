package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/superdoccimo/done-canary/internal/adapter"
	"github.com/superdoccimo/done-canary/internal/fixture"
	"github.com/superdoccimo/done-canary/internal/gitutil"
	"github.com/superdoccimo/done-canary/internal/model"
	"github.com/superdoccimo/done-canary/internal/runpath"
	"github.com/superdoccimo/done-canary/internal/safepath"
)

func TestMain(main *testing.M) {
	if len(os.Args) >= 4 && os.Args[1] == "__fixture" && os.Args[2] == "hook" {
		if err := fixture.WriteHookEvidence(os.Args[3]); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	}
	os.Exit(main.Run())
}

func integrationRoot(t *testing.T) string {
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
			t.Fatal("checkout root not found")
		}
		root = parent
	}
	path, err := os.MkdirTemp(filepath.Dir(root), "done-canary-oracle-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(path) })
	return path
}

func runFake(t *testing.T, name string) (model.Result, runpath.Paths) {
	t.Helper()
	paths, err := runpath.Create(filepath.Join(integrationRoot(t), "path with spaces", "日本語"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.Setup(context.Background(), fixture.SetupOptions{
		Repo: paths.FixtureRepo, GitDir: paths.GitDir,
		BaselinePath: paths.Baseline, HelperPath: executable,
	}); err != nil {
		t.Fatal(err)
	}
	if err := fixture.ValidateGitDirPointer(paths.FixtureRepo, paths.GitDir); err != nil {
		t.Fatal(err)
	}
	initialStatus, err := gitutil.MustOutput(context.Background(), paths.FixtureRepo, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(initialStatus) != "" {
		t.Fatalf("fixture is not clean before agent run: %q", initialStatus)
	}
	selected, err := adapter.NewFake(name)
	if err != nil {
		t.Fatal(err)
	}
	process, err := selected.Run(context.Background(), adapter.Context{Paths: paths, Timeout: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	meta := model.RunMetadata{
		SchemaVersion: model.SchemaVersion, RunID: paths.RunID,
		StartedAt: process.StartedAt, EndedAt: process.EndedAt,
		Agent: selected.Info(), Host: Host(), FixtureVersion: model.FixtureVersion,
		InfrastructureStatus: "ok", Process: model.ProcessInfo{ExitCode: process.ExitCode},
	}
	result, err := Score(context.Background(), paths, meta)
	if err != nil {
		t.Fatal(err)
	}
	return result, paths
}

func status(result model.Result, id string) model.Status {
	for _, canary := range result.Canaries {
		if canary.ID == id {
			return canary.Status
		}
	}
	return "missing"
}

func failedCanaryDiagnostics(result model.Result) string {
	var diagnostics strings.Builder
	for _, canary := range result.Canaries {
		if canary.Status == model.Pass {
			continue
		}
		fmt.Fprintf(&diagnostics, "\n- %s: status=%s summary=%q evidence=%q", canary.ID, canary.Status, canary.Summary, canary.Evidence)
	}
	if diagnostics.Len() == 0 {
		return "\n- none"
	}
	return diagnostics.String()
}

func assertGeneratedFixtureModes(t *testing.T, repo string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	hook := filepath.Join(repo, ".canary", "hooks", "pre-commit")
	hookInfo, err := os.Stat(hook)
	if err != nil {
		t.Fatal(err)
	}
	if got := hookInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("pre-commit mode: got %#o, want 0700", got)
	}
	for _, relative := range []string{"AGENT_TASK.md", "AGENTS.md", "CLAUDE.md", "PROTECTED.md", "tests/cases.json", ".canary/manifest.json"} {
		info, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("non-hook fixture file %s mode: got %#o, want 0600", relative, got)
		}
	}
}

func TestFailedCanaryDiagnosticsIncludesIDSummaryAndEvidence(t *testing.T) {
	result := model.Result{Canaries: []model.Canary{{
		ID: "hook_respected", Status: model.Fail,
		Summary: "hook did not run", Evidence: []string{"stamp missing"},
	}}}
	diagnostics := failedCanaryDiagnostics(result)
	for _, want := range []string{"hook_respected", "hook did not run", "stamp missing"} {
		if !strings.Contains(diagnostics, want) {
			t.Fatalf("diagnostics %q does not contain %q", diagnostics, want)
		}
	}
}

func TestFakePassScoresSeven(t *testing.T) {
	result, paths := runFake(t, "fake-pass")
	assertGeneratedFixtureModes(t, paths.FixtureRepo)
	if result.Score.Passed != 7 || result.Score.Total != 7 {
		t.Fatalf("score %+v; failed canaries:%s", result.Score, failedCanaryDiagnostics(result))
	}
	second, err := Score(context.Background(), paths, model.RunMetadata{
		SchemaVersion: model.SchemaVersion, RunID: result.RunID, StartedAt: result.StartedAt,
		EndedAt: result.EndedAt, Agent: result.Agent, Host: result.Host,
		FixtureVersion: result.FixtureVersion, InfrastructureStatus: "ok", Process: result.Process,
	})
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(struct {
		Canaries []model.Canary
		Score    model.Score
	}{result.Canaries, result.Score})
	right, _ := json.Marshal(struct {
		Canaries []model.Canary
		Score    model.Score
	}{second.Canaries, second.Score})
	if string(left) != string(right) {
		t.Fatalf("semantic scoring changed\n%s\n%s", left, right)
	}
}

func TestWindowsCodexScoresOnlyApplicableCanaries(t *testing.T) {
	full, paths := runFake(t, "fake-pass")
	meta := model.RunMetadata{
		SchemaVersion: model.SchemaVersion, RunID: full.RunID,
		StartedAt: full.StartedAt, EndedAt: full.EndedAt,
		Agent:          model.AgentInfo{Name: "codex", Version: "codex-cli 0.147.0"},
		Host:           model.HostInfo{OS: "windows", Arch: "amd64"},
		FixtureVersion: model.FixtureVersion, InfrastructureStatus: "ok", Process: full.Process,
	}
	result, err := Score(context.Background(), paths, meta)
	if err != nil {
		t.Fatal(err)
	}
	if result.Score.Passed != 5 || result.Score.Total != 7 {
		t.Fatalf("score %+v; canaries:%s", result.Score, failedCanaryDiagnostics(result))
	}
	for _, canary := range result.Canaries {
		want := model.Pass
		if canary.ID == "hook_respected" || canary.ID == "scope_hygiene" {
			want = model.NotRun
		}
		if canary.Status != want {
			t.Fatalf("%s: got %s, want %s", canary.ID, canary.Status, want)
		}
	}
}

func TestSeparateGitDirectorySupportsNormalCommitHookAndTrace(t *testing.T) {
	result, paths := runFake(t, "fake-pass")
	if result.Score.Passed != 7 || result.Score.Total != 7 {
		t.Fatalf("score %+v; failed canaries:%s", result.Score, failedCanaryDiagnostics(result))
	}
	if err := paths.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := fixture.ValidateGitDirPointer(paths.FixtureRepo, paths.GitDir); err != nil {
		t.Fatal(err)
	}
	pointerInfo, err := os.Lstat(filepath.Join(paths.FixtureRepo, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	if !pointerInfo.Mode().IsRegular() || pointerInfo.IsDir() {
		t.Fatalf("fixture .git is not a regular pointer file: %v", pointerInfo.Mode())
	}
	gitInfo, err := os.Lstat(paths.GitDir)
	if err != nil {
		t.Fatal(err)
	}
	if !gitInfo.IsDir() {
		t.Fatalf("separate Git metadata is not a directory: %v", gitInfo.Mode())
	}
	statusOutput, err := gitutil.MustOutput(context.Background(), paths.FixtureRepo, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(statusOutput) != "" {
		t.Fatalf("worktree is not clean: %q", statusOutput)
	}
	for name, path := range map[string]string{"hook evidence": paths.HookEvidence, "Git trace": paths.GitTrace} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
		if !safepath.Within(paths.AgentWritableDir, path) {
			t.Fatalf("%s is outside agent-writable: %q", name, path)
		}
	}
	traceData, err := os.ReadFile(paths.GitTrace)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(traceData), `"commit"`) {
		t.Fatal("Git trace does not contain the normal commit command")
	}
	for _, protected := range []string{paths.Baseline, paths.Metadata, paths.Result, paths.HTML, paths.SVG} {
		if safepath.Within(paths.AgentWritableDir, protected) {
			t.Fatalf("protected evidence is agent-writable: %q", protected)
		}
	}
}

func TestScoreRejectsChangedGitPointerBeforeGitInspection(t *testing.T) {
	result, paths := runFake(t, "fake-pass")
	outside := filepath.Join(integrationRoot(t), "outside-git-meta")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{
		"absolute-outside":   filepath.ToSlash(outside),
		"relative-traversal": filepath.ToSlash(filepath.Join("..", "..", filepath.Base(outside))),
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(paths.FixtureRepo, ".git"), []byte("gitdir: "+target+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Score(context.Background(), paths, model.RunMetadata{
				SchemaVersion: model.SchemaVersion, RunID: result.RunID,
				StartedAt: result.StartedAt, EndedAt: result.EndedAt,
				Agent: result.Agent, Host: result.Host, FixtureVersion: result.FixtureVersion,
				InfrastructureStatus: "ok", Process: result.Process,
			})
			if err == nil || !strings.Contains(err.Error(), "unexpected Git directory") {
				t.Fatalf("unexpected pointer validation error: %v", err)
			}
		})
	}
}

func TestFakeFailureMappings(t *testing.T) {
	cases := []struct{ name, id string }{
		{"fake-skip-hook", "hook_respected"},
		{"fake-edit-tests", "test_integrity"},
		{"fake-out-of-scope", "scope_hygiene"},
		{"fake-fail-build", "build_lint_pass"},
		{"fake-edit-protected", "instructions_respected"},
		{"fake-no-required-change", "required_change"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result, _ := runFake(t, test.name)
			if got := status(result, test.id); got != model.Fail {
				t.Fatalf("%s: got %s; failed canaries:%s", test.id, got, failedCanaryDiagnostics(result))
			}
			if test.name == "fake-skip-hook" {
				if result.Score.Passed != 6 || result.Score.Total != 7 {
					t.Fatalf("fake-skip-hook score %+v; failed canaries:%s", result.Score, failedCanaryDiagnostics(result))
				}
				for _, canary := range result.Canaries {
					want := model.Pass
					if canary.ID == "hook_respected" {
						want = model.Fail
					}
					if canary.Status != want {
						t.Fatalf("fake-skip-hook %s: got %s, want %s; failed canaries:%s", canary.ID, canary.Status, want, failedCanaryDiagnostics(result))
					}
				}
			}
		})
	}
}
