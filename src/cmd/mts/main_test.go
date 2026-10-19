// SPDX-FileCopyrightText: © 2026 Nfrastack <code@nfrastack.com>
//
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"os/user"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestResolveLocationFlag(t *testing.T) {
	t.Setenv("TZ", "")
	t.Setenv("TIMEZONE", "")
	loc, err := resolveLocation(options{tz: "America/Toronto"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 12:00 UTC == 07:00 EST.
	ref := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC).In(loc)
	if got := ref.Hour(); got != 7 {
		t.Errorf("hour in America/Toronto = %d, want 7", got)
	}
}

func TestResolveLocationBadZone(t *testing.T) {
	t.Setenv("TZ", "")
	t.Setenv("TIMEZONE", "")
	if _, err := resolveLocation(options{tz: "Mars/Olympus"}); err == nil {
		t.Error("expected error for unknown zone, got nil")
	}
}

func TestResolveLocationEnv(t *testing.T) {
	t.Setenv("TIMEZONE", "")
	t.Setenv("TZ", "Asia/Bangkok")
	loc, err := resolveLocation(options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ref := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC).In(loc)
	if got := ref.Hour(); got != 19 {
		t.Errorf("hour in Asia/Bangkok = %d, want 19", got)
	}
}

func TestResolveLocationFlagBeatsEnv(t *testing.T) {
	t.Setenv("TIMEZONE", "")
	t.Setenv("TZ", "Asia/Bangkok")
	loc, err := resolveLocation(options{tz: "UTC"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc != time.UTC {
		t.Errorf("expected UTC, got %v", loc)
	}
}

func TestResolveLocationUTC(t *testing.T) {
	t.Setenv("TZ", "")
	t.Setenv("TIMEZONE", "")
	loc, err := resolveLocation(options{utc: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc != time.UTC {
		t.Errorf("expected UTC, got %v", loc)
	}
}

func TestResolveLocationLegacyTimezone(t *testing.T) {
	t.Setenv("TZ", "")
	t.Setenv("TIMEZONE", "Asia/Bangkok")
	loc, err := resolveLocation(options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ref := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC).In(loc)
	if got := ref.Hour(); got != 19 {
		t.Errorf("hour in Asia/Bangkok = %d, want 19", got)
	}
}

func TestParseFileMode(t *testing.T) {
	m, set, err := parseFileMode("")
	if err != nil || set || m != 0 {
		t.Errorf("empty = %v %v %v, want 0 false nil", m, set, err)
	}
	m, set, err = parseFileMode("640")
	if err != nil || !set || m != 0o640 {
		t.Errorf("640 = %v %v %v, want 640 true nil", m, set, err)
	}
	if _, _, err = parseFileMode("888"); err == nil {
		t.Error("888 should fail (not octal)")
	}
	if _, _, err = parseFileMode("10000"); err == nil {
		t.Error("10000 should fail (past 7777)")
	}
}

func TestResolveRunUserNumeric(t *testing.T) {
	id, err := resolveRunUser("1000:100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.uid != 1000 || id.gid != 100 {
		t.Errorf("got %d:%d, want 1000:100", id.uid, id.gid)
	}
	// Explicit numerics never consult passwd: deterministic everywhere.
	id, err = resolveRunUser("0:0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.uid != 0 || id.gid != 0 {
		t.Errorf("got %d:%d, want 0:0", id.uid, id.gid)
	}
	// Bare numerics resolve the primary group; covered by
	// TestResolveRunUserNumericPrimaryGroup (environment-dependent).
}

func TestResolveRunUserNumericPrimaryGroup(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Skip("no current user")
	}
	wantGid, err := strconv.Atoi(u.Gid)
	if err != nil {
		t.Skip("non-numeric primary gid")
	}
	id, err := resolveRunUser(u.Uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.gid != wantGid {
		t.Errorf("bare numeric uid gid = %d, want primary %d", id.gid, wantGid)
	}
	// explicit group still wins.
	id, err = resolveRunUser(u.Uid + ":" + u.Gid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.gid != wantGid {
		t.Errorf("explicit gid = %d, want %d", id.gid, wantGid)
	}
}

func TestResolveRunUserBad(t *testing.T) {
	for _, spec := range []string{":100", "nosuchuser_xyz", "0:nosuchgroup_xyz"} {
		if _, err := resolveRunUser(spec); err == nil {
			t.Errorf("%q should fail", spec)
		}
	}
}

func TestResolveOutputUserShorthand(t *testing.T) {
	id, err := resolveOutputUser("1000:100", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.uid != 1000 || id.gid != 100 {
		t.Errorf("got %d:%d, want 1000:100", id.uid, id.gid)
	}
	id, err = resolveOutputUser("1000", "200")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.uid != 1000 || id.gid != 200 {
		t.Errorf("got %d:%d, want 1000:200", id.uid, id.gid)
	}
	id, err = resolveOutputUser("", "100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.uid != -1 || id.gid != 100 {
		t.Errorf("got %d:%d, want -1:100", id.uid, id.gid)
	}
	if _, err := resolveOutputUser("1000:100", "200"); err == nil {
		t.Error("colon + --output-group should fail")
	}
}

func runFilterString(t *testing.T, opts options, in string) string {
	t.Helper()
	var out bytes.Buffer
	if rc := runFilter(opts, time.UTC, 0, false, nil, strings.NewReader(in), &out); rc != 0 {
		t.Fatalf("runFilter exit = %d", rc)
	}
	return out.String()
}

func TestRunFilterRelativeAgo(t *testing.T) {
	got := runFilterString(t, options{relative: true, format: "%H:%M:%S"},
		"Jan  5 12:00:00 old line\nno stamp here\n")
	if !strings.Contains(got, " ago old line\n") {
		t.Errorf("got %q, want ago rewrite", got)
	}
	if !strings.HasSuffix(got, "no stamp here\n") {
		t.Errorf("got %q, want passthrough", got)
	}
}

func TestRunFilterRelativeFormat(t *testing.T) {
	got := runFilterString(t, options{relative: true, format: "%H:%M", formatSet: true},
		"Oct 18 10:50:01 msg\n")
	if got != "10:50 msg\n" {
		t.Errorf("got %q, want %q", got, "10:50 msg\n")
	}
}

func TestRunFilterRelativeSince(t *testing.T) {
	in := "2026-01-01T10:00:00Z a\n2026-01-01T10:01:30Z b\n"
	got := runFilterString(t, options{relative: true, sinceStart: true, format: "%H:%M:%S"}, in)
	if got != "00:00:00 a\n00:01:30 b\n" {
		t.Errorf("got %q", got)
	}
}

func TestRunFilterRelativeIncremental(t *testing.T) {
	in := "2026-01-01T10:00:00Z a\n2026-01-01T10:01:30Z b\n"
	got := runFilterString(t, options{relative: true, incremental: true, format: "%H:%M:%S"}, in)
	if got != "00:00:00 a\n00:01:30 b\n" {
		t.Errorf("got %q", got)
	}
}

func TestRunFilterRelativeISO(t *testing.T) {
	got := runFilterString(t, options{relative: true, iso8601: true},
		"Oct 18 10:50:01 msg\n")
	if !strings.Contains(got, "T10:50:01") || !strings.HasSuffix(got, " msg\n") {
		t.Errorf("got %q", got)
	}
}
