// Copyright (c) 2026 allddd <me@allddd.onl>
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Package systemdtime parses time values based on the systemd time specification.
// For full details, see the systemd.time(7) man page or visit:
// https://www.freedesktop.org/software/systemd/man/latest/systemd.time.html
package systemdtime

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	Nanosecond  = time.Nanosecond
	Microsecond = time.Microsecond
	Millisecond = time.Millisecond
	Second      = time.Second
	Minute      = time.Minute
	Hour        = time.Hour
	Day         = 24 * Hour
	Week        = 7 * Day
	Month       = Year / 12                            // 30.4375 days
	Year        = time.Duration(365.25 * float64(Day)) // 365.25 days
)

// readFrac reads a number from s starting at position pos and returns the number
// (as nanoseconds), the position after the number, and any error.
func readFrac(s string, pos int) (int, int, error) {
	i := pos
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == pos {
		return 0, pos, fmt.Errorf("expected number in %q", s)
	}
	frac := s[pos:i]
	if len(frac) > 9 { // 9 digits (nanosecond precision)
		frac = frac[:9]
	}
	n, err := strconv.Atoi(frac)
	if err != nil {
		return 0, pos, fmt.Errorf("expected number, got %q in %q: %w", frac, s, err)
	}
	for j := len(frac); j < 9; j++ { // pad to nanosecond precision
		n *= 10
	}
	return n, i, nil
}

// readNum reads a number from s starting at position pos and returns the number,
// the position after the number, and any error.
func readNum(s string, pos int) (int, int, error) {
	i := pos
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == pos {
		return 0, pos, fmt.Errorf("expected number in %q", s)
	}
	n, err := strconv.Atoi(s[pos:i])
	if err != nil {
		return 0, pos, fmt.Errorf("expected number, got %q in %q: %w", s[pos:i], s, err)
	}
	return n, i, nil
}

// readWord reads all non-digit, non-space characters from s starting at position
// pos and returns the string and the position after it.
func readWord(s string, pos int) (string, int) {
	i := pos
	for i < len(s) && s[i] != ' ' && (s[i] < '0' || s[i] > '9') {
		i++
	}
	return s[pos:i], i
}

// handleDate parses a date from s starting at position pos and returns the year,
// month, day, position after the date, whether the year is full 4-digit, and any
// error. Dates must be in YYYY-MM-DD or YY-MM-DD format.
func handleDate(s string, pos int) (int, int, int, int, bool, error) {
	if pos >= len(s) {
		return 0, 0, 0, pos, false, fmt.Errorf("expected date (YYYY-MM-DD or YY-MM-DD), got %q", s)
	}

	// parse year
	year, i, err := readNum(s, pos)
	if err != nil {
		return 0, 0, 0, pos, false, err
	}
	fullYear := year >= 100 // 100 is threshold for 2-digit year
	if !fullYear {
		// 0-68 is 2000-2068, 69-99 is 1969-1999
		// systemd does the same thing but rejects 69 and 70 for whatever reason
		if year <= 68 {
			year += 2000
		} else {
			year += 1900
		}
	}

	if i >= len(s) || s[i] != '-' {
		return 0, 0, 0, pos, false, fmt.Errorf("expected date (YYYY-MM-DD or YY-MM-DD), got %q", s)
	}
	i++

	// parse month
	month, i, err := readNum(s, i)
	if err != nil {
		return 0, 0, 0, pos, false, err
	}
	if month < 1 || month > 12 {
		return 0, 0, 0, pos, false, fmt.Errorf("expected month in range 1-12, got %d in %q", month, s)
	}

	if i >= len(s) || s[i] != '-' {
		return 0, 0, 0, pos, false, fmt.Errorf("expected date (YYYY-MM-DD or YY-MM-DD), got %q", s)
	}
	i++

	// parse day
	day, i, err := readNum(s, i)
	if err != nil {
		return 0, 0, 0, pos, false, err
	}
	if day < 1 || day > 31 {
		return 0, 0, 0, pos, false, fmt.Errorf("expected day in range 1-31, got %d in %q", day, s)
	}

	return year, month, day, i, fullYear, nil
}

// handleToken parses special tokens ("today", "yesterday", or "tomorrow") with
// optional timezone and returns the parsed time, whether a token was found, and
// any error. Tokens are case-sensitive (must be lowercase) and refer to 00:00:00
// of the respective day.
func handleToken(s string, now time.Time) (time.Time, bool, error) {
	var tokenLen, offset int

	switch {
	case len(s) >= 5 && s[:5] == "today":
		tokenLen = 5
	case len(s) >= 9 && s[:9] == "yesterday":
		tokenLen = 9
		offset = -1
	case len(s) >= 8 && s[:8] == "tomorrow":
		tokenLen = 8
		offset = 1
	default:
		return time.Time{}, false, nil
	}

	loc := now.Location()

	// parse (optional) timezone after token
	if i := tokenLen; i < len(s) {
		for i < len(s) && s[i] == ' ' {
			i++
		}
		if i < len(s) {
			var err error
			loc, i, err = handleTimezone(s, i)
			if err != nil {
				return time.Time{}, true, err
			}
			if i < len(s) {
				return time.Time{}, true, fmt.Errorf("expected end of input, got %q in %q", s[i:], s)
			}
		}
	}

	year, month, day := now.In(loc).Date()
	return time.Date(year, month, day+offset, 0, 0, 0, 0, loc), true, nil
}

// handleTime parses a time from s starting at position pos and returns the hour, minute,
// second, nanosecond, position after the time, and any error. Times are specified as
// HH:MM:SS or HH:MM (seconds default to 0). Fractional seconds are supported.
func handleTime(s string, pos int) (int, int, int, int, int, error) {
	if pos >= len(s) {
		return 0, 0, 0, 0, pos, fmt.Errorf("expected time (HH:MM or HH:MM:SS), got %q", s)
	}

	var minute, second, nsec int

	// parse hour
	hour, i, err := readNum(s, pos)
	if err != nil {
		return 0, 0, 0, 0, pos, err
	}
	if hour > 23 { // 23 is max valid hour
		return 0, 0, 0, 0, pos, fmt.Errorf("expected hour in range 0-23, got %d in %q", hour, s)
	}

	// parse minute
	if i < len(s) && s[i] == ':' {
		i++
		minute, i, err = readNum(s, i)
		if err != nil {
			return 0, 0, 0, 0, pos, err
		}
		if minute > 59 { // 59 is max valid minute
			return 0, 0, 0, 0, pos, fmt.Errorf("expected minute in range 0-59, got %d in %q", minute, s)
		}

		// parse second
		if i < len(s) && s[i] == ':' {
			i++
			second, i, err = readNum(s, i)
			if err != nil {
				return 0, 0, 0, 0, pos, err
			}
			if second > 59 { // 59 is max valid second
				return 0, 0, 0, 0, pos, fmt.Errorf("expected second in range 0-59, got %d in %q", second, s)
			}

			if i < len(s) && s[i] == '.' {
				i++
				nsec, i, err = readFrac(s, i)
				if err != nil {
					return 0, 0, 0, 0, pos, err
				}
			}
		}
	}

	return hour, minute, second, nsec, i, nil
}

// handleTimezone parses a timezone from s starting at position pos and returns the location,
// position after the timezone, and any error. Timezones can be "UTC", "Z", an IANA timezone
// name (e.g. "Europe/Amsterdam"), or an offset in ±HH:MM, ±HHMM, or ±HH format. Unlike
// systemd, ±HH and ±HHMM are also accepted when directly affixed to a timestamp.
func handleTimezone(s string, pos int) (*time.Location, int, error) {
	if pos >= len(s) {
		return nil, pos, fmt.Errorf("expected timezone, got %q", s)
	}

	i := pos

	// check for UTC
	switch s[i:] {
	case "Z":
		return time.UTC, i + 1, nil // 1 is length of "Z"
	case "UTC":
		return time.UTC, i + 3, nil // 3 is length of "UTC"
	}

	// check for offset format: +05:30, +0530, +05, -05:30, etc.
	if s[i] == '+' || s[i] == '-' {
		sign := 1
		if s[i] == '-' {
			sign = -1
		}
		i++
		if i >= len(s) {
			return nil, pos, fmt.Errorf("expected number after %q in %q", string(s[pos]), s)
		}

		var num int
		var err error

		numStart := i
		num, i, err = readNum(s, i)
		if err != nil {
			return nil, pos, err
		}
		digits := i - numStart

		switch digits {
		case 2: // 2 is the digit count for HH format
			hours := num
			if i < len(s) && s[i] == ':' {
				i++
				minsStart := i
				num, i, err = readNum(s, i)
				if err != nil {
					return nil, pos, err
				}
				if i-minsStart != 2 { // 2 is the required digit count for MM
					return nil, pos, fmt.Errorf("expected 2-digit offset, got %d digits in %q", i-minsStart, s)
				}
				minutes := num
				if minutes >= 60 {
					return nil, pos, fmt.Errorf("timezone offset minutes out of range (0-59), got %d in %q", minutes, s)
				}
				offsetSecs := hours*3600 + minutes*60
				if offsetSecs > 86400 { // 24h is the maximum allowed offset
					return nil, pos, fmt.Errorf("timezone offset out of range (max 24h), got %d seconds in %q", offsetSecs, s)
				}
				return time.FixedZone("", sign*offsetSecs), i, nil
			}
			if hours > 24 {
				return nil, pos, fmt.Errorf("timezone offset out of range (max 24h), got %dh in %q", hours, s)
			}
			return time.FixedZone("", sign*hours*3600), i, nil // 3600 seconds per hour
		case 4: // 4 is the digit count for HHMM format
			hours, minutes := num/100, num%100
			if minutes >= 60 {
				return nil, pos, fmt.Errorf("timezone offset minutes out of range (0-59), got %d in %q", minutes, s)
			}
			offsetSecs := hours*3600 + minutes*60
			if offsetSecs > 86400 {
				return nil, pos, fmt.Errorf("timezone offset out of range (max 24h), got %d seconds in %q", offsetSecs, s)
			}
			return time.FixedZone("", sign*offsetSecs), i, nil
		default:
			return nil, pos, fmt.Errorf("expected 2- or 4-digit offset, got %d digits in %q", digits, s)
		}
	}

	// try IANA timezone database
	for i < len(s) && s[i] != ' ' {
		i++
	}
	if i == pos {
		return nil, pos, fmt.Errorf("expected timezone, got %q", s)
	}
	tz := s[pos:i]
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, pos, fmt.Errorf("expected timezone, got %q in %q: %w", tz, s, err)
	}

	return loc, i, nil
}

// handleWeekday parses a weekday name from s starting at position pos and returns the weekday,
// position after the weekday name, and whether a weekday was found. Weekday names can be
// abbreviated ("Mon") or full ("Monday") and are case-insensitive.
func handleWeekday(s string, pos int) (time.Weekday, int, bool) {
	word, i := readWord(s, pos)
	if word == "" {
		return 0, pos, false
	}

	switch strings.ToLower(word) {
	case "mon", "monday":
		return time.Monday, i, true
	case "tue", "tuesday":
		return time.Tuesday, i, true
	case "wed", "wednesday":
		return time.Wednesday, i, true
	case "thu", "thursday":
		return time.Thursday, i, true
	case "fri", "friday":
		return time.Friday, i, true
	case "sat", "saturday":
		return time.Saturday, i, true
	case "sun", "sunday":
		return time.Sunday, i, true
	}

	return 0, pos, false
}

// handleUnix parses a unix timestamp with optional fractional seconds from s and returns
// the parsed time and any error.
func handleUnix(s string) (time.Time, error) {
	num, i, err := readNum(s, 0)
	if err != nil {
		return time.Time{}, err
	}
	nsec := 0
	if i < len(s) && s[i] == '.' {
		i++
		nsec, i, err = readFrac(s, i)
		if err != nil {
			return time.Time{}, err
		}
	}
	if i < len(s) {
		return time.Time{}, fmt.Errorf("expected end of input, got %q in %q", s[i:], s)
	}
	return time.Unix(int64(num), int64(nsec)), nil
}

// ParseTimespan parses a time span string and returns the duration.
//
// Time spans are sequences of numeric values with optional time units. Separating
// spaces may be omitted. All values are added together (e.g. "2h 30min" is 150 minutes).
// Numeric values can include decimal points. If no unit is specified, seconds are
// assumed. Unit names are case-sensitive and only English names are accepted.
//
// The following time units are supported:
//
//	nsec, ns
//	usec, us, μs
//	msec, ms
//	seconds, second, sec, s
//	minutes, minute, min, m
//	hours, hour, hr, h
//	days, day, d
//	weeks, week, w
//	months, month, M (defined as 30.4375 days)
//	years, year, y (defined as 365.25 days)
//
// Examples for valid time spans:
//
//	2 h
//	2hours
//	48hr
//	1y 12month
//	55s500ms
//	300ms20s 5day
//	1.5h
//	60
func ParseTimespan(s string) (time.Duration, error) {
	switch s {
	case "":
		return 0, errors.New("expected time span, got empty string")
	case "0":
		return 0, nil
	}

	var d time.Duration
	foundAny := false
	for i := 0; i < len(s); {
		// skip spaces
		for i < len(s) && s[i] == ' ' {
			i++
		}

		// break if we reached the end
		if i >= len(s) {
			break
		}

		// read number
		var num int
		var err error
		if s[i] >= '0' && s[i] <= '9' {
			num, i, err = readNum(s, i)
			if err != nil {
				return 0, err
			}
		} else if s[i] != '.' {
			return 0, fmt.Errorf("expected number, got %q in %q", string(s[i]), s)
		}
		nsec := 0
		if i < len(s) && s[i] == '.' {
			i++
			nsec, i, err = readFrac(s, i)
			if err != nil {
				return 0, err
			}
		}

		// skip spaces again
		for i < len(s) && s[i] == ' ' {
			i++
		}

		// read unit
		var unit time.Duration
		var unitStr string
		unitStr, i = readWord(s, i)
		if unitStr == "" {
			unit = Second // no unit specified, default to seconds
		} else {
			// switch was ca. 20% faster than a map in my tests
			switch unitStr {
			case "ns", "nsec":
				unit = Nanosecond
			case "us", "µs", "μs", "usec": // 1st is the micro symbol (U+00B5), 2nd is the Greek letter mu (U+03BC)
				unit = Microsecond
			case "ms", "msec":
				unit = Millisecond
			case "s", "sec", "second", "seconds":
				unit = Second
			case "m", "min", "minute", "minutes":
				unit = Minute
			case "h", "hr", "hour", "hours":
				unit = Hour
			case "d", "day", "days":
				unit = Day
			case "w", "week", "weeks":
				unit = Week
			case "M", "month", "months":
				unit = Month
			case "y", "year", "years":
				unit = Year
			default:
				return 0, fmt.Errorf("expected unit, got %q in %q", unitStr, s)
			}
		}

		d += time.Duration(num) * unit
		if nsec > 0 {
			if unit >= Second {
				d += time.Duration(nsec) * (unit / Second)
			} else {
				d += time.Duration(nsec) / (Second / unit)
			}
		}
		foundAny = true
	}

	if !foundAny {
		return 0, fmt.Errorf("expected time span, got %q", s)
	}

	return d, nil
}

// ParseTimestamp parses a timestamp string and returns the time.
//
// Timestamps consist of optional weekday, date, time, and timezone. Fields can be
// omitted. Dates are specified as YYYY-MM-DD or YY-MM-DD (0-68 is 2000-2068, 69-99
// is 1969-1999). Times are specified as HH:MM:SS or HH:MM (seconds default to 0).
// The space between date and time can be replaced with "T" (RFC 3339), but only
// when the year is 4 digits.
//
// The timezone defaults to the current timezone if not specified. It may be given
// after a space as: "UTC", an IANA timezone database entry (e.g. "Asia/Tokyo"), or
// an offset in ±HH:MM, ±HHMM, or ±HH format. It may also be affixed directly to the
// timestamp in RFC 3339 format: "Z" or "±HH:MM".
//
// A timestamp can start with a weekday in abbreviated ("Wed") or full ("Wednesday")
// English form (case-insensitive). If specified, the weekday must match the date.
//
// If the date is omitted, it defaults to today. If the time is omitted, it defaults
// to 00:00:00. Fractional seconds can be specified. Seconds can also be omitted,
// defaulting to 0.
//
// Special tokens "now", "today", "yesterday", and "tomorrow" may be used. "now"
// refers to the current time. "today", "yesterday", and "tomorrow" refer to 00:00:00
// of the respective day and may be followed by a timezone.
//
// Relative times are time spans (see ParseTimespan) prefixed with "+" or "-", or
// suffixed with " ago" or " left".
//
// Finally, an integer prefixed with "@" is evaluated relative to the UNIX epoch
// (1970-01-01 00:00:00 UTC). Fractional seconds are supported.
//
// Examples for valid timestamps:
//
//	now
//	today
//	yesterday UTC
//	tomorrow Pacific/Auckland
//	2009-11-10
//	2009-11-10 18:15:22
//	2009-11-10 11:12:13.654321
//	Tue 2009-11-10 18:15:22 UTC
//	2009-11-10T18:15:22Z
//	18:15:22
//	11:12:13.5
//	18:15:22 +05:30
//	+5h
//	-10m
//	5min ago
//	@1234567890
//	@1234567890.987
//
// The optional now parameter specifies the reference time for relative timestamps.
// If not provided, the current time is used.
func ParseTimestamp(s string, now ...time.Time) (time.Time, error) {
	ref := time.Now()
	if len(now) > 0 {
		ref = now[0]
	}

	switch s {
	case "":
		return time.Time{}, errors.New("expected timestamp, got empty string")
	case "now":
		return ref, nil
	}

	c := s[0]

	// unix
	if c == '@' {
		if len(s) == 1 {
			return time.Time{}, fmt.Errorf("expected number after %q in %q", c, s)
		}
		return handleUnix(s[1:])
	}

	// relative
	switch {
	case c == '-':
		d, err := ParseTimespan(s[1:])
		if err != nil {
			return time.Time{}, err
		}
		return ref.Add(-d), nil
	case c == '+':
		d, err := ParseTimespan(s[1:])
		if err != nil {
			return time.Time{}, err
		}
		return ref.Add(d), nil
	case strings.HasSuffix(s, " ago"):
		d, err := ParseTimespan(s[:len(s)-4])
		if err != nil {
			return time.Time{}, err
		}
		return ref.Add(-d), nil
	case strings.HasSuffix(s, " left"):
		d, err := ParseTimespan(s[:len(s)-5])
		if err != nil {
			return time.Time{}, err
		}
		return ref.Add(d), nil
	}

	// starts with letter (special token or weekday)
	if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
		if t, matched, err := handleToken(s, ref); matched {
			return t, err
		}
	}

	// parse full timestamp: date and/or time with optional weekday/timezone
	if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
		year, m, day := ref.Date()
		month := int(m)
		hour, minute, second, nsec := 0, 0, 0, 0
		loc := ref.Location()
		var expectedWeekday time.Weekday
		foundWeekday := false

		i := 0

		// try to parse optional weekday
		wd, i, found := handleWeekday(s, i)
		if found {
			expectedWeekday = wd
			foundWeekday = true

			// skip spaces after weekday
			for i < len(s) && s[i] == ' ' {
				i++
			}
		}

		// determine if we have a date or time
		foundColon := false
		foundDash := false
		if i < len(s) && s[i] >= '0' && s[i] <= '9' {
			// look ahead for colon or dash
			for j := i; j < len(s) && j < i+5; j++ {
				if s[j] == ':' {
					foundColon = true
					break
				}
				if s[j] == '-' {
					foundDash = true
					break
				}
			}
		}

		// try to parse date (if dash detected and no colon)
		if i < len(s) && foundDash && !foundColon {
			var fullYear bool
			var err error
			year, month, day, i, fullYear, err = handleDate(s, i)
			if err != nil {
				return time.Time{}, err
			}

			// skip spaces after date, or 'T' if full year
			if i < len(s) && s[i] == 'T' {
				if !fullYear {
					return time.Time{}, fmt.Errorf("expected 4-digit year before 'T' separator, got 2-digit year in %q", s)
				}
				i++
			} else {
				for i < len(s) && s[i] == ' ' {
					i++
				}
			}
		}

		// try to parse time (if present)
		if i < len(s) && (s[i] >= '0' && s[i] <= '9') {
			// if no date was parsed, there must be a colon
			if !foundDash && !foundColon {
				return time.Time{}, fmt.Errorf("expected ':' in time-only format, got %q", s)
			}
			var err error
			hour, minute, second, nsec, i, err = handleTime(s, i)
			if err != nil {
				return time.Time{}, err
			}

			// skip spaces after time
			for i < len(s) && s[i] == ' ' {
				i++
			}

			// try to parse timezone directly after time
			if i < len(s) && (s[i] == '+' || s[i] == '-' || s[i] == 'Z' ||
				(s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z')) {
				loc, i, err = handleTimezone(s, i)
				if err != nil {
					return time.Time{}, err
				}
			}
		} else if i < len(s) {
			// try to parse timezone after date only
			var err error
			loc, i, err = handleTimezone(s, i)
			if err != nil {
				return time.Time{}, err
			}
		}

		if i < len(s) {
			return time.Time{}, fmt.Errorf("expected end of input, got %q in %q", s[i:], s)
		}

		if foundWeekday && !foundDash {
			return time.Time{}, fmt.Errorf("expected date after weekday in %q", s)
		}

		t := time.Date(year, time.Month(month), day, hour, minute, second, nsec, loc)

		// validate weekday if it was specified
		if foundWeekday && t.Weekday() != expectedWeekday {
			return time.Time{}, fmt.Errorf("expected weekday %s for %s, got %s in %q",
				expectedWeekday, t.Format("2009-11-10"), t.Weekday(), s)
		}

		return t, nil
	}

	return time.Time{}, fmt.Errorf("expected timestamp, got %q", s)
}
