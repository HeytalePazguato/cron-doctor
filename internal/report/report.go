// Package report renders audit results in text, JSON, or calendar form.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/HeytalePazguato/cron-doctor/internal/audit"
	"github.com/HeytalePazguato/cron-doctor/internal/parser"
)

// IsTerminal reports whether w refers to a character device (TTY).
// It is conservative: if w isn't an *os.File, the answer is false.
func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}

// ANSI color helpers. Empty strings when color is disabled so callers don't
// have to branch.
type palette struct {
	red, yellow, blue, gray, bold, reset string
}

func newPalette(useColor bool) palette {
	if !useColor {
		return palette{}
	}
	return palette{
		red:    "\x1b[31m",
		yellow: "\x1b[33m",
		blue:   "\x1b[34m",
		gray:   "\x1b[90m",
		bold:   "\x1b[1m",
		reset:  "\x1b[0m",
	}
}

// Text writes the default per-line report to w.
func Text(w io.Writer, lines []*parser.Line, findings audit.Result, useColor bool) {
	p := newPalette(useColor)
	for i, ln := range lines {
		writeLine(w, p, ln, findings[i], false)
	}
	writeSummary(w, p, lines, findings)
}

// Expression writes the same body as Text but without the "Line N:" prefix.
func Expression(w io.Writer, lines []*parser.Line, findings audit.Result, useColor bool) {
	p := newPalette(useColor)
	for i, ln := range lines {
		writeLine(w, p, ln, findings[i], true)
	}
	writeSummary(w, p, lines, findings)
}

func writeLine(w io.Writer, p palette, ln *parser.Line, fs []audit.Finding, expressionMode bool) {
	switch ln.Type {
	case parser.LineBlank, parser.LineComment, parser.LineEnv:
		return
	case parser.LineParseError:
		if expressionMode {
			fmt.Fprintf(w, "%sParse error:%s %s\n", p.red, p.reset, ln.Error)
		} else {
			fmt.Fprintf(w, "%sLine %d:%s %s\n  %s✗ ERROR:%s %s\n\n",
				p.bold, ln.Number, p.reset, ln.Raw, p.red, p.reset, ln.Error)
		}
		return
	}

	if expressionMode {
		fmt.Fprintf(w, "%sSchedule:%s %s\n", p.bold, p.reset, ln.Schedule)
	} else {
		fmt.Fprintf(w, "%sLine %d:%s %s\n", p.bold, ln.Number, p.reset, ln.Raw)
	}

	if ln.Schedule != "" {
		fmt.Fprintf(w, "  %s\n", audit.Explain(ln.Schedule))
	}
	if ln.IsReboot {
		fmt.Fprintf(w, "  %sFires once when cron starts; no recurring schedule.%s\n", p.gray, p.reset)
	} else if ln.CronSchedule != nil {
		runs := nextRuns(ln, time.Now(), 3)
		fmt.Fprintf(w, "  Next: %s.\n", formatRuns(runs))
	}

	if expressionMode && ln.Command == "" {
		fmt.Fprintf(w, "  %sNo command provided — schedule-only audit.%s\n", p.gray, p.reset)
	}

	for _, f := range fs {
		fmt.Fprintf(w, "  %s\n", formatFinding(p, f))
	}
	fmt.Fprintln(w)
}

func writeSummary(w io.Writer, p palette, lines []*parser.Line, findings audit.Result) {
	s := audit.Summarize(lines, findings)
	fmt.Fprintf(w, "%s%d lines parsed%s — %s%d errors%s, %s%d warnings%s, %s%d info%s.\n",
		p.bold, s.Lines, p.reset,
		p.red, s.Errors, p.reset,
		p.yellow, s.Warnings, p.reset,
		p.blue, s.Info, p.reset)
}

func formatFinding(p palette, f audit.Finding) string {
	switch f.Severity {
	case audit.SevError:
		return fmt.Sprintf("%s✗ ERROR:%s %s", p.red, p.reset, f.Message)
	case audit.SevWarn:
		return fmt.Sprintf("%s⚠ WARN:%s %s", p.yellow, p.reset, f.Message)
	case audit.SevInfo:
		return fmt.Sprintf("%sℹ INFO:%s %s", p.blue, p.reset, f.Message)
	}
	return f.Message
}

func nextRuns(ln *parser.Line, now time.Time, n int) []time.Time {
	out := make([]time.Time, 0, n)
	t := now
	for i := 0; i < n; i++ {
		t = ln.CronSchedule.Next(t)
		out = append(out, t)
	}
	return out
}

func formatRuns(ts []time.Time) string {
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = t.Format("2006-01-02 15:04")
	}
	return joinComma(parts)
}

func joinComma(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += ", " + p
	}
	return out
}

// JSON output ------------------------------------------------------------

type jsonLine struct {
	LineNumber  int              `json:"line_number"`
	Raw         string           `json:"raw"`
	Schedule    string           `json:"schedule,omitempty"`
	Command     string           `json:"command,omitempty"`
	User        string           `json:"user,omitempty"`
	Explanation string           `json:"explanation,omitempty"`
	NextRuns    []string         `json:"next_runs,omitempty"`
	Findings    []audit.Finding  `json:"findings"`
}

type jsonOutput struct {
	Lines   []jsonLine    `json:"lines"`
	Summary audit.Summary `json:"summary"`
}

// JSON serializes the result for machine consumers.
func JSON(w io.Writer, lines []*parser.Line, findings audit.Result) error {
	out := jsonOutput{}
	now := time.Now()
	for i, ln := range lines {
		if ln.Type != parser.LineJob && ln.Type != parser.LineParseError {
			continue
		}
		jl := jsonLine{
			LineNumber: ln.Number,
			Raw:        ln.Raw,
			Schedule:   ln.Schedule,
			Command:    ln.Command,
			User:       ln.User,
			Findings:   findings[i],
		}
		if ln.Schedule != "" {
			jl.Explanation = audit.Explain(ln.Schedule)
		}
		if ln.CronSchedule != nil {
			for _, t := range nextRuns(ln, now, 3) {
				jl.NextRuns = append(jl.NextRuns, t.UTC().Format(time.RFC3339))
			}
		}
		if jl.Findings == nil {
			jl.Findings = []audit.Finding{}
		}
		out.Lines = append(out.Lines, jl)
	}
	out.Summary = audit.Summarize(lines, findings)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}
