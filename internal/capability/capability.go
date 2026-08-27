package capability

import "github.com/superdoccimo/done-canary/internal/model"

const (
	hookNotRunSummary  = "Not run because the native Windows Codex safe sandbox cannot reliably complete the required hook-respecting Git commit."
	scopeNotRunSummary = "Not run because this canary requires the same Git commit path that is unavailable in the native Windows Codex safe sandbox."
)

// Profile describes which canaries can be evaluated for an agent and host.
// It does not alter the oracle checks for canaries that remain applicable.
type Profile struct {
	windowsCodexSafeSandbox bool
}

func For(agentName, hostOS string) Profile {
	return Profile{windowsCodexSafeSandbox: agentName == "codex" && hostOS == "windows"}
}

func (profile Profile) Applicable(canaryID string) bool {
	if !profile.windowsCodexSafeSandbox {
		return true
	}
	return canaryID != "hook_respected" && canaryID != "scope_hygiene"
}

func (profile Profile) ApplicableCount() int {
	count := 0
	for _, canary := range model.CanaryOrder {
		if profile.Applicable(canary.ID) {
			count++
		}
	}
	return count
}

func (profile Profile) NotRun(canaryID string) (model.Canary, bool) {
	if profile.Applicable(canaryID) {
		return model.Canary{}, false
	}
	summary := ""
	switch canaryID {
	case "hook_respected":
		summary = hookNotRunSummary
	case "scope_hygiene":
		summary = scopeNotRunSummary
	default:
		return model.Canary{}, false
	}
	return model.Canary{
		ID:      canaryID,
		Status:  model.NotRun,
		Summary: summary,
		Evidence: []string{
			"agent: codex",
			"host OS: windows",
			"safe sandbox limitation: the normal hook-respecting Git commit path is not reliably available",
			"dangerous permission bypass was not enabled",
		},
	}, true
}
