package report

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/superdoccimo/done-canary/internal/model"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller path unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestCheckedInExamplesValidate(t *testing.T) {
	root := repositoryRoot(t)
	for _, name := range []string{"result-7-of-7.json", "result-5-of-7.json"} {
		data, err := os.ReadFile(filepath.Join(root, "examples", name))
		if err != nil {
			t.Fatal(err)
		}
		var result model.Result
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := Validate(result); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestSchemaAndSVGAssetsParse(t *testing.T) {
	root := repositoryRoot(t)
	schema, err := os.ReadFile(filepath.Join(root, "schemas", "result-v0.1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schemaValue map[string]any
	if err := json.Unmarshal(schema, &schemaValue); err != nil {
		t.Fatal(err)
	}
	if schemaValue["$schema"] == nil {
		t.Fatal("schema declaration missing")
	}
	for _, name := range []string{"scorecard-7-of-7.svg", "scorecard-5-of-7.svg"} {
		data, err := os.ReadFile(filepath.Join(root, "assets", name))
		if err != nil {
			t.Fatal(err)
		}
		decoder := xml.NewDecoder(bytes.NewReader(data))
		for {
			if _, err := decoder.Token(); err != nil {
				if err == io.EOF {
					break
				}
				t.Fatalf("%s: %v", name, err)
			}
		}
		if !bytes.Contains(data, []byte("DoneCanary · @superdoccimo")) {
			t.Fatalf("%s is missing visible creator attribution", name)
		}
	}
}
