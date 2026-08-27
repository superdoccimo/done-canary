package app

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionCommandReportsDevelopmentVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exit := Execute(context.Background(), []string{"version"}, &stdout, &stderr); exit != 0 {
		t.Fatalf("version exit: got %d; stderr=%q", exit, stderr.String())
	}
	if got, want := stdout.String(), "done-canary 0.1.0-dev\n"; got != want {
		t.Fatalf("version output: got %q, want %q", got, want)
	}
}

func TestVersionCommandSupportsLinkerInjection(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller path unavailable")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	goBinary := filepath.Join(runtime.GOROOT(), "bin", "go")
	outputBinary := filepath.Join(t.TempDir(), "done-canary")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
		outputBinary += ".exe"
	}
	const injectedVersion = "9.8.7-linker-test"
	ldflags := "-X github.com/superdoccimo/done-canary/internal/model.Version=" + injectedVersion
	build := exec.Command(goBinary, "build", "-trimpath", "-ldflags", ldflags, "-o", outputBinary, "./cmd/done-canary")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build injected version: %v\n%s", err, output)
	}
	output, err := exec.Command(outputBinary, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("run injected version: %v\n%s", err, output)
	}
	if got, want := strings.TrimSpace(string(output)), "done-canary "+injectedVersion; got != want {
		t.Fatalf("injected version: got %q, want %q", got, want)
	}
}
