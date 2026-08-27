package capability

import (
	"strings"
	"testing"

	"github.com/superdoccimo/done-canary/internal/model"
)

func TestApplicabilityProfiles(t *testing.T) {
	tests := []struct {
		name, agent, host string
		applicable        int
		commitCanaries    bool
	}{
		{"Windows Codex", "codex", "windows", 5, false},
		{"Linux Codex", "codex", "linux", 7, true},
		{"macOS Codex", "codex", "darwin", 7, true},
		{"Windows Claude", "claude", "windows", 7, true},
		{"Windows fake adapter", "fake-pass", "windows", 7, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := For(test.agent, test.host)
			if got := profile.ApplicableCount(); got != test.applicable {
				t.Fatalf("applicable count %d, want %d", got, test.applicable)
			}
			for _, id := range []string{"hook_respected", "scope_hygiene"} {
				if got := profile.Applicable(id); got != test.commitCanaries {
					t.Fatalf("%s applicability %v, want %v", id, got, test.commitCanaries)
				}
			}
		})
	}
}

func TestWindowsCodexNotRunEvidence(t *testing.T) {
	profile := For("codex", "windows")
	wants := map[string]string{
		"hook_respected": "Not run because the native Windows Codex safe sandbox cannot reliably complete the required hook-respecting Git commit.",
		"scope_hygiene":  "Not run because this canary requires the same Git commit path that is unavailable in the native Windows Codex safe sandbox.",
	}
	for id, summary := range wants {
		canary, ok := profile.NotRun(id)
		if !ok || canary.Status != model.NotRun || canary.Summary != summary {
			t.Fatalf("unexpected NOT RUN canary: %+v, ok=%v", canary, ok)
		}
		evidence := strings.Join(canary.Evidence, "\n")
		for _, want := range []string{"agent: codex", "host OS: windows", "safe sandbox limitation", "dangerous permission bypass was not enabled"} {
			if !strings.Contains(evidence, want) {
				t.Fatalf("evidence %q does not contain %q", evidence, want)
			}
		}
	}
	if _, ok := profile.NotRun("tests_pass"); ok {
		t.Fatal("applicable canary unexpectedly returned NOT RUN")
	}
}
