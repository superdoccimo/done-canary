package gitutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Result struct {
	Output   string
	ExitCode int
}

func Available() (string, error) {
	path, err := exec.LookPath("git")
	if err != nil {
		return "", errors.New("Git executable was not found")
	}
	return path, nil
}

func Run(ctx context.Context, dir string, env map[string]string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = mergeEnv(env)
	out, err := cmd.CombinedOutput()
	result := Result{Output: strings.TrimSpace(string(out)), ExitCode: 0}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, fmt.Errorf("git %s exited %d: %s", strings.Join(args, " "), result.ExitCode, result.Output)
	}
	return result, fmt.Errorf("run git %s: %w", strings.Join(args, " "), err)
}

func MustOutput(ctx context.Context, dir string, args ...string) (string, error) {
	result, err := Run(ctx, dir, nil, args...)
	return result.Output, err
}

func CommitCount(ctx context.Context, dir, revisionRange string) (int, error) {
	out, err := MustOutput(ctx, dir, "rev-list", "--count", revisionRange)
	if err != nil {
		return 0, err
	}
	count, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, fmt.Errorf("parse commit count %q: %w", out, err)
	}
	return count, nil
}

func mergeEnv(extra map[string]string) []string {
	env := os.Environ()
	for key, value := range extra {
		prefix := key + "="
		filtered := env[:0]
		for _, item := range env {
			if !strings.HasPrefix(strings.ToUpper(item), strings.ToUpper(prefix)) {
				filtered = append(filtered, item)
			}
		}
		env = append(filtered, prefix+value)
	}
	return env
}
