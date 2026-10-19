// SPDX-FileCopyrightText: © 2026 Nfrastack <code@nfrastack.com>
//
// SPDX-License-Identifier: BSD-3-Clause

package rstamp

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		in   string
		want string
	}{
		{"Oct 18 10:50:01", "2026-10-18 10:50:01 +0000"},
		{"Oct  5 12:00:00", "2026-10-05 12:00:00 +0000"},
		{"21 dec 17:05", "2026-12-21 17:05:00 +0000"},
		{"21-dec 17:05:30", "2026-12-21 17:05:30 +0000"},
		{"21 dec/93 17:05", "1993-12-21 17:05:00 +0000"},
		{"16 Jun 94 07:29:35", "1994-06-16 07:29:35 +0000"},
		{"Mon 16 Jun 94 07:29:35", "1994-06-16 07:29:35 +0000"},
		{"16 Jun 94 07:29:35 EST", "1994-06-16 07:29:35 +0000"},
		{"Mon Oct 18 17:50", "2026-10-18 17:50:00 +0000"},
		{"2026-01-01T10:00:00Z", "2026-01-01 10:00:00 +0000"},
		{"2026-01-01T10:00:00.5+07:00", "2026-01-01 03:00:00.5 +0000"},
		{"2026-01-01T10:00:00+0700", "2026-01-01 03:00:00 +0000"},
		{"2026-01-01T10:00:00.25", "2026-01-01 10:00:00.25 +0000"},
		{"21 dec 17:05 +0200", "2026-12-21 15:05:00 +0000"},
	}
	for _, c := range cases {
		got, err := Parse(c.in, time.UTC, now)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", c.in, err)
			continue
		}
		if s := got.Format("2006-01-02 15:04:05.999999999 -0700"); s != c.want {
			t.Errorf("Parse(%q) = %q, want %q", c.in, s, c.want)
		}
	}
	for _, in := range []string{"", "hello", "99 xyz 99:99:99", "12:34"} {
		if _, err := Parse(in, time.UTC, now); err == nil {
			t.Errorf("Parse(%q) should fail", in)
		}
	}
}

func TestParseLocation(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	loc, err := time.LoadLocation("America/Toronto")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	got, err := Parse("Oct 18 10:50:01", loc, now)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if got.Hour() != 10 || got.Location() != loc {
		t.Errorf("got %v, want wall 10:50 in Toronto", got)
	}
}

func TestAgo(t *testing.T) {
	cases := map[int64]string{
		-5: "0s ago", 0: "0s ago", 5: "5s ago", 59: "59s ago",
		65: "1m5s ago", 3600: "1h ago", 5400: "1h30m ago",
		90000: "1d1h ago", 10 * 86400: "1w3d ago", 400 * 86400: "1y1M ago",
	}
	for secs, want := range cases {
		if got := Ago(time.Duration(secs) * time.Second); got != want {
			t.Errorf("Ago(%d) = %q, want %q", secs, got, want)
		}
	}
}

func TestRewriteLine(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	var w Rewriter
	out := w.RewriteLine(time.UTC, now, []byte("Oct 18 10:50:01 msg"), func(_, _, cur time.Time) string {
		return cur.Format("15:04")
	})
	if string(out) != "10:50 msg" {
		t.Errorf("got %q, want %q", out, "10:50 msg")
	}
	out = w.RewriteLine(time.UTC, now, []byte("nothing here"), func(_, _, cur time.Time) string {
		return "X"
	})
	if string(out) != "nothing here" {
		t.Errorf("got %q, want passthrough", out)
	}
}

func TestRewriteDeltas(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	var w Rewriter
	render := func(first, _ time.Time, cur time.Time) string {
		return cur.Sub(first).String()
	}
	l1 := w.RewriteLine(time.UTC, now, []byte("2026-01-01T10:00:00Z a"), render)
	l2 := w.RewriteLine(time.UTC, now, []byte("2026-01-01T10:01:00Z b"), render)
	if string(l1) != "0s a" || string(l2) != "1m0s b" {
		t.Errorf("got %q %q, want deltas from first stamp", l1, l2)
	}
}
