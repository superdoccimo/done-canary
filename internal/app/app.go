package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/superdoccimo/done-canary/internal/adapter"
	"github.com/superdoccimo/done-canary/internal/capability"
	"github.com/superdoccimo/done-canary/internal/fixture"
	"github.com/superdoccimo/done-canary/internal/gitutil"
	"github.com/superdoccimo/done-canary/internal/jsonfile"
	"github.com/superdoccimo/done-canary/internal/model"
	"github.com/superdoccimo/done-canary/internal/oracle"
	"github.com/superdoccimo/done-canary/internal/report"
	"github.com/superdoccimo/done-canary/internal/runpath"
	"github.com/superdoccimo/done-canary/internal/safepath"
)

const defaultTimeout = 20 * time.Minute

type runOutcome struct {
	Result model.Result
	Paths  runpath.Paths
	Exit   int
}

func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "version":
		fmt.Fprintf(stdout, "done-canary %s\n", model.Version)
		return 0
	case "doctor":
		if len(args) != 1 {
			return argumentError(stderr, "doctor accepts no arguments")
		}
		return doctor(ctx, stdout)
	case "run":
		if len(args) != 2 {
			return argumentError(stderr, "usage: done-canary run <codex|claude>")
		}
		outcome := runOne(ctx, args[1], "", stdout, stderr, true)
		return outcome.Exit
	case "score":
		if len(args) != 2 {
			return argumentError(stderr, "usage: done-canary score <run-directory>")
		}
		return scoreExisting(ctx, args[1], stdout, stderr)
	case "report":
		return reportExisting(args[1:], stdout, stderr)
	case "selftest":
		if len(args) != 1 {
			return argumentError(stderr, "selftest accepts no arguments")
		}
		return selftest(ctx, stdout, stderr)
	case "__fixture":
		return fixtureCommand(args[1:], stderr)
	case "help", "--help", "-h":
		usage(stdout)
		return 0
	default:
		return argumentError(stderr, fmt.Sprintf("unknown command %q", args[0]))
	}
}

func runOne(ctx context.Context, name, dataRoot string, stdout, stderr io.Writer, show bool) runOutcome {
	return runOneWithTimeout(ctx, name, dataRoot, stdout, stderr, show, defaultTimeout)
}

func runOneWithTimeout(ctx context.Context, name, dataRoot string, stdout, stderr io.Writer, show bool, timeout time.Duration) runOutcome {
	selected, err := adapter.Resolve(ctx, name)
	if err != nil {
		fmt.Fprintln(stderr, err)
		if errors.Is(err, adapter.ErrUnavailable) || errors.Is(err, adapter.ErrUnsupported) {
			return runOutcome{Exit: 3}
		}
		return runOutcome{Exit: 2}
	}
	if err := selected.Doctor(ctx); err != nil {
		fmt.Fprintln(stderr, err)
		return runOutcome{Exit: 3}
	}
	paths, err := runpath.Create(dataRoot, time.Now())
	if err != nil {
		fmt.Fprintln(stderr, "create run:", err)
		return runOutcome{Exit: 2}
	}
	started := time.Now().UTC()
	runContext := adapter.Context{Paths: paths, Timeout: timeout}
	meta := model.RunMetadata{
		SchemaVersion: model.SchemaVersion, RunID: paths.RunID, StartedAt: started,
		Agent: adapter.Describe(selected, runContext), Host: oracle.Host(),
		FixtureVersion: model.FixtureVersion, InfrastructureStatus: "preparing",
		Process: model.ProcessInfo{ExitCode: -1},
	}
	_ = jsonfile.Write(paths.Metadata, meta)
	executable, err := os.Executable()
	if err != nil {
		return infrastructureFailure(paths, meta, "resolve DoneCanary executable: "+err.Error(), stdout, stderr, show, 2)
	}
	if _, err := fixture.Setup(ctx, fixture.SetupOptions{
		Repo: paths.FixtureRepo, GitDir: paths.GitDir,
		BaselinePath: paths.Baseline, HelperPath: executable,
	}); err != nil {
		return infrastructureFailure(paths, meta, "prepare disposable fixture: "+err.Error(), stdout, stderr, show, 2)
	}
	process, runErr := selected.Run(ctx, runContext)
	meta.EndedAt = process.EndedAt
	if meta.EndedAt.IsZero() {
		meta.EndedAt = time.Now().UTC()
	}
	meta.Process = model.ProcessInfo{
		ExitCode: process.ExitCode, TimedOut: process.TimedOut,
		Interrupted: process.Interrupted, LogTruncated: process.Truncated,
	}
	if runErr != nil {
		exit := 2
		status := "error"
		if process.Interrupted || errors.Is(runErr, context.Canceled) {
			exit, status = 130, "interrupted"
		}
		meta.InfrastructureStatus = status
		meta.InfrastructureError = runErr.Error()
		return infrastructureFailure(paths, meta, runErr.Error(), stdout, stderr, show, exit)
	}
	meta.InfrastructureStatus = "ok"
	if err := jsonfile.Write(paths.Metadata, meta); err != nil {
		return infrastructureFailure(paths, meta, err.Error(), stdout, stderr, show, 2)
	}
	result, err := oracle.Score(ctx, paths, meta)
	if err != nil {
		return infrastructureFailure(paths, meta, "score run: "+err.Error(), stdout, stderr, show, 2)
	}
	if err := report.WriteAll(paths, result); err != nil {
		fmt.Fprintln(stderr, "write reports:", err)
		return runOutcome{Result: result, Paths: paths, Exit: 2}
	}
	if show {
		report.Terminal(stdout, result)
		printPaths(stdout, paths)
	}
	return runOutcome{Result: result, Paths: paths, Exit: resultExit(result)}
}

func infrastructureFailure(paths runpath.Paths, meta model.RunMetadata, message string, stdout, stderr io.Writer, show bool, exit int) runOutcome {
	meta.EndedAt = time.Now().UTC()
	meta.InfrastructureError = message
	if meta.InfrastructureStatus != "interrupted" {
		meta.InfrastructureStatus = "error"
	}
	_ = jsonfile.Write(paths.Metadata, meta)
	result := model.InfrastructureFailure(meta, message)
	if meta.InfrastructureStatus == "interrupted" {
		result.InfrastructureStatus = "interrupted"
	}
	if err := report.WriteAll(paths, result); err != nil {
		fmt.Fprintln(stderr, "write infrastructure report:", err)
	}
	if show {
		fmt.Fprintln(stderr, "DoneCanary infrastructure error:", message)
		report.Terminal(stdout, result)
		printPaths(stdout, paths)
	}
	return runOutcome{Result: result, Paths: paths, Exit: exit}
}

func selftest(ctx context.Context, stdout, stderr io.Writer) int {
	root, err := runpath.DefaultDataRoot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	testRoot := filepath.Join(root, "selftest")
	expectations := []struct {
		name    string
		fails   []string
		timeout bool
	}{
		{"fake-pass", nil, false},
		{"fake-skip-hook", []string{"hook_respected"}, false},
		{"fake-edit-tests", []string{"test_integrity", "scope_hygiene"}, false},
		{"fake-out-of-scope", []string{"scope_hygiene"}, false},
		{"fake-fail-build", []string{"build_lint_pass"}, false},
		{"fake-edit-protected", []string{"instructions_respected"}, false},
		{"fake-no-required-change", []string{"required_change"}, false},
		{"fake-timeout", nil, true},
	}
	passed := 0
	for _, expected := range expectations {
		timeout := defaultTimeout
		if expected.timeout {
			timeout = 250 * time.Millisecond
		}
		outcome := runOneWithTimeout(ctx, expected.name, testRoot, io.Discard, stderr, false, timeout)
		ok := true
		if expected.timeout {
			ok = outcome.Exit == 2 && outcome.Result.Process.TimedOut && outcome.Result.InfrastructureStatus == "error"
		} else {
			if outcome.Exit == 2 || outcome.Exit == 3 || outcome.Result.InfrastructureStatus != "ok" {
				ok = false
			}
			if expected.name == "fake-pass" && outcome.Result.Score.Passed != 7 {
				ok = false
			}
			for _, id := range expected.fails {
				if canaryStatus(outcome.Result, id) != model.Fail {
					ok = false
				}
			}
		}
		status := "PASS"
		if !ok {
			status = "FAIL"
		}
		fmt.Fprintf(stdout, "%-24s %s\n", expected.name, status)
		if ok {
			passed++
		}
	}
	fmt.Fprintf(stdout, "\nSELFTEST %d / %d\n", passed, len(expectations))
	if passed != len(expectations) {
		return 1
	}
	return 0
}

func scoreExisting(ctx context.Context, runDir string, stdout, stderr io.Writer) int {
	paths, err := openOwnedRun(runDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	var meta model.RunMetadata
	if err := jsonfile.Read(paths.Metadata, &meta); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if meta.Process.ExitCode != 0 {
		if message := adapter.StartupFailure(paths.StdoutLog, paths.StderrLog); message != "" {
			meta.InfrastructureStatus = "error"
			meta.InfrastructureError = message
			outcome := infrastructureFailure(paths, meta, message, stdout, stderr, true, 2)
			return outcome.Exit
		}
	}
	if meta.InfrastructureStatus != "ok" {
		fmt.Fprintln(stderr, "run infrastructure status is", meta.InfrastructureStatus)
		return 2
	}
	result, err := oracle.Score(ctx, paths, meta)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := report.WriteAll(paths, result); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	report.Terminal(stdout, result)
	printPaths(stdout, paths)
	return resultExit(result)
}

func reportExisting(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("report", flag.ContinueOnError)
	flags.SetOutput(stderr)
	htmlPath := flags.String("html", "", "HTML output path inside the run directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 || *htmlPath == "" {
		return argumentError(stderr, "usage: done-canary report <run-directory> --html <path>")
	}
	paths, err := openOwnedRun(flags.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	var result model.Result
	if err := jsonfile.Read(paths.Result, &result); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := report.Validate(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	target, err := ownedOutputPath(paths.RunDir, *htmlPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := report.WriteHTML(target, paths, result); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprintln(stdout, target)
	return 0
}

func doctor(ctx context.Context, stdout io.Writer) int {
	exit := 0
	host := oracle.Host()
	if path, err := gitutil.Available(); err != nil {
		fmt.Fprintln(stdout, "Git: unavailable")
		exit = 2
	} else {
		fmt.Fprintln(stdout, "Git:", path)
	}
	if root, err := runpath.DefaultDataRoot(); err != nil {
		fmt.Fprintln(stdout, "Data root: error:", err)
		exit = 2
	} else if err := safepath.RejectGitTree(root); err != nil {
		fmt.Fprintln(stdout, "Data root: blocked:", err)
		exit = 2
	} else {
		fmt.Fprintln(stdout, "Data root:", root)
	}
	for _, name := range []string{"codex", "claude"} {
		selected, err := adapter.Resolve(ctx, name)
		if err != nil {
			state := "unsupported"
			if errors.Is(err, adapter.ErrUnavailable) {
				state = "absent"
			}
			fmt.Fprintf(stdout, "%s: %s (%v)\n", name, state, err)
			continue
		}
		info := selected.Info()
		fmt.Fprintln(stdout, doctorVerifiedLine(name, info, host))
	}
	return exit
}

func doctorVerifiedLine(name string, info model.AgentInfo, host model.HostInfo) string {
	profile := capability.For(info.Name, host.OS)
	if profile.ApplicableCount() != len(model.CanaryOrder) {
		return fmt.Sprintf("%s: verified (%s; %d/%d canaries applicable on native Windows safe sandbox)",
			name, info.Version, profile.ApplicableCount(), len(model.CanaryOrder))
	}
	return fmt.Sprintf("%s: verified (%s)", name, info.Version)
}

func resultExit(result model.Result) int {
	if result.InfrastructureStatus != "ok" {
		return 2
	}
	for _, canary := range result.Canaries {
		if canary.Status == model.Fail {
			return 1
		}
	}
	return 0
}

func fixtureCommand(args []string, stderr io.Writer) int {
	if len(args) == 0 {
		return argumentError(stderr, "fixture subcommand required")
	}
	switch args[0] {
	case "lint", "test", "build":
		if len(args) != 1 {
			return argumentError(stderr, "fixture check accepts no arguments")
		}
		working, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		if err := fixture.Check(working, args[0]); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case "hook":
		if len(args) != 2 {
			return argumentError(stderr, "fixture hook requires evidence path")
		}
		if err := fixture.WriteHookEvidence(args[1]); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		return 0
	case "sleep":
		if len(args) != 2 {
			return argumentError(stderr, "fixture sleep requires duration")
		}
		duration, err := time.ParseDuration(args[1])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		time.Sleep(duration)
		return 0
	default:
		return argumentError(stderr, "unknown fixture subcommand")
	}
}

func openOwnedRun(runDir string) (runpath.Paths, error) {
	root, err := runpath.DefaultDataRoot()
	if err != nil {
		return runpath.Paths{}, err
	}
	return runpath.OpenOwned(root, runDir)
}

func ownedOutputPath(runDir, value string) (string, error) {
	target := value
	if !filepath.IsAbs(target) {
		target = filepath.Join(runDir, target)
	}
	if !safepath.Within(runDir, target) {
		return "", errors.New("report output must stay inside the owned run directory")
	}
	relative, err := filepath.Rel(runDir, target)
	if err != nil {
		return "", err
	}
	return safepath.Join(runDir, relative)
}

func canaryStatus(result model.Result, id string) model.Status {
	for _, canary := range result.Canaries {
		if canary.ID == id {
			return canary.Status
		}
	}
	return "missing"
}

func printPaths(writer io.Writer, paths runpath.Paths) {
	fmt.Fprintf(writer, "\nJSON:      %s\nHTML:      %s\nScorecard: %s\n", paths.Result, paths.HTML, paths.SVG)
}

func argumentError(writer io.Writer, message string) int { fmt.Fprintln(writer, message); return 2 }

func usage(writer io.Writer) {
	fmt.Fprintln(writer, strings.TrimSpace(`DoneCanary — Does your coding agent actually finish the job?

Usage:
  done-canary doctor
  done-canary run <codex|claude>
  done-canary score <run-directory>
  done-canary report <run-directory> --html <path>
  done-canary selftest
  done-canary version`))
}
