package adapter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/superdoccimo/done-canary/internal/fixture"
	"github.com/superdoccimo/done-canary/internal/gitutil"
	"github.com/superdoccimo/done-canary/internal/model"
	"github.com/superdoccimo/done-canary/internal/runner"
)

var fakeNames = map[string]bool{
	"fake-pass": true, "fake-skip-hook": true, "fake-edit-tests": true,
	"fake-out-of-scope": true, "fake-fail-build": true,
	"fake-edit-protected": true, "fake-no-required-change": true,
	"fake-timeout": true,
}

type fakeAdapter struct{ name string }

func NewFake(name string) (Adapter, error) {
	if !fakeNames[name] {
		return nil, fmt.Errorf("%w: %s", ErrUnavailable, name)
	}
	return &fakeAdapter{name: name}, nil
}

func (adapter *fakeAdapter) Info() model.AgentInfo {
	return model.AgentInfo{Name: adapter.name, Version: model.Version, Invocation: []string{"internal", adapter.name}}
}

func (adapter *fakeAdapter) Doctor(context.Context) error { return nil }

func (adapter *fakeAdapter) Run(ctx context.Context, run Context) (runner.Result, error) {
	started := time.Now().UTC()
	if adapter.name == "fake-timeout" {
		helper := filepath.Join(run.Paths.FixtureRepo, filepath.FromSlash(fixture.HelperRelativePath()))
		return runner.Run(ctx, runner.Request{
			Path: helper, Args: []string{"__fixture", "sleep", "30s"}, Dir: run.Paths.FixtureRepo,
			StdoutPath: run.Paths.StdoutLog, StderrPath: run.Paths.StderrLog,
			MaxLogBytes: 64 << 10, Timeout: run.Timeout,
		})
	}
	if err := os.WriteFile(run.Paths.StdoutLog, []byte("deterministic adapter: "+adapter.name+"\n"), 0o600); err != nil {
		return runner.Result{}, err
	}
	if err := os.WriteFile(run.Paths.StderrLog, nil, 0o600); err != nil {
		return runner.Result{}, err
	}
	if adapter.name != "fake-no-required-change" {
		if err := fixture.WriteFixedPolicy(run.Paths.FixtureRepo); err != nil {
			return runner.Result{}, err
		}
	}
	switch adapter.name {
	case "fake-edit-tests":
		path := filepath.Join(run.Paths.FixtureRepo, "tests", "cases.json")
		data, err := os.ReadFile(path)
		if err != nil {
			return runner.Result{}, err
		}
		if err := os.WriteFile(path, append(data, ' ', '\n'), 0o600); err != nil {
			return runner.Result{}, err
		}
	case "fake-out-of-scope":
		if err := os.WriteFile(filepath.Join(run.Paths.FixtureRepo, "UNRELATED.txt"), []byte("out of scope\n"), 0o600); err != nil {
			return runner.Result{}, err
		}
	case "fake-fail-build":
		if err := os.WriteFile(filepath.Join(run.Paths.RunDir, "fake-fail-build"), []byte("1\n"), 0o600); err != nil {
			return runner.Result{}, err
		}
	case "fake-edit-protected":
		if err := os.WriteFile(filepath.Join(run.Paths.FixtureRepo, "PROTECTED.md"), []byte("modified\n"), 0o600); err != nil {
			return runner.Result{}, err
		}
	}
	if adapter.name != "fake-no-required-change" {
		_ = fixture.CheckLint(run.Paths.FixtureRepo)
		_ = fixture.CheckTest(run.Paths.FixtureRepo)
		_ = fixture.CheckBuild(run.Paths.FixtureRepo)
	}
	if _, err := gitutil.Run(ctx, run.Paths.FixtureRepo, nil, "add", "-A"); err != nil {
		return runner.Result{}, err
	}
	args := []string{"commit"}
	if adapter.name == "fake-skip-hook" {
		args = append(args, "--no-verify")
	}
	if adapter.name == "fake-no-required-change" {
		args = append(args, "--allow-empty")
	}
	args = append(args, "-m", fixture.CommitMessage)
	env := map[string]string{
		"GIT_TRACE2_EVENT":         run.Paths.GitTrace,
		"DONECANARY_HOOK_EVIDENCE": run.Paths.HookEvidence,
	}
	if _, err := gitutil.Run(ctx, run.Paths.FixtureRepo, env, args...); err != nil {
		return runner.Result{}, err
	}
	return runner.Result{ExitCode: 0, StartedAt: started, EndedAt: time.Now().UTC()}, nil
}
