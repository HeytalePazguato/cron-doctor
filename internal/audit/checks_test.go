package audit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/HeytalePazguato/cron-doctor/internal/parser"
)

func isPosix() bool { return runtime.GOOS != "windows" }

func parseString(t *testing.T, s string, system bool) []*parser.Line {
	t.Helper()
	pr, err := parser.Parse(strings.NewReader(s), system)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return pr.Lines
}

func findingCodes(fs []Finding) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Code
	}
	return out
}

func hasCode(fs []Finding, code string) bool {
	for _, f := range fs {
		if f.Code == code {
			return true
		}
	}
	return false
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
}

func TestCheckMissingScript(t *testing.T) {
	lines := parseString(t, "0 5 * * * /opt/definitely/not/here.sh > /dev/null 2>&1\n", false)
	r := Run(lines, Config{Now: fixedNow})
	if !hasCode(r[0], "missing_script") {
		t.Errorf("missing_script not raised; got %v", findingCodes(r[0]))
	}
}

func TestCheckMissingTimeout(t *testing.T) {
	lines := parseString(t, "* * * * * curl https://example.com > /dev/null 2>&1\n", false)
	r := Run(lines, Config{Now: fixedNow, MailTo: "ops@example.com"})
	if !hasCode(r[0], "missing_timeout") {
		t.Errorf("missing_timeout not raised; got %v", findingCodes(r[0]))
	}

	lines = parseString(t, "* * * * * curl --max-time 30 https://example.com > /dev/null 2>&1\n", false)
	r = Run(lines, Config{Now: fixedNow, MailTo: "ops@example.com"})
	if hasCode(r[0], "missing_timeout") {
		t.Errorf("missing_timeout should not fire when --max-time is present; got %v", findingCodes(r[0]))
	}
}

func TestCheckMissingRedirection(t *testing.T) {
	// MAILTO empty + no redirection = warn.
	lines := parseString(t, "0 5 * * * /usr/local/bin/x.sh\n", false)
	r := Run(lines, Config{Now: fixedNow})
	if !hasCode(r[0], "missing_redirection") {
		t.Errorf("missing_redirection not raised; got %v", findingCodes(r[0]))
	}
	// MAILTO set silences the warning.
	r = Run(lines, Config{Now: fixedNow, MailTo: "ops@example.com"})
	if hasCode(r[0], "missing_redirection") {
		t.Errorf("missing_redirection should be silenced when MAILTO is set")
	}
}

func TestCheckRunAsRoot(t *testing.T) {
	lines := parseString(t,
		"0 5 * * * root /home/alice/scripts/run.sh > /dev/null 2>&1\n",
		true)
	r := Run(lines, Config{Now: fixedNow, IsSystem: true, MailTo: "ops@example.com"})
	if !hasCode(r[0], "root_in_user_path") {
		t.Errorf("root_in_user_path not raised; got %v", findingCodes(r[0]))
	}
}

func TestCheckOverlap(t *testing.T) {
	src := "0 3 * * * /tmp/sync.sh > /dev/null 2>&1\n2 3 * * * /tmp/sync.sh > /dev/null 2>&1\n"
	lines := parseString(t, src, false)
	r := Run(lines, Config{Now: fixedNow, MailTo: "ops@example.com"})
	if !hasCode(r[0], "overlap_risk") {
		t.Errorf("overlap_risk not raised on line 0; got %v", findingCodes(r[0]))
	}
	if !hasCode(r[1], "overlap_risk") {
		t.Errorf("overlap_risk not raised on line 1; got %v", findingCodes(r[1]))
	}
}

func TestCheckNoFlock(t *testing.T) {
	lines := parseString(t, "*/5 * * * * /usr/local/bin/x.sh > /dev/null 2>&1\n", false)
	r := Run(lines, Config{Now: fixedNow, MailTo: "ops@example.com"})
	if !hasCode(r[0], "no_flock") {
		t.Errorf("no_flock not raised; got %v", findingCodes(r[0]))
	}

	lines = parseString(t, "*/5 * * * * flock -n /tmp/x.lock /usr/local/bin/x.sh > /dev/null 2>&1\n", false)
	r = Run(lines, Config{Now: fixedNow, MailTo: "ops@example.com"})
	if hasCode(r[0], "no_flock") {
		t.Errorf("no_flock should be silenced when flock is present")
	}
}

func TestCheckDriftSmell(t *testing.T) {
	src := "0 * * * * /usr/local/bin/a.sh > /dev/null 2>&1\n0 * * * * /usr/local/bin/b.sh > /dev/null 2>&1\n"
	lines := parseString(t, src, false)
	r := Run(lines, Config{Now: fixedNow, MailTo: "ops@example.com"})
	if !hasCode(r[0], "drift_smell") {
		t.Errorf("drift_smell not raised on line 0; got %v", findingCodes(r[0]))
	}
}

func TestCheckWorldWritableAndNotExecutable(t *testing.T) {
	dir := t.TempDir()
	wwPath := filepath.Join(dir, "ww.sh")
	if err := os.WriteFile(wwPath, []byte("#!/bin/sh\n"), 0o777); err != nil {
		t.Fatalf("write ww.sh: %v", err)
	}
	// os.WriteFile honors umask (typically 0o022), which strips the world-write
	// bit. Re-apply the mode explicitly so the world-writable check has
	// something to find.
	if err := os.Chmod(wwPath, 0o777); err != nil {
		t.Fatalf("chmod ww.sh: %v", err)
	}
	nePath := filepath.Join(dir, "ne.sh")
	if err := os.WriteFile(nePath, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write ne.sh: %v", err)
	}
	if err := os.Chmod(nePath, 0o644); err != nil {
		t.Fatalf("chmod ne.sh: %v", err)
	}

	src := "0 5 * * * " + filepath.ToSlash(wwPath) + " > /dev/null 2>&1\n" +
		"0 6 * * * " + filepath.ToSlash(nePath) + " > /dev/null 2>&1\n"
	lines := parseString(t, src, false)
	r := Run(lines, Config{Now: fixedNow, MailTo: "ops@example.com"})

	// Windows file ACLs don't expose POSIX 0o002, so skip the world-writable
	// assertion there. The "not executable" path should still trigger on POSIX.
	if isPosix() {
		if !hasCode(r[0], "world_writable") {
			t.Errorf("world_writable not raised on line 0; got %v", findingCodes(r[0]))
		}
		if !hasCode(r[1], "not_executable") {
			t.Errorf("not_executable not raised on line 1; got %v", findingCodes(r[1]))
		}
	}
}

func TestParseErrorSurface(t *testing.T) {
	lines := parseString(t, "*/15 25 * * * /bin/echo hi\n", false)
	r := Run(lines, Config{Now: fixedNow})
	if !hasCode(r[0], "parse_error") {
		t.Errorf("parse_error not raised; got %v", findingCodes(r[0]))
	}
}
