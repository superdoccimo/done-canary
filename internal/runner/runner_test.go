package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("DONECANARY_RUNNER_HELPER") != "1" {
		return
	}
	mode := os.Getenv("DONECANARY_RUNNER_MODE")
	if mode == "sleep" {
		time.Sleep(30 * time.Second)
		return
	}
	if mode == "large" {
		fmt.Print(strings.Repeat("x", 4096))
		return
	}
	if mode == "tree" {
		command := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
		command.Env = append(os.Environ(), "DONECANARY_RUNNER_HELPER=1", "DONECANARY_RUNNER_MODE=sleep")
		if err := command.Start(); err != nil {
			os.Exit(3)
		}
		if err := os.WriteFile(os.Getenv("DONECANARY_CHILD_PID_FILE"), []byte(strconv.Itoa(command.Process.Pid)), 0o600); err != nil {
			os.Exit(4)
		}
		_ = command.Wait()
		return
	}
}

func helperRequest(t *testing.T, mode string) Request {
	t.Helper()
	return Request{
		Path: os.Args[0], Args: []string{"-test.run=TestHelperProcess"},
		Dir: t.TempDir(), Env: map[string]string{
			"DONECANARY_RUNNER_HELPER": "1", "DONECANARY_RUNNER_MODE": mode,
		},
		StdoutPath:  filepath.Join(t.TempDir(), "stdout.log"),
		StderrPath:  filepath.Join(t.TempDir(), "stderr.log"),
		MaxLogBytes: 128, Timeout: 10 * time.Second,
	}
}

func TestBoundedLogs(t *testing.T) {
	request := helperRequest(t, "large")
	result, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated {
		t.Fatal("expected truncation")
	}
	data, err := os.ReadFile(request.StdoutPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), TruncationMarker) {
		t.Fatal("missing truncation marker")
	}
}

func TestTimeout(t *testing.T) {
	request := helperRequest(t, "sleep")
	request.Timeout = 200 * time.Millisecond
	started := time.Now()
	result, err := Run(context.Background(), request)
	if err == nil || !result.TimedOut {
		t.Fatalf("got result %+v error %v", result, err)
	}
	if time.Since(started) > 5*time.Second {
		t.Fatal("timeout cleanup was too slow")
	}
}

func TestCancellationReturnsInterrupted(t *testing.T) {
	request := helperRequest(t, "sleep")
	request.Timeout = 10 * time.Second
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)
	result, err := Run(ctx, request)
	if err == nil || !result.Interrupted || result.ExitCode != 130 {
		t.Fatalf("got result %+v error %v", result, err)
	}
}

func TestTimeoutTerminatesDescendants(t *testing.T) {
	request := helperRequest(t, "tree")
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	request.Env["DONECANARY_CHILD_PID_FILE"] = pidFile
	request.Timeout = 500 * time.Millisecond
	result, err := Run(context.Background(), request)
	if err == nil || !result.TimedOut {
		t.Fatalf("got result %+v error %v", result, err)
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if processAlive(pid) {
		t.Fatalf("descendant process %d is still alive", pid)
	}
}
