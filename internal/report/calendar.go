package report

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/HeytalePazguato/cron-doctor/internal/parser"
)

// Calendar prints a 7-day ASCII heatmap of fire times. Each cell shows the
// number of jobs that will fire in that hour. Hour-cells with two or more
// jobs are listed below the grid for collision investigation.
func Calendar(w io.Writer, lines []*parser.Line, useColor bool) {
	p := newPalette(useColor)
	now := time.Now()
	startDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := startDay.Add(7 * 24 * time.Hour)

	type cell struct {
		count int
		lines []int
	}
	grid := make([][24]cell, 7)
	for _, ln := range lines {
		if ln.Type != parser.LineJob || ln.IsReboot || ln.CronSchedule == nil {
			continue
		}
		t := startDay
		for {
			t = ln.CronSchedule.Next(t)
			if !t.Before(end) {
				break
			}
			day := int(t.Sub(startDay).Hours() / 24)
			if day < 0 || day >= 7 {
				continue
			}
			hour := t.Hour()
			c := &grid[day][hour]
			c.count++
			c.lines = appendUnique(c.lines, ln.Number)
		}
	}

	// Header row.
	fmt.Fprintf(w, "%s7-day calendar starting %s%s\n\n", p.bold, startDay.Format("2006-01-02"), p.reset)
	fmt.Fprint(w, "       ")
	for h := 0; h < 24; h++ {
		fmt.Fprintf(w, "%2d ", h)
	}
	fmt.Fprintln(w)

	for d := 0; d < 7; d++ {
		day := startDay.AddDate(0, 0, d)
		fmt.Fprintf(w, "%s  ", day.Format("Mon 02"))
		for h := 0; h < 24; h++ {
			c := grid[d][h]
			fmt.Fprintf(w, "%s ", cellRender(p, c.count))
		}
		fmt.Fprintln(w)
	}

	// Collisions list.
	type collision struct {
		when  time.Time
		lines []int
		count int
	}
	var collisions []collision
	for d := 0; d < 7; d++ {
		for h := 0; h < 24; h++ {
			c := grid[d][h]
			if c.count >= 2 {
				when := startDay.Add(time.Duration(d*24+h) * time.Hour)
				collisions = append(collisions, collision{when, c.lines, c.count})
			}
		}
	}
	if len(collisions) == 0 {
		return
	}
	sort.Slice(collisions, func(i, j int) bool { return collisions[i].when.Before(collisions[j].when) })
	fmt.Fprintf(w, "\n%sHour-cells with overlapping jobs:%s\n", p.bold, p.reset)
	for _, c := range collisions {
		nums := make([]string, len(c.lines))
		for i, n := range c.lines {
			nums[i] = fmt.Sprintf("%d", n)
		}
		fmt.Fprintf(w, "  %s — %d fires across lines %s\n",
			c.when.Format("Mon 02 15:00"), c.count, strings.Join(nums, ", "))
	}
}

func cellRender(p palette, count int) string {
	switch {
	case count == 0:
		return fmt.Sprintf("%s.%s ", p.gray, p.reset)
	case count == 1:
		return fmt.Sprintf("%s1%s ", p.blue, p.reset)
	case count < 10:
		return fmt.Sprintf("%s%d%s ", p.yellow, count, p.reset)
	default:
		return fmt.Sprintf("%s+%s ", p.red, p.reset)
	}
}

func appendUnique(s []int, v int) []int {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
