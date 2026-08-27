package security

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller path unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestApplicationHasNoNetworkClientImports(t *testing.T) {
	root := projectRoot(t)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "bin" || name == "dist" || name == "output" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range parsed.Imports {
			name, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if name == "net" || strings.HasPrefix(name, "net/") {
				t.Errorf("application source %s imports network package %s", path, name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestApplicationDoesNotCaptureEnvironment(t *testing.T) {
	root := projectRoot(t)
	for _, path := range []string{
		filepath.Join(root, "internal", "model", "model.go"),
		filepath.Join(root, "internal", "app", "app.go"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, forbidden := range []string{"os.Environ()", "API_KEY", "AUTH_TOKEN"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s contains forbidden environment capture token %q", path, forbidden)
			}
		}
	}
}
