package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const TruncationMarker = "\n[done-canary: log truncated]\n"

type Request struct {
	Path        string
	Args        []string
	Dir         string
	Env         map[string]string
	StdoutPath  string
	StderrPath  string
	MaxLogBytes int64
	Timeout     time.Duration
	Stdin       string
}

type Result struct {
	ExitCode    int
	TimedOut    bool
	Interrupted bool
	Truncated   bool
	StartedAt   time.Time
	EndedAt     time.Time
}

func Run(ctx context.Context, request Request) (Result, error) {
	if request.Path == "" {
		return Result{}, errors.New("process path is empty")
	}
	if request.MaxLogBytes <= 0 {
		return Result{}, errors.New("maximum log size must be positive")
	}
	stdout, err := os.OpenFile(request.StdoutPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return Result{}, fmt.Errorf("open stdout log: %w", err)
	}
	defer stdout.Close()
	stderr, err := os.OpenFile(request.StderrPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return Result{}, fmt.Errorf("open stderr log: %w", err)
	}
	defer stderr.Close()
	outWriter := newBoundedWriter(stdout, request.MaxLogBytes)
	errWriter := newBoundedWriter(stderr, request.MaxLogBytes)
	cmd := exec.Command(request.Path, request.Args...)
	cmd.Dir = request.Dir
	cmd.Env = mergeEnv(request.Env)
	cmd.Stdout = outWriter
	cmd.Stderr = errWriter
	if request.Stdin != "" {
		cmd.Stdin = strings.NewReader(request.Stdin)
	}
	configureCommand(cmd)
	result := Result{StartedAt: time.Now().UTC(), ExitCode: -1}
	if err := cmd.Start(); err != nil {
		result.EndedAt = time.Now().UTC()
		return result, fmt.Errorf("start agent process: %w", err)
	}
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	var waitErr error
	var runContext context.Context = ctx
	var cancel context.CancelFunc
	if request.Timeout > 0 {
		runContext, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	}
	select {
	case waitErr = <-wait:
	case <-runContext.Done():
		result.TimedOut = errors.Is(runContext.Err(), context.DeadlineExceeded)
		result.Interrupted = !result.TimedOut
		_ = killProcessTree(cmd.Process.Pid)
		waitErr = <-wait
	}
	result.EndedAt = time.Now().UTC()
	result.Truncated = outWriter.Truncated() || errWriter.Truncated()
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if result.Interrupted {
		result.ExitCode = 130
	}
	if result.TimedOut {
		return result, context.DeadlineExceeded
	}
	if result.Interrupted {
		return result, context.Canceled
	}
	var exitErr *exec.ExitError
	if waitErr != nil && !errors.As(waitErr, &exitErr) {
		return result, fmt.Errorf("wait for agent process: %w", waitErr)
	}
	return result, nil
}

type boundedWriter struct {
	mu        sync.Mutex
	dst       io.Writer
	remaining int64
	truncated bool
	marked    bool
}

func newBoundedWriter(dst io.Writer, maximum int64) *boundedWriter {
	return &boundedWriter{dst: dst, remaining: maximum}
}

func (writer *boundedWriter) Write(data []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	original := len(data)
	written := int64(0)
	if writer.remaining > 0 {
		count := int64(len(data))
		if count > writer.remaining {
			count = writer.remaining
		}
		if _, err := writer.dst.Write(data[:count]); err != nil {
			return 0, err
		}
		writer.remaining -= count
		written = count
	}
	if written < int64(original) {
		writer.truncated = true
		if !writer.marked {
			writer.marked = true
			if _, err := io.WriteString(writer.dst, TruncationMarker); err != nil {
				return 0, err
			}
		}
	}
	return original, nil
}

func (writer *boundedWriter) Truncated() bool {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.truncated
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
