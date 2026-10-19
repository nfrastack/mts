// SPDX-FileCopyrightText: © 2026 Nfrastack <code@nfrastack.com>
//
// SPDX-License-Identifier: BSD-3-Clause

package strftime

import (
	"fmt"
	"testing"
	"time"
)

func TestFormatBasic(t *testing.T) {
	loc := time.UTC
	tm := time.Date(2026, 10, 17, 14, 30, 45, 0, loc)
	cases := map[string]string{
		"%Y-%m-%d":       "2026-10-17",
		"%H:%M:%S":       "14:30:45",
		"%T":             "14:30:45",
		"%b %d %H:%M:%S": "Oct 17 14:30:45",
		"%F":             "2026-10-17",
		"%D":             "10/17/26",
		"%R":             "14:30",
		"%%":             "%",
		"%s":             fmt.Sprintf("%d", tm.Unix()),
		"%z":             "+0000",
		"%n":             "\n",
		"%t":             "\t",
		"%Q":             "%Q", // passthrough
	}
	for in, want := range cases {
		if got := Format(tm, in); got != want {
			t.Errorf("Format(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatFractional(t *testing.T) {
	tm := time.Date(2026, 10, 17, 1, 15, 30, 10_000, time.UTC) // 10 microseconds
	if got := Format(tm, "%.S"); got != "30.00001" {
		t.Errorf("%%.S = %q, want %q", got, "30.00001")
	}
	if got := Format(tm, "%.T"); got != "01:15:30.00001" {
		t.Errorf("%%.T = %q, want 01:15:30.00001", got)
	}
	if got, want := Format(tm, "%.s"), fmt.Sprintf("%d.00001", tm.Unix()); got != want {
		t.Errorf("%%.s = %q, want %q", got, want)
	}
	// zero nanos, one digit.
	tm0 := time.Date(2026, 10, 17, 1, 15, 30, 0, time.UTC)
	if got := Format(tm0, "%.S"); got != "30.0" {
		t.Errorf("%%.S zero = %q, want 30.0", got)
	}
}

func TestFormatDuration(t *testing.T) {
	d := 3723*time.Second + 250*time.Millisecond
	cases := map[string]string{
		"%H:%M:%S": "01:02:03",
		"%T":       "01:02:03",
		"%s":       "3723",
		"%.S":      "03.25",
		"%.T":      "01:02:03.25",
	}
	for in, want := range cases {
		if got := FormatDuration(d, in); got != want {
			t.Errorf("FormatDuration(%q) = %q, want %q", in, got, want)
		}
	}
	if got := FormatDuration(90*1000*1000*1000, "%.s"); got == "" {
		t.Error("expected non-empty percent-dot-s")
	}
	// negatives clamp.
	if got := FormatDuration(-5*time.Second, "%H:%M:%S"); got != "00:00:00" {
		t.Errorf("negative = %q, want 00:00:00", got)
	}
}

func TestFormatWeekNumbers(t *testing.T) {
	cases := []struct {
		date    time.Time
		weekSun string // %U
		weekMon string // %W
	}{
		{time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), "01", "00"},   // Sun
		{time.Date(2023, 1, 7, 12, 0, 0, 0, time.UTC), "01", "01"},   // Sat
		{time.Date(2023, 1, 8, 12, 0, 0, 0, time.UTC), "02", "01"},   // Sun
		{time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC), "00", "00"},   // Sat
		{time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), "00", "01"},   // Mon
		{time.Date(2023, 12, 31, 12, 0, 0, 0, time.UTC), "53", "52"}, // Sun
	}
	for _, c := range cases {
		if got := Format(c.date, "%U"); got != c.weekSun {
			t.Errorf("Format(%s, %%U) = %q, want %q", c.date.Format("2006-01-02"), got, c.weekSun)
		}
		if got := Format(c.date, "%W"); got != c.weekMon {
			t.Errorf("Format(%s, %%W) = %q, want %q", c.date.Format("2006-01-02"), got, c.weekMon)
		}
	}
}

func TestFormatWeekNumbersDST(t *testing.T) {
	// Spring forward week must match calendar days not elapsed hours
	loc, err := time.LoadLocation("America/Toronto")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	// 2026-03-08 is the DST transition (Sunday) Mar 9 is Monday
	mar9 := time.Date(2026, 3, 9, 12, 0, 0, 0, loc)
	if got, want := Format(mar9, "%U"), "10"; got != want {
		t.Errorf("%%U after spring forward = %q, want %q", got, want)
	}
	if got, want := Format(mar9, "%W"), "10"; got != want {
		t.Errorf("%%W after spring forward = %q, want %q", got, want)
	}
}

func BenchmarkFormatDefault(b *testing.B) {
	tm := time.Date(2026, 10, 17, 14, 30, 45, 0, time.UTC)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if got := Format(tm, "%b %d %H:%M:%S"); len(got) != 15 {
			b.Fatalf("unexpected length %d", len(got))
		}
	}
}

func BenchmarkFormatFractional(b *testing.B) {
	tm := time.Date(2026, 10, 17, 14, 30, 45, 123456789, time.UTC)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if got := Format(tm, "%.T"); len(got) == 0 {
			b.Fatal("empty")
		}
	}
}

func BenchmarkFormatDuration(b *testing.B) {
	d := 3723*time.Second + 250*time.Millisecond
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if got := FormatDuration(d, "%H:%M:%S"); len(got) != 8 {
			b.Fatalf("unexpected %q", got)
		}
	}
}
