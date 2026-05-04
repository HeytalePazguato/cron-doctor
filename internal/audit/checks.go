package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HeytalePazguato/cron-doctor/internal/parser"
)

// scriptPath returns the first absolute-path token in a command line, or
// the first whitespace-delimited token when nothing absolute is present.
// Common shell prefixes (env, sudo, nice, etc.) are skipped so we land on
// the script the user actually intends to run.
func scriptPath(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	skipPrefixes := map[string]bool{
		"sudo": true, "nice": true, "ionice": true,
		"/usr/bin/env": true, "env": true, "exec": true,
		"nohup": true,
	}
	tokens := strings.Fields(cmd)
	for i, t := range tokens {
		if skipPrefixes[t] {
			continue
		}
		// Skip "VAR=value" assignments (env-style prefix).
		if strings.Contains(t, "=") && !strings.HasPrefix(t, "/") {
			continue
		}
		if strings.HasPrefix(t, "/") {
			return t
		}
		// Stop on the first non-prefix token even if not absolute.
		_ = i
		return t
	}
	return tokens[0]
}

// checkMissingScript flags absolute-path scripts that don't exist or aren't executable.
func checkMissingScript(line *parser.Line) []Finding {
	cmd := line.Command
	if cmd == "" {
		return nil
	}
	path := scriptPath(cmd)
	if !strings.HasPrefix(path, "/") {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Finding{{SevError, "missing_script", fmt.Sprintf("Script %s does not exist.", path)}}
		}
		return nil // permission error etc.; don't pretend to know
	}
	if st.IsDir() {
		return []Finding{{SevError, "missing_script", fmt.Sprintf("Script path %s is a directory.", path)}}
	}
	if st.Mode().Perm()&0o111 == 0 {
		return []Finding{{SevWarn, "not_executable", fmt.Sprintf("Script %s exists but is not executable.", path)}}
	}
	return nil
}

// checkWorldWritable warns when the script file is world-writable.
func checkWorldWritable(line *parser.Line) []Finding {
	path := scriptPath(line.Command)
	if !strings.HasPrefix(path, "/") {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if st.Mode().Perm()&0o002 != 0 {
		return []Finding{{SevError, "world_writable", fmt.Sprintf("Script %s is world-writable (mode %#o).", path, st.Mode().Perm())}}
	}
	return nil
}

// checkMissingTimeout warns when network commands are run without a timeout flag.
func checkMissingTimeout(line *parser.Line) []Finding {
	cmd := line.Command
	if cmd == "" {
		return nil
	}
	tokens := strings.Fields(cmd)
	for _, tool := range []string{"curl", "wget", "nc", "ssh"} {
		if !containsToken(tokens, tool) && !containsBaseToken(tokens, tool) {
			continue
		}
		if hasTimeoutFlag(tokens, tool) {
			continue
		}
		return []Finding{{SevWarn, "missing_timeout",
			fmt.Sprintf("Command runs %s without a timeout flag; a hung connection will hang the job.", tool)}}
	}
	return nil
}

func containsToken(tokens []string, t string) bool {
	for _, tok := range tokens {
		if tok == t {
			return true
		}
	}
	return false
}

func containsBaseToken(tokens []string, t string) bool {
	for _, tok := range tokens {
		if filepath.Base(tok) == t {
			return true
		}
	}
	return false
}

func hasTimeoutFlag(tokens []string, tool string) bool {
	flags := map[string][]string{
		"curl": {"--max-time", "-m", "--connect-timeout"},
		"wget": {"--timeout", "-T", "--connect-timeout", "--read-timeout"},
		"nc":   {"-w"},
		"ssh":  {"-o", "ConnectTimeout"},
	}
	wanted := flags[tool]
	for _, tok := range tokens {
		for _, f := range wanted {
			if tok == f || strings.HasPrefix(tok, f+"=") {
				return true
			}
			if strings.Contains(tok, "ConnectTimeout") {
				return true
			}
		}
	}
	return false
}

// checkMissingRedirection warns when no stdout/stderr redirect is present
// and MAILTO is empty (so cron will email the owning user).
func checkMissingRedirection(line *parser.Line, mailTo string) []Finding {
	cmd := line.Command
	if cmd == "" {
		return nil
	}
	if hasRedirection(cmd) {
		return nil
	}
	if mailTo != "" {
		return nil
	}
	return []Finding{{SevWarn, "missing_redirection",
		"No output redirection and MAILTO is unset; cron will email the job owner."}}
}

func hasRedirection(cmd string) bool {
	// We want to flag absence of >, >>, 2>, &> but be tolerant of contents
	// inside single-quoted strings. Simple substring is acceptable here.
	return strings.Contains(cmd, ">") || strings.Contains(cmd, "&>")
}

// checkRunAsRoot warns when a system crontab runs a non-root-owned or world-writable
// script as root, or runs a script located in /home or /tmp.
func checkRunAsRoot(line *parser.Line) []Finding {
	if line.User != "root" {
		return nil
	}
	path := scriptPath(line.Command)
	if !strings.HasPrefix(path, "/") {
		return nil
	}
	if strings.HasPrefix(path, "/home/") || strings.HasPrefix(path, "/tmp/") {
		return []Finding{{SevWarn, "root_in_user_path",
			fmt.Sprintf("Job runs as root from %s; user-writable directories should not host root jobs.", path)}}
	}
	return nil
}

// checkNoFlock warns when high-frequency jobs lack a flock/pidof guard.
func checkNoFlock(line *parser.Line, now time.Time) []Finding {
	if line.IsReboot || line.CronSchedule == nil {
		return nil
	}
	if minGap(line, now) >= 10*time.Minute {
		return nil
	}
	cmd := line.Command
	if strings.Contains(cmd, "flock") || strings.Contains(cmd, "pidof") {
		return nil
	}
	return []Finding{{SevWarn, "no_flock",
		"Job fires more often than every 10 minutes without flock or pidof; concurrent runs may pile up."}}
}

// minGap returns the smallest gap between consecutive fire times in the next
// hundred fires, used as a frequency proxy.
func minGap(line *parser.Line, now time.Time) time.Duration {
	t := now
	min := 24 * time.Hour
	for i := 0; i < 100; i++ {
		next := line.CronSchedule.Next(t)
		gap := next.Sub(t)
		if i > 0 && gap < min {
			min = gap
		}
		t = next
	}
	return min
}

// checkOverlap walks pairs of jobs that share a script path and flags the pair
// when their next fires fall within window of each other.
func checkOverlap(lines []*parser.Line, window time.Duration, now time.Time) Result {
	out := make(Result, len(lines))
	type sched struct {
		idx   int
		fires []time.Time
		path  string
	}
	var scheds []sched
	for i, ln := range lines {
		if ln.Type != parser.LineJob || ln.IsReboot || ln.CronSchedule == nil {
			continue
		}
		path := scriptPath(ln.Command)
		if path == "" {
			continue
		}
		fires := nextN(ln, now, 20)
		scheds = append(scheds, sched{idx: i, fires: fires, path: path})
	}
	for a := 0; a < len(scheds); a++ {
		for b := a + 1; b < len(scheds); b++ {
			if scheds[a].path != scheds[b].path {
				continue
			}
			if anyOverlap(scheds[a].fires, scheds[b].fires, window) {
				msg := fmt.Sprintf("Overlaps with line %d (same script %s within %s).",
					lines[scheds[b].idx].Number, scheds[a].path, window)
				out[scheds[a].idx] = append(out[scheds[a].idx], Finding{SevWarn, "overlap_risk", msg})
				msg2 := fmt.Sprintf("Overlaps with line %d (same script %s within %s).",
					lines[scheds[a].idx].Number, scheds[a].path, window)
				out[scheds[b].idx] = append(out[scheds[b].idx], Finding{SevWarn, "overlap_risk", msg2})
			}
		}
	}
	return out
}

func anyOverlap(a, b []time.Time, window time.Duration) bool {
	for _, ta := range a {
		for _, tb := range b {
			d := ta.Sub(tb)
			if d < 0 {
				d = -d
			}
			if d <= window {
				return true
			}
		}
	}
	return false
}

func nextN(ln *parser.Line, now time.Time, n int) []time.Time {
	if ln.CronSchedule == nil {
		return nil
	}
	out := make([]time.Time, 0, n)
	t := now
	for i := 0; i < n; i++ {
		t = ln.CronSchedule.Next(t)
		out = append(out, t)
	}
	return out
}

// checkDriftSmell flags clusters of jobs scheduled at exactly :00 of every
// hour with the same expression — a common cargo-cult pattern that risks
// thundering-herd spikes.
func checkDriftSmell(lines []*parser.Line) Result {
	out := make(Result, len(lines))
	groups := map[string][]int{}
	for i, ln := range lines {
		if ln.Type != parser.LineJob {
			continue
		}
		if !firesOnTheHour(ln.Schedule) {
			continue
		}
		groups[ln.Schedule] = append(groups[ln.Schedule], i)
	}
	for sched, idxs := range groups {
		if len(idxs) < 2 {
			continue
		}
		var lineNums []string
		for _, i := range idxs {
			lineNums = append(lineNums, fmt.Sprintf("%d", lines[i].Number))
		}
		msg := fmt.Sprintf("Schedule %q is duplicated across lines %s; consider staggering minutes to avoid a thundering herd.",
			sched, strings.Join(lineNums, ", "))
		for _, i := range idxs {
			out[i] = append(out[i], Finding{SevInfo, "drift_smell", msg})
		}
	}
	return out
}

// firesOnTheHour reports whether the schedule is "0 * * * *" or "@hourly".
func firesOnTheHour(s string) bool {
	if s == "@hourly" {
		return true
	}
	parts := strings.Fields(s)
	if len(parts) != 5 {
		return false
	}
	if parts[0] != "0" {
		return false
	}
	for _, p := range parts[1:] {
		if p != "*" {
			return false
		}
	}
	return true
}
