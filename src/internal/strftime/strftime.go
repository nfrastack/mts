// SPDX-FileCopyrightText: © 2026 Nfrastack <code@nfrastack.com>
//
// SPDX-License-Identifier: BSD-3-Clause

// strftime formatting + moreutils %.S/%.s/%.T subsecond extensions.
package strftime

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var shortWeekdays = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
var longWeekdays = [7]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
var shortMonths = [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
var longMonths = [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}

// zero-padded 2-digit field; out-of-range (durations past 99h,
// far-future %C) falls back to fmt so output never changes.
func append2(b *strings.Builder, n int) {
	if n < 0 || n > 99 {
		fmt.Fprintf(b, "%02d", n)
		return
	}
	b.WriteByte(byte('0' + n/10))
	b.WriteByte(byte('0' + n%10))
}

// space-padded 2-digit variant (%e %k %l).
func append2space(b *strings.Builder, n int) {
	if n < 0 || n > 99 {
		fmt.Fprintf(b, "%2d", n)
		return
	}
	if n < 10 {
		b.WriteByte(' ')
	} else {
		b.WriteByte(byte('0' + n/10))
	}
	b.WriteByte(byte('0' + n%10))
}

// zero-padded 3-digit day-of-year (%j).
func append3(b *strings.Builder, n int) {
	if n < 0 || n > 999 {
		fmt.Fprintf(b, "%03d", n)
		return
	}
	b.WriteByte(byte('0' + n/100))
	b.WriteByte(byte('0' + (n/10)%10))
	b.WriteByte(byte('0' + n%10))
}

// wall-clock time in strftime style.
//
// conversions: %% %Y %m %d %e %H %k %I %l %M %S %T %p %P
// %a %A %b %B %y %C %j %U %W %D %F %R %s %z %Z %n %t
// + moreutils %.S %.s %.T.
//
// unknown left verbatim ("%Q" stays "%Q").
func Format(t time.Time, format string) string {
	var b strings.Builder
	b.Grow(len(format) + 32)
	// Fractions cost formatted conversions; skip them unless used.
	var frac, epochFrac string
	if strings.Contains(format, "%.") {
		frac = fracString(t.Nanosecond())
		epochFrac = epochFracString(t)
	}
	hour, minute, second := t.Hour(), t.Minute(), t.Second()
	i := 0
	for i < len(format) {
		c := format[i]
		if c != '%' {
			b.WriteByte(c)
			i++
			continue
		}
		i++
		if i >= len(format) {
			b.WriteByte('%')
			break
		}
		// %.S/%.s/%.T extensions
		if format[i] == '.' {
			if i+1 < len(format) {
				switch format[i+1] {
				case 'S':
					append2(&b, second)
					b.WriteString(frac)
					i += 2
					continue
				case 's':
					b.WriteString(epochFrac)
					i += 2
					continue
				case 'T':
					append2(&b, hour)
					b.WriteByte(':')
					append2(&b, minute)
					b.WriteByte(':')
					append2(&b, second)
					b.WriteString(frac)
					i += 2
					continue
				}
			}
			b.WriteByte('%')
			b.WriteByte('.')
			i++
			continue
		}
		switch format[i] {
		case '%':
			b.WriteByte('%')
		case 'Y':
			fmt.Fprintf(&b, "%04d", t.Year())
		case 'm':
			append2(&b, int(t.Month()))
		case 'd':
			append2(&b, t.Day())
		case 'e':
			append2space(&b, t.Day())
		case 'H':
			append2(&b, hour)
		case 'k':
			append2space(&b, hour)
		case 'I':
			h := hour % 12
			if h == 0 {
				h = 12
			}
			append2(&b, h)
		case 'l':
			h := hour % 12
			if h == 0 {
				h = 12
			}
			append2space(&b, h)
		case 'M':
			append2(&b, minute)
		case 'S':
			append2(&b, second)
		case 'T':
			append2(&b, hour)
			b.WriteByte(':')
			append2(&b, minute)
			b.WriteByte(':')
			append2(&b, second)
		case 'p':
			if hour < 12 {
				b.WriteString("AM")
			} else {
				b.WriteString("PM")
			}
		case 'P':
			if hour < 12 {
				b.WriteString("am")
			} else {
				b.WriteString("pm")
			}
		case 'a':
			b.WriteString(shortWeekdays[t.Weekday()])
		case 'A':
			b.WriteString(longWeekdays[t.Weekday()])
		case 'b':
			b.WriteString(shortMonths[int(t.Month())-1])
		case 'B':
			b.WriteString(longMonths[int(t.Month())-1])
		case 'y':
			append2(&b, t.Year()%100)
		case 'C':
			append2(&b, t.Year()/100)
		case 'j':
			append3(&b, t.YearDay())
		case 'U':
			append2(&b, weekSunday(t))
		case 'W':
			append2(&b, weekMonday(t))
		case 'D':
			append2(&b, int(t.Month()))
			b.WriteByte('/')
			append2(&b, t.Day())
			b.WriteByte('/')
			append2(&b, t.Year()%100)
		case 'F':
			fmt.Fprintf(&b, "%04d", t.Year())
			b.WriteByte('-')
			append2(&b, int(t.Month()))
			b.WriteByte('-')
			append2(&b, t.Day())
		case 'R':
			append2(&b, hour)
			b.WriteByte(':')
			append2(&b, minute)
		case 's':
			b.WriteString(strconv.FormatInt(t.Unix(), 10))
		case 'z':
			b.WriteString(t.Format("-0700"))
		case 'Z':
			b.WriteString(t.Format("PDT"))
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		default:
			b.WriteByte('%')
			b.WriteByte(format[i])
		}
		i++
	}
	return b.String()
}

// %U, Sunday first, 00-53.
func weekSunday(t time.Time) int {
	yday0 := t.YearDay() - 1
	return (yday0 - int(t.Weekday()) + 7) / 7
}

// %W, Monday first, 00-53.
func weekMonday(t time.Time) int {
	yday0 := t.YearDay() - 1
	mon0 := (int(t.Weekday()) + 6) % 7 // Mon=0..Sun=6
	return (yday0 - mon0 + 7) / 7
}

// -i/-s elapsed. %H runs past 24, %M/%S parts, %s/%.s whole/fractional totals
func FormatDuration(d time.Duration, format string) string {
	if d < 0 {
		d = 0
	}
	totalSecs := int64(d / time.Second)
	ns := int(d % time.Second / time.Nanosecond)
	hours := totalSecs / 3600
	mins := (totalSecs % 3600) / 60
	secs := totalSecs % 60
	var frac string
	if strings.Contains(format, "%.") {
		frac = fracString(ns)
	}

	var b strings.Builder
	b.Grow(len(format) + 16)
	i := 0
	for i < len(format) {
		c := format[i]
		if c != '%' {
			b.WriteByte(c)
			i++
			continue
		}
		i++
		if i >= len(format) {
			b.WriteByte('%')
			break
		}
		if format[i] == '.' {
			if i+1 < len(format) {
				switch format[i+1] {
				case 'S':
					append2(&b, int(secs))
					b.WriteString(frac)
					i += 2
					continue
				case 's':
					b.WriteString(strconv.FormatInt(totalSecs, 10))
					b.WriteString(frac)
					i += 2
					continue
				case 'T':
					append2(&b, int(hours))
					b.WriteByte(':')
					append2(&b, int(mins))
					b.WriteByte(':')
					append2(&b, int(secs))
					b.WriteString(frac)
					i += 2
					continue
				}
			}
			b.WriteByte('%')
			b.WriteByte('.')
			i++
			continue
		}
		switch format[i] {
		case '%':
			b.WriteByte('%')
		case 'H':
			append2(&b, int(hours))
		case 'M':
			append2(&b, int(mins))
		case 'S':
			append2(&b, int(secs))
		case 's':
			b.WriteString(strconv.FormatInt(totalSecs, 10))
		case 'T':
			append2(&b, int(hours))
			b.WriteByte(':')
			append2(&b, int(mins))
			b.WriteByte(':')
			append2(&b, int(secs))
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		default:
			b.WriteByte('%')
			b.WriteByte(format[i])
		}
		i++
	}
	return b.String()
}

// .nnn - trailing zeros trimmed, .0 min
func fracString(ns int) string {
	if ns == 0 {
		return ".0"
	}
	var buf [10]byte // '.' + 9 digits
	buf[0] = '.'
	n := ns
	for i := 9; i >= 1; i-- {
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	end := 10
	for end > 2 && buf[end-1] == '0' {
		end--
	}
	return string(buf[:end])
}

func epochFracString(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10) + fracString(t.Nanosecond())
}
