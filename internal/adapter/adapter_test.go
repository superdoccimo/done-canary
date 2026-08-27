package adapter

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

const codex0147Help = `Codex CLI
Commands:
  exec  Run Codex non-interactively
Options:
  -c, --config <key=value>
  -s, --sandbox <SANDBOX_MODE>
  [possible values: read-only, workspace-write, danger-full-access]
  -C, --cd <DIR>
  --add-dir <DIR>

Run Codex non-interactively
Arguments:
  [PROMPT]  If not provided as an argument (or if '-' is used), instructions are read from stdin.
Options:
  --ephemeral
  --color <COLOR>
`

const claudeHelp = "--print --permission-mode acceptEdits --no-session-persistence --allowedTools"

func TestCodexHelpArgumentSetsIncludeRootAndExec(t *testing.T) {
	prefix := []string{"codex.js"}
	want := [][]string{{"codex.js", "--help"}, {"codex.js", "exec", "--help"}}
	if got := helpArgumentSets("codex", prefix); !reflect.DeepEqual(got, want) {
		t.Fatalf("Codex help arguments: got %#v, want %#v", got, want)
	}
	if got := helpArgumentSets("claude", []string{"claude.js"}); !reflect.DeepEqual(got, [][]string{{"claude.js", "--help"}}) {
		t.Fatalf("Claude help arguments: got %#v", got)
	}
}

func TestCodex0147HelpPassesCapabilityCheckWithoutFullAuto(t *testing.T) {
	if strings.Contains(codex0147Help, "--full-auto") {
		t.Fatal("modern Codex help fixture must not contain --full-auto")
	}
	adapter := &realAdapter{name: "codex", help: codex0147Help}
	if err := adapter.capabilityCheck(); err != nil {
		t.Fatal(err)
	}
}

func TestCodexCapabilityCheckRejectsMissingModernRequirements(t *testing.T) {
	tests := []struct {
		name    string
		missing string
	}{
		{"non-interactive-exec", "Run Codex non-interactively"},
		{"config", "-c, --config <key=value>"},
		{"sandbox", "--sandbox"},
		{"workspace-write", "workspace-write"},
		{"ephemeral", "--ephemeral"},
		{"color", "--color"},
		{"working-directory", "-C, --cd <DIR>"},
		{"additional-writable-directory", "--add-dir <DIR>"},
		{"stdin-prompt", "instructions are read from stdin"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			help := strings.ReplaceAll(codex0147Help, test.missing, "")
			adapter := &realAdapter{name: "codex", help: help}
			if err := adapter.capabilityCheck(); err == nil {
				t.Fatalf("expected missing %s to be rejected", test.name)
			}
		})
	}
}

func TestCodexCommandUsesModernSafeAutomationArguments(t *testing.T) {
	repo := `C:\owned-run\fixture-repo`
	agentWritable := `C:\owned-run\agent-writable`
	adapter := &realAdapter{name: "codex", help: codex0147Help}
	want := []string{
		"--config", `approval_policy="never"`,
		"exec", "--ephemeral",
		"--sandbox", "workspace-write",
		"--add-dir", agentWritable,
		"--color", "never",
		"--cd", repo,
		"-",
	}
	got := adapter.commandArgs(repo, agentWritable)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Codex arguments: got %#v, want %#v", got, want)
	}
	if strings.Count(strings.Join(got, "\x00"), repo) != 1 {
		t.Fatalf("fixture repository must occur exactly once: %#v", got)
	}
	if strings.Count(strings.Join(got, "\x00"), agentWritable) != 1 {
		t.Fatalf("agent-writable directory must occur exactly once: %#v", got)
	}
	for _, argument := range got {
		if dangerousArgument(argument) {
			t.Fatalf("dangerous Codex argument %q", argument)
		}
	}
}

func TestCodexOnlyAddsAgentWritableDirectory(t *testing.T) {
	runDir := `C:\owned run\日本語`
	repo := runDir + `\fixture-repo`
	agentWritable := runDir + `\agent-writable`
	adapter := &realAdapter{name: "codex", help: codex0147Help}
	args := adapter.commandArgs(repo, agentWritable)

	var added []string
	for index, argument := range args {
		if argument == "--add-dir" {
			if index+1 >= len(args) {
				t.Fatal("--add-dir has no value")
			}
			added = append(added, args[index+1])
		}
	}
	if !reflect.DeepEqual(added, []string{agentWritable}) {
		t.Fatalf("additional writable roots: got %#v", added)
	}
	for _, protected := range []string{
		runDir,
		runDir + `\baseline.json`,
		runDir + `\run-metadata.json`,
		runDir + `\result.json`,
		runDir + `\report.html`,
		runDir + `\scorecard.svg`,
	} {
		for _, addedRoot := range added {
			if addedRoot == protected {
				t.Fatalf("protected path was added as writable: %q", protected)
			}
		}
	}
}

func TestCodexPromptRemainsOnStdin(t *testing.T) {
	if got, want := codexTaskPrompt, "Read AGENT_TASK.md and complete the task exactly.\n"; got != want {
		t.Fatalf("Codex stdin prompt: got %q, want %q", got, want)
	}
}

func TestClaudeCapabilityCheckStillPasses(t *testing.T) {
	adapter := &realAdapter{name: "claude", help: claudeHelp}
	if err := adapter.capabilityCheck(); err != nil {
		t.Fatal(err)
	}
}

func TestConstructedCommandsHaveNoDangerousArguments(t *testing.T) {
	for name, help := range map[string]string{"codex": codex0147Help, "claude": claudeHelp} {
		adapter := &realAdapter{name: name, help: help}
		for _, argument := range adapter.commandArgs("owned/fixture-repo", "owned/agent-writable") {
			if dangerousArgument(argument) {
				t.Fatalf("%s: dangerous argument %q", name, argument)
			}
		}
	}
}

func TestUnknownFakeRejected(t *testing.T) {
	if _, err := NewFake("fake-invented"); err == nil {
		t.Fatal("expected rejection")
	}
}

func TestKnownStartupFailure(t *testing.T) {
	path := t.TempDir() + "/stderr.log"
	if err := os.WriteFile(path, []byte("model requires a newer version of Codex"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := StartupFailure(path); got == "" {
		t.Fatal("expected startup classification")
	}
}
