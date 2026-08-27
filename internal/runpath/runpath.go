package runpath

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/superdoccimo/done-canary/internal/safepath"
)

type Paths struct {
	DataRoot         string
	RunsRoot         string
	RunDir           string
	RunID            string
	FixtureRepo      string
	AgentWritableDir string
	GitDir           string
	StdoutLog        string
	StderrLog        string
	GitTrace         string
	HookEvidence     string
	Baseline         string
	Result           string
	HTML             string
	SVG              string
	Metadata         string
}

func DefaultDataRoot() (string, error) {
	if configured := os.Getenv("DONECANARY_DATA_DIR"); configured != "" {
		return safepath.CleanAbs(configured)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache directory: %w", err)
	}
	return safepath.CleanAbs(filepath.Join(cache, "done-canary"))
}

func Create(dataRoot string, now time.Time) (Paths, error) {
	if dataRoot == "" {
		var err error
		dataRoot, err = DefaultDataRoot()
		if err != nil {
			return Paths{}, err
		}
	}
	root, err := safepath.CleanAbs(dataRoot)
	if err != nil {
		return Paths{}, err
	}
	if err := safepath.RejectGitTree(root); err != nil {
		return Paths{}, err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return Paths{}, fmt.Errorf("create data root: %w", err)
	}
	runs, err := safepath.Join(root, "runs")
	if err != nil {
		return Paths{}, err
	}
	if err := os.MkdirAll(runs, 0o700); err != nil {
		return Paths{}, fmt.Errorf("create runs root: %w", err)
	}
	id, err := newID(now)
	if err != nil {
		return Paths{}, err
	}
	runDir, err := safepath.Join(runs, id)
	if err != nil {
		return Paths{}, err
	}
	if err := os.Mkdir(runDir, 0o700); err != nil {
		return Paths{}, fmt.Errorf("create run directory: %w", err)
	}
	paths, err := build(root, runs, runDir, id)
	if err != nil {
		return Paths{}, err
	}
	if err := os.Mkdir(paths.AgentWritableDir, 0o700); err != nil {
		return Paths{}, fmt.Errorf("create agent-writable directory: %w", err)
	}
	if err := paths.Validate(); err != nil {
		return Paths{}, err
	}
	return paths, nil
}

func OpenOwned(dataRoot, runDir string) (Paths, error) {
	root, err := safepath.CleanAbs(dataRoot)
	if err != nil {
		return Paths{}, err
	}
	runs, err := safepath.Join(root, "runs")
	if err != nil {
		return Paths{}, err
	}
	dir, err := safepath.CleanAbs(runDir)
	if err != nil {
		return Paths{}, err
	}
	if !safepath.Within(runs, dir) || dir == runs || filepath.Dir(dir) != runs {
		return Paths{}, fmt.Errorf("run directory is outside the owned runs root: %q", dir)
	}
	if _, err := safepath.Join(runs, filepath.Base(dir)); err != nil {
		return Paths{}, err
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return Paths{}, fmt.Errorf("invalid run directory: %q", dir)
	}
	paths, err := build(root, runs, dir, filepath.Base(dir))
	if err != nil {
		return Paths{}, err
	}
	if err := paths.Validate(); err != nil {
		return Paths{}, err
	}
	return paths, nil
}

func build(root, runs, dir, id string) (Paths, error) {
	join := func(parts ...string) (string, error) { return safepath.Join(dir, parts...) }
	fixture, err := join("fixture-repo")
	if err != nil {
		return Paths{}, err
	}
	agentWritable, err := join("agent-writable")
	if err != nil {
		return Paths{}, err
	}
	gitDir, err := safepath.Join(agentWritable, "git-meta")
	if err != nil {
		return Paths{}, err
	}
	stdout, err := join("agent.stdout.log")
	if err != nil {
		return Paths{}, err
	}
	stderr, err := join("agent.stderr.log")
	if err != nil {
		return Paths{}, err
	}
	trace, err := safepath.Join(agentWritable, "git-trace.jsonl")
	if err != nil {
		return Paths{}, err
	}
	hook, err := safepath.Join(agentWritable, "hook-evidence.json")
	if err != nil {
		return Paths{}, err
	}
	baseline, err := join("baseline.json")
	if err != nil {
		return Paths{}, err
	}
	result, err := join("result.json")
	if err != nil {
		return Paths{}, err
	}
	html, err := join("report.html")
	if err != nil {
		return Paths{}, err
	}
	svg, err := join("scorecard.svg")
	if err != nil {
		return Paths{}, err
	}
	metadata, err := join("run-metadata.json")
	if err != nil {
		return Paths{}, err
	}
	return Paths{
		DataRoot: root, RunsRoot: runs, RunDir: dir, RunID: id,
		FixtureRepo: fixture, AgentWritableDir: agentWritable, GitDir: gitDir,
		StdoutLog: stdout, StderrLog: stderr,
		GitTrace: trace, HookEvidence: hook, Baseline: baseline,
		Result: result, HTML: html, SVG: svg, Metadata: metadata,
	}, nil
}

func (paths Paths) Validate() error {
	root, err := safepath.CleanAbs(paths.DataRoot)
	if err != nil {
		return err
	}
	runs, err := safepath.Join(root, "runs")
	if err != nil {
		return err
	}
	if !samePath(paths.RunsRoot, runs) {
		return errors.New("runs root does not match the current data root")
	}
	if filepath.Base(paths.RunDir) != paths.RunID || filepath.Dir(paths.RunDir) != paths.RunsRoot {
		return errors.New("run directory does not match the current owned run")
	}
	expected, err := build(root, runs, paths.RunDir, paths.RunID)
	if err != nil {
		return err
	}
	actualPaths := []string{
		paths.FixtureRepo, paths.AgentWritableDir, paths.GitDir,
		paths.StdoutLog, paths.StderrLog, paths.GitTrace, paths.HookEvidence,
		paths.Baseline, paths.Result, paths.HTML, paths.SVG, paths.Metadata,
	}
	expectedPaths := []string{
		expected.FixtureRepo, expected.AgentWritableDir, expected.GitDir,
		expected.StdoutLog, expected.StderrLog, expected.GitTrace, expected.HookEvidence,
		expected.Baseline, expected.Result, expected.HTML, expected.SVG, expected.Metadata,
	}
	for index := range actualPaths {
		if !samePath(actualPaths[index], expectedPaths[index]) {
			return fmt.Errorf("run path does not match owned layout: %q", actualPaths[index])
		}
	}
	info, err := os.Lstat(paths.AgentWritableDir)
	if err != nil {
		return fmt.Errorf("inspect agent-writable directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("agent-writable path is not a regular directory")
	}
	return nil
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func newID(now time.Time) (string, error) {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate run id: %w", err)
	}
	return now.UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(raw[:]), nil
}
