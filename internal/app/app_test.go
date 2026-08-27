package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/superdoccimo/done-canary/internal/jsonfile"
	"github.com/superdoccimo/done-canary/internal/model"
	"github.com/superdoccimo/done-canary/internal/report"
)

func TestMain(main *testing.M) {
	if len(os.Args) == 4 && os.Args[1] == "__fixture" && os.Args[2] == "sleep" {
		duration, err := time.ParseDuration(os.Args[3])
		if err != nil {
			os.Exit(2)
		}
		time.Sleep(duration)
		os.Exit(0)
	}
	os.Exit(main.Run())
}

func appIntegrationRoot(t *testing.T) string {
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
	path, err := os.MkdirTemp(filepath.Dir(root), "done-canary-app-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(path) })
	return path
}

func TestRunDoesNotAcceptRepositoryPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Execute(context.Background(), []string{"run", "codex", filepath.Join("some", "repo")}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("got exit %d", exit)
	}
}

func TestOwnedOutputRejectsTraversal(t *testing.T) {
	run := t.TempDir()
	if _, err := ownedOutputPath(run, filepath.Join("..", "outside.html")); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestInterruptedRunLeavesValidRecord(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(2*time.Second, cancel)
	outcome := runOneWithTimeout(ctx, "fake-timeout", appIntegrationRoot(t), &bytes.Buffer{}, &bytes.Buffer{}, false, 20*time.Second)
	if outcome.Exit != 130 || !outcome.Result.Process.Interrupted || outcome.Result.InfrastructureStatus != "interrupted" {
		t.Fatalf("unexpected outcome: exit=%d result=%+v", outcome.Exit, outcome.Result)
	}
	var persisted model.Result
	if err := jsonfile.Read(outcome.Paths.Result, &persisted); err != nil {
		t.Fatal(err)
	}
	if err := report.Validate(persisted); err != nil {
		t.Fatal(err)
	}
}

func TestResultExitUsesApplicableFailures(t *testing.T) {
	partial := model.Result{InfrastructureStatus: "ok", Canaries: []model.Canary{
		{Status: model.Pass}, {Status: model.Pass}, {Status: model.Pass}, {Status: model.Pass},
		{Status: model.NotRun}, {Status: model.Pass}, {Status: model.NotRun},
	}}
	if got := resultExit(partial); got != 0 {
		t.Fatalf("5 PASS / 0 FAIL / 2 NOT RUN exit %d, want 0", got)
	}
	partial.Canaries[0].Status = model.Fail
	if got := resultExit(partial); got != 1 {
		t.Fatalf("applicable failure exit %d, want 1", got)
	}
	partial.InfrastructureStatus = "error"
	if got := resultExit(partial); got != 2 {
		t.Fatalf("infrastructure failure exit %d, want 2", got)
	}
}

func TestDoctorVerifiedLineReportsWindowsCodexCoverage(t *testing.T) {
	info := model.AgentInfo{Name: "codex", Version: "codex-cli 0.147.0"}
	want := "codex: verified (codex-cli 0.147.0; 5/7 canaries applicable on native Windows safe sandbox)"
	if got := doctorVerifiedLine("codex", info, model.HostInfo{OS: "windows", Arch: "amd64"}); got != want {
		t.Fatalf("doctor line %q, want %q", got, want)
	}
	if got := doctorVerifiedLine("codex", info, model.HostInfo{OS: "linux", Arch: "amd64"}); got != "codex: verified (codex-cli 0.147.0)" {
		t.Fatalf("Linux doctor line changed: %q", got)
	}
}
