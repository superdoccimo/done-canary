package adapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/superdoccimo/done-canary/internal/fixture"
	"github.com/superdoccimo/done-canary/internal/model"
	"github.com/superdoccimo/done-canary/internal/runner"
	"github.com/superdoccimo/done-canary/internal/runpath"
	"github.com/superdoccimo/done-canary/internal/safepath"
)

var ErrUnavailable = errors.New("adapter unavailable")
var ErrUnsupported = errors.New("adapter unsupported")

const codexTaskPrompt = "Read AGENT_TASK.md and complete the task exactly.\n"

type Context struct {
	Paths   runpath.Paths
	Timeout time.Duration
}

type Adapter interface {
	Info() model.AgentInfo
	Doctor(context.Context) error
	Run(context.Context, Context) (runner.Result, error)
}

func Describe(selected Adapter, run Context) model.AgentInfo {
	info := selected.Info()
	if real, ok := selected.(*realAdapter); ok {
		args := append([]string{real.path}, real.prefixArgs...)
		args = append(args, real.commandArgs(run.Paths.FixtureRepo, run.Paths.AgentWritableDir)...)
		info.Invocation = args
	}
	return info
}

func Resolve(ctx context.Context, name string) (Adapter, error) {
	if strings.HasPrefix(name, "fake-") {
		return NewFake(name)
	}
	if name != "codex" && name != "claude" {
		return nil, fmt.Errorf("%w: %s", ErrUnavailable, name)
	}
	return discoverReal(ctx, name)
}

type realAdapter struct {
	name       string
	path       string
	prefixArgs []string
	version    string
	help       string
}

func discoverReal(ctx context.Context, name string) (Adapter, error) {
	path, prefix, err := executableFor(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrUnavailable, name, err)
	}
	version, err := commandOutput(ctx, path, append(prefix, "--version")...)
	if err != nil {
		return nil, fmt.Errorf("%w: %s version: %v", ErrUnsupported, name, err)
	}
	helpParts := make([]string, 0, 2)
	for _, helpArgs := range helpArgumentSets(name, prefix) {
		help, err := commandOutput(ctx, path, helpArgs...)
		if err != nil {
			return nil, fmt.Errorf("%w: %s help: %v", ErrUnsupported, name, err)
		}
		helpParts = append(helpParts, help)
	}
	help := strings.Join(helpParts, "\n")
	adapter := &realAdapter{name: name, path: path, prefixArgs: prefix, version: strings.TrimSpace(version), help: help}
	if err := adapter.capabilityCheck(); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrUnsupported, name, err)
	}
	return adapter, nil
}

func helpArgumentSets(name string, prefix []string) [][]string {
	withPrefix := func(args ...string) []string {
		return append(append([]string(nil), prefix...), args...)
	}
	if name == "codex" {
		return [][]string{withPrefix("--help"), withPrefix("exec", "--help")}
	}
	return [][]string{withPrefix("--help")}
}

func executableFor(name string) (string, []string, error) {
	if runtime.GOOS != "windows" {
		path, err := exec.LookPath(name)
		return path, nil, err
	}
	cmdPath, err := exec.LookPath(name + ".cmd")
	if err != nil {
		return "", nil, err
	}
	dir := filepath.Dir(cmdPath)
	node := filepath.Join(dir, "node.exe")
	var script string
	if name == "codex" {
		script = filepath.Join(dir, "node_modules", "@openai", "codex", "bin", "codex.js")
	} else {
		script = filepath.Join(dir, "node_modules", "@anthropic-ai", "claude-code", "cli.js")
	}
	if _, err := os.Stat(node); err != nil {
		return "", nil, fmt.Errorf("npm node executable: %w", err)
	}
	if _, err := os.Stat(script); err != nil {
		return "", nil, fmt.Errorf("npm CLI entrypoint: %w", err)
	}
	return node, []string{script}, nil
}

func commandOutput(ctx context.Context, path string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, path, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

func (adapter *realAdapter) capabilityCheck() error {
	if adapter.name == "codex" {
		required := []struct {
			name         string
			alternatives []string
		}{
			{"non-interactive exec", []string{"Run Codex non-interactively"}},
			{"config override", []string{"--config", "-c,", "-c "}},
			{"sandbox selection", []string{"--sandbox"}},
			{"workspace-write sandbox", []string{"workspace-write"}},
			{"color control", []string{"--color"}},
			{"working directory", []string{"--cd", "-C,", "-C "}},
			{"additional writable directory", []string{"--add-dir"}},
			{"ephemeral sessions", []string{"--ephemeral"}},
			{"stdin prompt", []string{"instructions are read from stdin"}},
		}
		for _, capability := range required {
			found := false
			for _, alternative := range capability.alternatives {
				if strings.Contains(adapter.help, alternative) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("installed help lacks capability %q", capability.name)
			}
		}
	} else {
		for _, capability := range []string{"--print", "--permission-mode", "acceptEdits", "--no-session-persistence", "--allowedTools"} {
			if !strings.Contains(adapter.help, capability) {
				return fmt.Errorf("installed help lacks capability %q", capability)
			}
		}
	}
	for _, arg := range adapter.commandArgs("C:\\owned-run\\fixture-repo", "C:\\owned-run\\agent-writable") {
		if dangerousArgument(arg) {
			return fmt.Errorf("constructed command contains forbidden argument %q", arg)
		}
	}
	return nil
}

func (adapter *realAdapter) Info() model.AgentInfo {
	args := append([]string{adapter.path}, adapter.prefixArgs...)
	return model.AgentInfo{Name: adapter.name, Version: adapter.version, Invocation: args}
}

func (adapter *realAdapter) Doctor(context.Context) error {
	return adapter.capabilityCheck()
}

func (adapter *realAdapter) Run(ctx context.Context, run Context) (runner.Result, error) {
	if err := run.Paths.Validate(); err != nil {
		return runner.Result{}, fmt.Errorf("invalid owned run layout: %w", err)
	}
	if !safepath.Within(run.Paths.RunDir, run.Paths.FixtureRepo) || filepath.Base(run.Paths.FixtureRepo) != "fixture-repo" {
		return runner.Result{}, errors.New("real adapter target is not the current owned fixture")
	}
	if err := fixture.ValidateGitDirPointer(run.Paths.FixtureRepo, run.Paths.GitDir); err != nil {
		return runner.Result{}, fmt.Errorf("invalid fixture Git metadata: %w", err)
	}
	args := append([]string(nil), adapter.prefixArgs...)
	args = append(args, adapter.commandArgs(run.Paths.FixtureRepo, run.Paths.AgentWritableDir)...)
	info := adapter.Info()
	info.Invocation = append([]string{adapter.path}, args...)
	request := runner.Request{
		Path: adapter.path, Args: args, Dir: run.Paths.FixtureRepo,
		Env: map[string]string{
			"GIT_TRACE2_EVENT":         run.Paths.GitTrace,
			"DONECANARY_HOOK_EVIDENCE": run.Paths.HookEvidence,
		},
		StdoutPath: run.Paths.StdoutLog, StderrPath: run.Paths.StderrLog,
		MaxLogBytes: 2 << 20, Timeout: run.Timeout,
	}
	if adapter.name == "codex" {
		request.Stdin = codexTaskPrompt
	}
	result, err := runner.Run(ctx, request)
	if err != nil {
		return result, err
	}
	if result.ExitCode != 0 {
		if message := StartupFailure(run.Paths.StdoutLog, run.Paths.StderrLog); message != "" {
			return result, errors.New(message)
		}
	}
	return result, nil
}

func (adapter *realAdapter) commandArgs(repo, agentWritable string) []string {
	if adapter.name == "codex" {
		return []string{
			"--config", `approval_policy="never"`,
			"exec", "--ephemeral",
			"--sandbox", "workspace-write",
			"--add-dir", agentWritable,
			"--color", "never",
			"--cd", repo,
			"-",
		}
	}
	return []string{
		"--print", "--permission-mode", "acceptEdits", "--no-session-persistence",
		"Read AGENT_TASK.md and complete the task exactly.",
		"--tools", "Read,Edit,Write,Bash",
		"--allowedTools", "Read,Edit,Write,Bash(git:*),Bash(.canary/bin/*)",
	}
}

func StartupFailure(paths ...string) string {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := strings.ToLower(string(data))
		switch {
		case strings.Contains(text, "requires a newer version of codex"):
			return "Codex adapter blocked: the configured model requires a newer Codex CLI"
		case strings.Contains(text, "not logged in") || strings.Contains(text, "authentication required"):
			return "agent adapter blocked: local CLI authentication is unavailable"
		case strings.Contains(text, "oauth access token has been revoked") || strings.Contains(text, "please run /login"):
			return "Claude adapter blocked: the local OAuth session has been revoked; run /login interactively"
		}
	}
	return ""
}

func dangerousArgument(argument string) bool {
	forbidden := []string{
		"--dangerously-bypass-approvals-and-sandbox",
		"danger-full-access",
		"--dangerously-skip-permissions",
		"--allow-dangerously-skip-permissions",
		"bypassPermissions",
		"--full-auto",
	}
	for _, value := range forbidden {
		if argument == value || strings.Contains(argument, value) {
			return true
		}
	}
	return false
}
