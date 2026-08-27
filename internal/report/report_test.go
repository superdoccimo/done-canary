package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/superdoccimo/done-canary/internal/model"
	"github.com/superdoccimo/done-canary/internal/runpath"
)

func sampleResult(passed int) model.Result {
	canaries := make([]model.Canary, 0, 7)
	for index, item := range model.CanaryOrder {
		status := model.Fail
		if index < passed {
			status = model.Pass
		}
		canaries = append(canaries, model.Canary{ID: item.ID, Status: status, Summary: "safe summary", Evidence: []string{"evidence"}})
	}
	return model.Result{
		SchemaVersion: model.SchemaVersion, RunID: "sample", StartedAt: time.Now().UTC(), EndedAt: time.Now().UTC(),
		Agent: model.AgentInfo{Name: "fake-pass", Version: model.Version, Invocation: []string{"internal"}},
		Host:  model.HostInfo{OS: "test", Arch: "test"}, FixtureVersion: model.FixtureVersion,
		InfrastructureStatus: "ok", Canaries: canaries, Score: model.Score{Passed: passed, Total: 7},
	}
}

func sampleWindowsCodexResult() model.Result {
	result := sampleResult(5)
	result.Agent = model.AgentInfo{Name: "codex", Version: "codex-cli 0.147.0"}
	result.Host = model.HostInfo{OS: "windows", Arch: "amd64"}
	for index := range result.Canaries {
		result.Canaries[index].Status = model.Pass
	}
	for _, index := range []int{4, 6} {
		result.Canaries[index].Status = model.NotRun
		result.Canaries[index].Summary = "safe sandbox capability limitation"
	}
	result.Score = model.Score{Passed: 5, Total: 7}
	return result
}

func TestHTMLIsEscapedAndANSIIsRemoved(t *testing.T) {
	dir := t.TempDir()
	paths := runpath.Paths{StdoutLog: filepath.Join(dir, "stdout"), StderrLog: filepath.Join(dir, "stderr")}
	if err := os.WriteFile(paths.StdoutLog, []byte("\x1b[31m<script>alert(1)</script>\x1b[0m"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.StderrLog, []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "report.html")
	if err := WriteHTML(output, paths, sampleResult(7)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "<script>alert") {
		t.Fatal("agent HTML was not escaped")
	}
	if !strings.Contains(text, "&lt;script&gt;alert") {
		t.Fatal("escaped text missing")
	}
	if strings.Contains(text, "\x1b[31m") {
		t.Fatal("ANSI sequence remained")
	}
	if !strings.Contains(text, "DoneCanary · @superdoccimo") {
		t.Fatal("visible creator attribution missing from HTML report")
	}
}

func TestSVGPerfectAndPartial(t *testing.T) {
	for _, passed := range []int{7, 5} {
		path := filepath.Join(t.TempDir(), "score.svg")
		if err := WriteSVG(path, sampleResult(passed)); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.Contains(text, "Generated locally by DoneCanary v0.1") {
			t.Fatal("generator missing")
		}
		if !strings.Contains(text, "DoneCanary · @superdoccimo") {
			t.Fatal("visible creator attribution missing from SVG scorecard")
		}
	}
}

func TestTerminalUsesCoverageForNotRunCanaries(t *testing.T) {
	var output bytes.Buffer
	Terminal(&output, sampleWindowsCodexResult())
	text := output.String()
	for _, want := range []string{"RESULT\n5 PASS\n0 FAIL\n2 NOT RUN", "COVERAGE 5 / 7"} {
		if !strings.Contains(text, want) {
			t.Fatalf("terminal output %q does not contain %q", text, want)
		}
	}
	if strings.Contains(text, "SCORE 5 / 7") {
		t.Fatalf("partial coverage was presented as a score: %q", text)
	}

	output.Reset()
	Terminal(&output, sampleResult(7))
	if !strings.Contains(output.String(), "SCORE 7 / 7") {
		t.Fatalf("full result lost score presentation: %q", output.String())
	}
}

func TestHTMLAndSVGUseNeutralPartialCoveragePresentation(t *testing.T) {
	dir := t.TempDir()
	paths := runpath.Paths{StdoutLog: filepath.Join(dir, "stdout"), StderrLog: filepath.Join(dir, "stderr")}
	htmlPath := filepath.Join(dir, "report.html")
	svgPath := filepath.Join(dir, "scorecard.svg")
	result := sampleWindowsCodexResult()
	if err := WriteHTML(htmlPath, paths, result); err != nil {
		t.Fatal(err)
	}
	if err := WriteSVG(svgPath, result); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{"HTML": htmlPath, "SVG": svgPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, want := range []string{"5 PASS", "0 FAIL", "2 NOT RUN", "Coverage 5/7", "Codex 0.147.0 · Windows"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s output does not contain %q", name, want)
			}
		}
		if strings.Contains(text, ">5 / 7<") {
			t.Fatalf("%s presents partial coverage as a large score", name)
		}
	}

	htmlData, _ := os.ReadFile(htmlPath)
	if !strings.Contains(string(htmlData), `class="row not_run"`) || !strings.Contains(string(htmlData), `.row.not_run .symbol{color:#94a3b8}`) {
		t.Fatal("HTML NOT RUN row is not rendered with the neutral status style")
	}
	svgData, _ := os.ReadFile(svgPath)
	if !strings.Contains(string(svgData), `fill="#94a3b8" font-family="Arial,sans-serif" font-size="25" font-weight="700">–</text>`) {
		t.Fatal("SVG NOT RUN row is not rendered with the neutral status color")
	}
}

func TestFullHTMLAndSVGRetainScorePresentation(t *testing.T) {
	dir := t.TempDir()
	paths := runpath.Paths{StdoutLog: filepath.Join(dir, "stdout"), StderrLog: filepath.Join(dir, "stderr")}
	htmlPath := filepath.Join(dir, "report.html")
	svgPath := filepath.Join(dir, "scorecard.svg")
	if err := WriteHTML(htmlPath, paths, sampleResult(7)); err != nil {
		t.Fatal(err)
	}
	if err := WriteSVG(svgPath, sampleResult(7)); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{htmlPath, svgPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), ">7 / 7<") {
			t.Fatalf("full result in %s lost score display", path)
		}
	}
}

func TestValidateRejectsWrongOrder(t *testing.T) {
	result := sampleResult(7)
	result.Canaries[0], result.Canaries[1] = result.Canaries[1], result.Canaries[0]
	if err := Validate(result); err == nil {
		t.Fatal("expected validation failure")
	}
}
