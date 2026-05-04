// Command cron-doctor audits crontabs and prints a human-readable report.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/HeytalePazguato/cron-doctor/internal/audit"
	"github.com/HeytalePazguato/cron-doctor/internal/parser"
	"github.com/HeytalePazguato/cron-doctor/internal/report"
)

const usage = `cron-doctor — audit crontabs and cron expressions.

Usage:
  cron-doctor <path>                 Audit a crontab file.
  cron-doctor -                      Read crontab from stdin.
  cron-doctor "0 4 * * 1-5"          Audit a single quoted expression.
  cron-doctor 0 4 '*' '*' 1-5        Audit an expression as 5 separate args.
  cron-doctor 0 4 '*' '*' 1-5 cmd    5 schedule fields followed by a command.

Flags:
  --calendar    Print a 7-day calendar view of fire times.
  --json        Emit machine-readable JSON.
  --no-color    Disable ANSI colors even on a TTY.
  --version     Print version info and exit.
`

// Stamped at build time via -ldflags="-X main.version=... -X main.commit=... -X main.date=...".
// "dev" is the default when the binary is built without ldflags (e.g. go build, go install).
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("cron-doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	calendar := fs.Bool("calendar", false, "print a 7-day calendar view")
	jsonOut := fs.Bool("json", false, "emit JSON output")
	noColor := fs.Bool("no-color", false, "disable ANSI colors")
	showVersion := fs.Bool("version", false, "print version info and exit")
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(stdout, versionString())
		return nil
	}
	pos := fs.Args()
	if len(pos) == 0 {
		fmt.Fprint(stderr, usage)
		return fmt.Errorf("no input")
	}

	mode, src, isSystem, err := classify(pos, stdin)
	if err != nil {
		return err
	}

	pr, err := parser.Parse(src, isSystem)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	cfg := audit.Config{
		MailTo:   pr.EnvVars["MAILTO"],
		IsSystem: isSystem,
	}
	findings := audit.Run(pr.Lines, cfg)

	useColor := !*noColor && report.IsTerminal(stdout)

	switch {
	case *jsonOut:
		return report.JSON(stdout, pr.Lines, findings)
	case *calendar:
		report.Calendar(stdout, pr.Lines, useColor)
		return nil
	case mode == modeExpression:
		report.Expression(stdout, pr.Lines, findings, useColor)
		return nil
	default:
		report.Text(stdout, pr.Lines, findings, useColor)
		return nil
	}
}

type inputMode int

const (
	modeFile inputMode = iota
	modeStdin
	modeExpression
)

// classify decides whether the positional args refer to stdin, a file, or an inline expression,
// and returns an io.Reader producing the crontab text.
func classify(pos []string, stdin io.Reader) (inputMode, io.Reader, bool, error) {
	first := pos[0]
	if first == "-" {
		return modeStdin, stdin, false, nil
	}
	if len(pos) == 1 {
		if isExistingFile(first) {
			f, err := os.Open(first)
			if err != nil {
				return 0, nil, false, err
			}
			return modeFile, f, isSystemPath(first), nil
		}
		// Single arg, not a file: treat as quoted expression.
		return modeExpression, strings.NewReader(first + "\n"), false, nil
	}
	// Multiple positional args. Detect shell glob expansion mishap: more than 6 schedule-position
	// args where some resolve to real cwd files.
	if len(pos) > 6 {
		expanded := 0
		for _, a := range pos {
			if isExistingFile(a) {
				expanded++
			}
		}
		if expanded >= 2 {
			return 0, nil, false, fmt.Errorf("too many arguments. The shell likely expanded * to filenames.\n  Quote the expression: cron-doctor \"0 4 * * 1-5\"\n  Or escape the asterisks: cron-doctor 0 4 \\* \\* 1-5")
		}
	}
	// Treat as expression: rejoin args with spaces.
	expr := strings.Join(pos, " ")
	return modeExpression, strings.NewReader(expr + "\n"), false, nil
}

func isExistingFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func versionString() string {
	s := "cron-doctor " + version
	if commit != "" {
		s += " (" + shortCommit(commit)
		if date != "" {
			s += ", " + date
		}
		s += ")"
	}
	return s
}

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

func isSystemPath(p string) bool {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	abs = filepath.ToSlash(abs)
	if abs == "/etc/crontab" {
		return true
	}
	if strings.HasPrefix(abs, "/etc/cron.d/") {
		return true
	}
	return false
}
