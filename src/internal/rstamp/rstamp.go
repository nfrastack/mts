// SPDX-FileCopyrightText: © 2026 Nfrastack <code@nfrastack.com>
//
// SPDX-License-Identifier: BSD-3-Clause

package rstamp

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var expr = regexp.MustCompile(`(?i)\b(?:` +
	`\d{3,4}[-:]\d\d[-:]\d\dT\d\d:\d\d:\d\d(?:\.\d+)?(?:Z|[+-]\d\d:?\d\d)?` +
	`|[A-Za-z]{3}\s+\d{1,2}\s+\d\d:\d\d:\d\d` +
	`|[A-Za-z]{3},?\s+\d{1,2}\s+[A-Za-z]{3}\s+\d{2,4}\s+\d\d:\d\d:\d\d` +
	`|\d{1,2}[-/\s][A-Za-z]{3}(?:/\d{2,4})?[\s:]\d\d:\d\d(?::\d\d)?(?:\s+[+-]\d\d:?\d\d)?` +
	`|[A-Za-z]{3}\s+[A-Za-z]{3}\s+\d{1,2}\s+\d\d:\d\d` +
	`)\b`)

var alphaZone = regexp.MustCompile(` [A-Za-z]{3}$`)

var zoneLayouts = []string{
	"2006-01-02T15:04:05Z07:00",
	"2006:01:02T15:04:05Z07:00",
	"2006-01-02T15:04:05-0700",
	"2006:01:02T15:04:05-0700",
	"02 Jan 15:04:05 -0700",
	"02 Jan 15:04 -0700",
	"02 Jan 15:04:05 -07:00",
	"02 Jan 15:04 -07:00",
	"2 Jan 06 15:04:05 -0700",
	"2 Jan 06 15:04:05 -07:00",
}

var naiveLayouts = []string{
	"2006-01-02T15:04:05",
	"2006:01:02T15:04:05",
	"Jan _2 15:04:05",
	"Jan 02 15:04:05",
	"02 Jan 15:04:05",
	"02-Jan 15:04:05",
	"02/Jan 15:04:05",
	"02 Jan 15:04",
	"02-Jan 15:04",
	"02/Jan 15:04",
	"02 Jan:15:04",
	"02 Jan:15:04:05",
	"02 Jan/06 15:04:05",
	"02 Jan/06 15:04",
	"2 Jan 06 15:04:05",
	"_2 Jan 06 15:04:05",
	"Mon _2 Jan 06 15:04:05",
	"Mon 02 Jan 06 15:04:05",
	"Mon, _2 Jan 06 15:04:05",
	"Mon, 02 Jan 06 15:04:05",
	"Mon Jan _2 15:04",
	"Mon Jan 02 15:04",
}

func Parse(s string, loc *time.Location, now time.Time) (time.Time, error) {
	v := alphaZone.ReplaceAllString(s, "")
	year := now.In(loc).Year()
	for _, layout := range zoneLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			if t.Year() == 0 {
				t = time.Date(year, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
			}
			return t.In(loc), nil
		}
	}
	for _, layout := range naiveLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			y := t.Year()
			if y == 0 {
				y = year
			}
			return time.Date(y, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc), nil
		}
	}
	return time.Time{}, fmt.Errorf("rstamp: unrecognized timestamp %q", s)
}

func Ago(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	s := int64(d / time.Second)
	units := []struct {
		n int64
		u string
	}{
		{365 * 24 * 3600, "y"},
		{30 * 24 * 3600, "M"},
		{7 * 24 * 3600, "w"},
		{24 * 3600, "d"},
		{3600, "h"},
		{60, "m"},
		{1, "s"},
	}
	var b strings.Builder
	parts := 0
	for _, u := range units {
		if parts == 2 {
			break
		}
		if v := s / u.n; v > 0 {
			fmt.Fprintf(&b, "%d%s", v, u.u)
			s -= v * u.n
			parts++
		}
	}
	if parts == 0 {
		return "0s ago"
	}
	b.WriteString(" ago")
	return b.String()
}

type Rewriter struct {
	seen        bool
	first, prev time.Time
}

func (w *Rewriter) RewriteLine(loc *time.Location, now time.Time, line []byte, render func(first, prev, cur time.Time) string) []byte {
	return expr.ReplaceAllFunc(line, func(m []byte) []byte {
		t, err := Parse(string(m), loc, now)
		if err != nil {
			return m
		}
		if !w.seen {
			w.first, w.prev, w.seen = t, t, true
		}
		out := render(w.first, w.prev, t)
		w.prev = t
		return []byte(out)
	})
}
