package report

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/superdoccimo/done-canary/internal/jsonfile"
	"github.com/superdoccimo/done-canary/internal/model"
	"github.com/superdoccimo/done-canary/internal/runpath"
)

const maximumReportLogBytes = 3 << 20

var ansiPattern = regexp.MustCompile("\\x1b(?:\\[[0-?]*[ -/]*[@-~]|\\][^\\x07]*(?:\\x07|\\x1b\\\\))")

type view struct {
	Result     model.Result
	Rows       []row
	Stdout     string
	Stderr     string
	ScoreClass string
	Counts     resultCounts
	Partial    bool
	CardDetail string
}

type resultCounts struct {
	Passed     int
	Failed     int
	NotRun     int
	Applicable int
	Total      int
}

type row struct {
	Symbol  string
	Display string
	Status  model.Status
	Summary string
}

func WriteAll(paths runpath.Paths, result model.Result) error {
	if err := Validate(result); err != nil {
		return err
	}
	if err := jsonfile.Write(paths.Result, result); err != nil {
		return err
	}
	if err := WriteHTML(paths.HTML, paths, result); err != nil {
		return err
	}
	if err := WriteSVG(paths.SVG, result); err != nil {
		return err
	}
	return nil
}

func Validate(result model.Result) error {
	if result.SchemaVersion != model.SchemaVersion {
		return fmt.Errorf("unsupported result schema %q", result.SchemaVersion)
	}
	if result.RunID == "" || result.StartedAt.IsZero() || result.EndedAt.IsZero() {
		return errors.New("result is missing required run metadata")
	}
	if result.Agent.Name == "" || result.Host.OS == "" || result.Host.Arch == "" {
		return errors.New("result is missing agent or host metadata")
	}
	if result.FixtureVersion != model.FixtureVersion {
		return fmt.Errorf("unsupported fixture version %q", result.FixtureVersion)
	}
	if result.InfrastructureStatus != "ok" && result.InfrastructureStatus != "error" && result.InfrastructureStatus != "interrupted" {
		return fmt.Errorf("invalid infrastructure status %q", result.InfrastructureStatus)
	}
	if len(result.Canaries) != len(model.CanaryOrder) || result.Score.Total != len(model.CanaryOrder) {
		return errors.New("result must contain exactly seven canaries")
	}
	passed := 0
	for index, expected := range model.CanaryOrder {
		canary := result.Canaries[index]
		if canary.ID != expected.ID {
			return fmt.Errorf("canary %d has ID %q, want %q", index, canary.ID, expected.ID)
		}
		if canary.Status != model.Pass && canary.Status != model.Fail && canary.Status != model.NotRun {
			return fmt.Errorf("canary %s has invalid status %q", canary.ID, canary.Status)
		}
		if canary.Status == model.Pass {
			passed++
		}
	}
	if result.Score.Passed != passed {
		return fmt.Errorf("score says %d passed, counted %d", result.Score.Passed, passed)
	}
	return nil
}

func Terminal(writer io.Writer, result model.Result) {
	fmt.Fprintln(writer, "DONECANARY")
	fmt.Fprintln(writer, "────────────")
	for index, canary := range result.Canaries {
		symbol := "✗"
		if canary.Status == model.Pass {
			symbol = "✓"
		}
		if canary.Status == model.NotRun {
			symbol = "–"
		}
		display := canary.ID
		if index < len(model.CanaryOrder) {
			display = model.CanaryOrder[index].Display
		}
		fmt.Fprintf(writer, "%s %s\n", symbol, display)
	}
	counts := countResults(result)
	if counts.NotRun > 0 {
		fmt.Fprintf(writer, "\nRESULT\n%d PASS\n%d FAIL\n%d NOT RUN\n\nCOVERAGE %d / %d\n",
			counts.Passed, counts.Failed, counts.NotRun, counts.Applicable, counts.Total)
		return
	}
	fmt.Fprintf(writer, "\nSCORE %d / %d\n", result.Score.Passed, result.Score.Total)
}

func WriteHTML(path string, paths runpath.Paths, result model.Result) error {
	stdout, err := readBounded(paths.StdoutLog)
	if err != nil {
		return err
	}
	stderr, err := readBounded(paths.StderrLog)
	if err != nil {
		return err
	}
	rows := make([]row, 0, len(result.Canaries))
	for index, canary := range result.Canaries {
		symbol := "✗"
		if canary.Status == model.Pass {
			symbol = "✓"
		}
		if canary.Status == model.NotRun {
			symbol = "–"
		}
		display := canary.ID
		if index < len(model.CanaryOrder) {
			display = model.CanaryOrder[index].Display
		}
		rows = append(rows, row{Symbol: symbol, Display: display, Status: canary.Status, Summary: canary.Summary})
	}
	scoreClass := "partial"
	if result.Score.Passed == result.Score.Total {
		scoreClass = "perfect"
	}
	counts := countResults(result)
	data := view{
		Result: result, Rows: rows, Stdout: StripANSI(stdout), Stderr: StripANSI(stderr), ScoreClass: scoreClass,
		Counts: counts, Partial: counts.NotRun > 0, CardDetail: cardDetail(result),
	}
	parsed, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	executeErr := parsed.Execute(file, data)
	closeErr := file.Close()
	if executeErr != nil {
		return executeErr
	}
	return closeErr
}

func WriteSVG(path string, result model.Result) error {
	counts := countResults(result)
	var output strings.Builder
	output.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630" role="img" aria-labelledby="title desc">`)
	output.WriteString(`<title id="title">DoneCanary scorecard</title><desc id="desc">`)
	if counts.NotRun > 0 {
		output.WriteString(html.EscapeString(fmt.Sprintf("%d pass, %d fail, %d not run; coverage %d of %d", counts.Passed, counts.Failed, counts.NotRun, counts.Applicable, counts.Total)))
	} else {
		output.WriteString(html.EscapeString(fmt.Sprintf("Score %d out of %d", result.Score.Passed, result.Score.Total)))
	}
	output.WriteString(`</desc><rect width="1200" height="630" fill="#08111f"/><text x="72" y="90" fill="#9fb3c8" font-family="Arial,sans-serif" font-size="28" font-weight="700" letter-spacing="3">DONECANARY</text>`)
	rowStart, rowGap := 245, 45
	if counts.NotRun > 0 {
		output.WriteString(fmt.Sprintf(`<text x="72" y="160" fill="#f7fafc" font-family="Arial,sans-serif" font-size="46" font-weight="800">%d PASS · %d FAIL · %d NOT RUN</text>`, counts.Passed, counts.Failed, counts.NotRun))
		output.WriteString(fmt.Sprintf(`<text x="72" y="207" fill="#cbd5e1" font-family="Arial,sans-serif" font-size="29" font-weight="700">Coverage %d/%d</text>`, counts.Applicable, counts.Total))
		output.WriteString(fmt.Sprintf(`<text x="72" y="244" fill="#94a3b8" font-family="Arial,sans-serif" font-size="22">%s</text>`, html.EscapeString(cardDetail(result))))
		rowStart, rowGap = 292, 39
	} else {
		output.WriteString(fmt.Sprintf(`<text x="72" y="185" fill="#f7fafc" font-family="Arial,sans-serif" font-size="78" font-weight="800">%d / %d</text>`, result.Score.Passed, result.Score.Total))
	}
	for index, canary := range result.Canaries {
		y := rowStart + index*rowGap
		color, symbol := "#ff6b6b", "✗"
		if canary.Status == model.Pass {
			color, symbol = "#4ade80", "✓"
		}
		if canary.Status == model.NotRun {
			color, symbol = "#94a3b8", "–"
		}
		display := canary.ID
		if index < len(model.CanaryOrder) {
			display = model.CanaryOrder[index].Display
		}
		output.WriteString(fmt.Sprintf(`<text x="78" y="%d" fill="%s" font-family="Arial,sans-serif" font-size="25" font-weight="700">%s</text>`, y, color, symbol))
		output.WriteString(fmt.Sprintf(`<text x="120" y="%d" fill="#e2e8f0" font-family="Arial,sans-serif" font-size="23">%s</text>`, y, html.EscapeString(display)))
	}
	output.WriteString(`<text x="72" y="598" fill="#cbd5e1" font-family="Arial,sans-serif" font-size="20" font-weight="700">DoneCanary · @superdoccimo</text>`)
	output.WriteString(`<text x="430" y="598" fill="#718096" font-family="Arial,sans-serif" font-size="18">Generated locally by DoneCanary v0.1</text></svg>`)
	return os.WriteFile(path, []byte(output.String()), 0o600)
}

func countResults(result model.Result) resultCounts {
	counts := resultCounts{Total: len(result.Canaries)}
	for _, canary := range result.Canaries {
		switch canary.Status {
		case model.Pass:
			counts.Passed++
		case model.Fail:
			counts.Failed++
		case model.NotRun:
			counts.NotRun++
		}
	}
	counts.Applicable = counts.Passed + counts.Failed
	return counts
}

func cardDetail(result model.Result) string {
	name := result.Agent.Name
	switch name {
	case "codex":
		name = "Codex"
	case "claude":
		name = "Claude"
	}
	version := result.Agent.Version
	if result.Agent.Name == "codex" {
		version = strings.TrimPrefix(version, "codex-cli ")
	}
	host := result.Host.OS
	switch host {
	case "windows":
		host = "Windows"
	case "linux":
		host = "Linux"
	case "darwin":
		host = "macOS"
	}
	agent := strings.TrimSpace(strings.TrimSpace(name + " " + version))
	return agent + " · " + host
}

func StripANSI(value string) string { return ansiPattern.ReplaceAllString(value, "") }

func readBounded(path string) (string, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer file.Close()
	var buffer bytes.Buffer
	if _, err := io.CopyN(&buffer, file, maximumReportLogBytes+1); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	value := buffer.String()
	if len(value) > maximumReportLogBytes {
		value = value[:maximumReportLogBytes] + "\n[done-canary: report log truncated]\n"
	}
	return value, nil
}

const htmlTemplate = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="icon" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 16'%3E%3Crect width='16' height='16' rx='3' fill='%2308111f'/%3E%3Cpath d='m4 8 2 2 5-5' fill='none' stroke='%234ade80' stroke-width='2'/%3E%3C/svg%3E">
<title>{{if .Partial}}DoneCanary — {{.Counts.Passed}} PASS · {{.Counts.Failed}} FAIL · {{.Counts.NotRun}} NOT RUN{{else}}DoneCanary — {{.Result.Score.Passed}} / {{.Result.Score.Total}}{{end}}</title>
<style>
:root{color-scheme:dark;font-family:Inter,ui-sans-serif,system-ui,sans-serif;background:#08111f;color:#e2e8f0}body{margin:0;padding:40px 20px}.shell{max-width:900px;margin:auto}.eyebrow{color:#9fb3c8;letter-spacing:.18em;font-weight:800}.hero{display:flex;align-items:end;justify-content:space-between;gap:24px;border-bottom:1px solid #25334a;padding-bottom:28px}.score{font-size:72px;font-weight:850;line-height:1}.score.perfect{color:#4ade80}.score.partial{color:#fbbf24}.coverage-panel{text-align:right}.outcome{font-size:26px;font-weight:800;white-space:nowrap}.pass-count{color:#4ade80}.fail-count{color:#ff6b6b}.not-run-count{color:#94a3b8}.coverage{font-size:22px;font-weight:750;color:#cbd5e1;margin-top:9px}.card-detail{font-size:14px;color:#94a3b8;margin-top:7px}.rows{display:grid;gap:10px;margin:28px 0}.row{display:grid;grid-template-columns:32px 1fr;gap:12px;background:#101d30;border:1px solid #26364f;border-radius:10px;padding:15px}.row.pass .symbol{color:#4ade80}.row.fail .symbol{color:#ff6b6b}.row.not_run .symbol{color:#94a3b8}.name{font-weight:750}.summary{color:#9fb3c8;font-size:14px;margin-top:4px}.meta{color:#94a3b8;font-size:14px}details{margin-top:16px;border:1px solid #26364f;border-radius:10px;padding:14px}pre{white-space:pre-wrap;overflow-wrap:anywhere;background:#050b14;padding:14px;border-radius:8px;max-height:360px;overflow:auto}.generator{display:flex;justify-content:space-between;gap:18px;margin-top:28px;color:#718096;font-size:13px}.generator strong{color:#cbd5e1;font-size:14px}@media(max-width:700px){.hero{display:block}.score{margin-top:18px;font-size:56px}.coverage-panel{text-align:left;margin-top:20px}.outcome{white-space:normal}.generator{display:block}.generator span{display:block;margin-top:6px}}
</style></head><body><main class="shell"><div class="hero"><div><div class="eyebrow">DONECANARY</div><h1>Does your coding agent finish?</h1><div class="meta">Run {{.Result.RunID}} · {{.Result.Agent.Name}} {{.Result.Agent.Version}}</div></div>
{{if .Partial}}<div class="coverage-panel"><div class="outcome"><span class="pass-count">{{.Counts.Passed}} PASS</span> · <span class="fail-count">{{.Counts.Failed}} FAIL</span> · <span class="not-run-count">{{.Counts.NotRun}} NOT RUN</span></div><div class="coverage">Coverage {{.Counts.Applicable}}/{{.Counts.Total}}</div><div class="card-detail">{{.CardDetail}}</div></div>{{else}}<div class="score {{.ScoreClass}}">{{.Result.Score.Passed}} / {{.Result.Score.Total}}</div>{{end}}</div>
<section class="rows">{{range .Rows}}<div class="row {{.Status}}"><div class="symbol">{{.Symbol}}</div><div><div class="name">{{.Display}}</div><div class="summary">{{.Summary}}</div></div></div>{{end}}</section>
<details><summary>Agent stdout</summary><pre>{{.Stdout}}</pre></details><details><summary>Agent stderr</summary><pre>{{.Stderr}}</pre></details>
<div class="generator"><strong>DoneCanary · @superdoccimo</strong><span>Generated locally by DoneCanary v0.1</span></div></main></body></html>`
