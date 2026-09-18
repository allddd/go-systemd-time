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

// systemd only accepts calendar events within these years.
const (
	minCalYear = 1970
	maxCalYear = 2199
)

// calValue is a single value in a calendar field, which is a plain number except in the
// second field where it is nanoseconds.
type calValue struct {
	start int64
	end   int64 // -1 if not a range
	step  int64 // 0 if no repetition
}

// calField is a single component (year, month, day, hour, minute, or second) of a
// calendar event.
type calField struct {
	wildcard bool // matches any value
	values   []calValue
	fromEnd  bool // values are counted from the end of the month (day field only)
}

// CalendarEvent is a parsed calendar event that refers to one or more points in time.
type CalendarEvent struct {
	weekdays uint8 // bitmask from Monday (bit 0) to Sunday (bit 6), 0 is no filter
	year     calField
	month    calField
	day      calField
	hour     calField
	minute   calField
	second   calField
	loc      *time.Location // nil is the local timezone
}

// readAlpha reads all alphabetic characters from s starting at position pos and returns
// the string and the position after it.
func readAlpha(s string, pos int) (string, int) {
	i := pos
	for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z')) {
		i++
	}
	return s[pos:i], i
}

// readCalNum reads a calendar field number from s starting at position pos and returns
// the number, the position after the number, and any error. Numbers in the second field
// can have a fraction and are returned as nanoseconds.
func readCalNum(s string, pos int, sec bool) (int64, int, error) {
	num, i, err := readNum(s, pos)
	if err != nil {
		return 0, pos, err
	}
	if !sec {
		return int64(num), i, nil
	}
	nsec := int64(num) * int64(Second)
	if i < len(s) && s[i] == '.' {
		i++
		var frac int
		frac, i, err = readFrac(s, i)
		if err != nil {
			return 0, pos, err
		}
		nsec += int64(frac)
	}
	return nsec, i, nil
}

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

// calAny returns a calendar field that matches any value.
func calAny() calField {
	return calField{wildcard: true, values: nil, fromEnd: false}
}

// calVals returns a calendar field that matches the given values.
func calVals(nums ...int64) calField {
	values := make([]calValue, len(nums))
	for i, num := range nums {
		values[i] = calValue{start: num, end: -1, step: 0}
	}
	return calField{wildcard: false, values: values, fromEnd: false}
}

// newCalEvent returns a calendar event with the default fields, which match any date at
// 00:00:00 in the local timezone.
func newCalEvent() CalendarEvent {
	return CalendarEvent{
		weekdays: 0,
		year:     calAny(),
		month:    calAny(),
		day:      calAny(),
		hour:     calVals(0),
		minute:   calVals(0),
		second:   calVals(0),
		loc:      nil,
	}
}

// expandYear expands a 2-digit year into a full year. 0-68 is 2000-2068, 69-99 is
// 1969-1999. systemd uses the same rule for timestamps, but rejects anything before the
// epoch, and maps 69 to 2069 in calendar events for whatever reason.
func expandYear(year int) int {
	if year <= 68 {
		return year + 2000
	}
	return year + 1900
}

// expandCalYear expands 2-digit years in the calendar field f into full years.
func expandCalYear(f *calField) {
	for i := range f.values {
		v := &f.values[i]
		if v.start < 100 {
			v.start = int64(expandYear(int(v.start)))
		}
		if v.end >= 0 && v.end < 100 {
			v.end = int64(expandYear(int(v.end)))
		}
	}
}

// monFirst converts a Sunday-first time.Weekday to a Monday-first index (Monday is 0,
// Sunday is 6).
func monFirst(wd time.Weekday) int {
	return (int(wd) + 6) % 7
}

// daysInMonth returns the number of days in the given month of the given year.
func daysInMonth(year int, month time.Month) int {
	switch month { //nolint:exhaustive
	case time.April, time.June, time.September, time.November:
		return 30
	case time.February:
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	}
	return 31
}

// firstCalSep returns the first ':' or '-' in s starting at position pos, or 0 if there
// is none before a space or the end of s.
func firstCalSep(s string, pos int) byte {
	for i := pos; i < len(s) && s[i] != ' '; i++ {
		if s[i] == ':' || s[i] == '-' {
			return s[i]
		}
	}
	return 0
}

// checkCalField returns an error if any value in f is outside the range lo-hi. Values are
// divided by unit before they are checked, which is 1 for all fields except the second
// field where values are nanoseconds.
func checkCalField(s string, f calField, name string, lo, hi, unit int64) error {
	for _, v := range f.values {
		start, end := v.start/unit, v.end/unit
		if start < lo || start > hi {
			return fmt.Errorf("expected %s in range %d-%d, got %d in %q", name, lo, hi, start, s)
		}
		if v.end >= 0 && (end < lo || end > hi) {
			return fmt.Errorf("expected %s in range %d-%d, got %d in %q", name, lo, hi, end, s)
		}
	}
	return nil
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
		year = expandYear(year)
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
					return nil, pos, fmt.Errorf("expected offset minutes in range 0-59, got %d in %q", minutes, s)
				}
				offsetSecs := hours*3600 + minutes*60
				if offsetSecs > 86400 { // 24h is the maximum allowed offset
					return nil, pos, fmt.Errorf("expected offset in range 0-86400 seconds, got %d in %q", offsetSecs, s)
				}
				return time.FixedZone("", sign*offsetSecs), i, nil
			}
			if hours > 24 {
				return nil, pos, fmt.Errorf("expected offset in range 0-24h, got %dh in %q", hours, s)
			}
			return time.FixedZone("", sign*hours*3600), i, nil // 3600 seconds per hour
		case 4: // 4 is the digit count for HHMM format
			hours, minutes := num/100, num%100
			if minutes >= 60 {
				return nil, pos, fmt.Errorf("expected offset minutes in range 0-59, got %d in %q", minutes, s)
			}
			offsetSecs := hours*3600 + minutes*60
			if offsetSecs > 86400 {
				return nil, pos, fmt.Errorf("expected offset in range 0-86400 seconds, got %d in %q", offsetSecs, s)
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
	name, i := readAlpha(s, pos)
	if name == "" {
		return 0, pos, false
	}

	switch strings.ToLower(name) {
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

// handleCalValue parses a single calendar field value from s starting at position pos and
// returns the value, the position after it, and any error. Values can be a single number,
// a range ("1..5"), a repetition ("1/2"), or a range with a repetition ("1..5/2").
func handleCalValue(s string, pos int, sec bool) (calValue, int, error) {
	var num int64

	// parse value
	start, i, err := readCalNum(s, pos, sec)
	if err != nil {
		return calValue{}, pos, err
	}
	v := calValue{start: start, end: -1, step: 0}

	// parse (optional) range end
	if i+1 < len(s) && s[i] == '.' && s[i+1] == '.' {
		i += 2 // 2 is length of ".."
		num, i, err = readCalNum(s, i, sec)
		if err != nil {
			return calValue{}, pos, err
		}
		if num < start {
			return calValue{}, pos, fmt.Errorf("expected range end after start, got %d..%d in %q", start, num, s)
		}
		v.end = num
	}

	// parse (optional) repetition
	if i < len(s) && s[i] == '/' {
		i++
		num, i, err = readCalNum(s, i, sec)
		if err != nil {
			return calValue{}, pos, err
		}
		if num == 0 {
			return calValue{}, pos, fmt.Errorf("expected repetition greater than 0 in %q", s)
		}
		v.step = num
	}

	return v, i, nil
}

// handleCalValues parses a comma-separated list of calendar field values from s starting at
// position pos and returns the values, the position after the list, and any error.
func handleCalValues(s string, pos int, sec bool) ([]calValue, int, error) {
	var values []calValue
	i := pos
	for {
		var v calValue
		var err error
		v, i, err = handleCalValue(s, i, sec)
		if err != nil {
			return nil, pos, err
		}
		values = append(values, v)

		// stop unless another value follows
		if i >= len(s) || s[i] != ',' {
			break
		}
		i++
	}

	return values, i, nil
}

// handleCalField parses a calendar field from s starting at position pos and returns the
// field, the position after it, and any error. Fields are either "*" (any value) or a list
// of values.
func handleCalField(s string, pos int, sec bool) (calField, int, error) {
	if pos >= len(s) {
		return calField{}, pos, fmt.Errorf("expected calendar field, got %q", s)
	}
	if s[pos] == '*' {
		return calAny(), pos + 1, nil
	}

	values, i, err := handleCalValues(s, pos, sec)
	if err != nil {
		return calField{}, pos, err
	}

	return calField{wildcard: false, values: values, fromEnd: false}, i, nil
}

// handleCalWeekdays parses a comma-separated list of weekdays from s starting at position pos
// and returns the weekday bitmask, the position after the list, whether any weekdays were
// found, and any error. Two weekdays separated by ".." are a range of continuous weekdays.
func handleCalWeekdays(s string, pos int) (uint8, int, bool, error) {
	var mask uint8
	var wd, wdEnd time.Weekday
	var found bool

	i := pos
	for {
		wd, i, found = handleWeekday(s, i)
		if !found {
			break
		}
		first, last := monFirst(wd), monFirst(wd)

		// parse (optional) weekday range
		if i+1 < len(s) && s[i] == '.' && s[i+1] == '.' {
			i += 2 // 2 is length of ".."
			wdEnd, i, found = handleWeekday(s, i)
			if !found {
				return 0, pos, false, fmt.Errorf("expected weekday after '..' in %q", s)
			}
			last = monFirst(wdEnd)
			if last < first {
				return 0, pos, false, fmt.Errorf("expected weekday range in order, got %s..%s in %q", wd, wdEnd, s)
			}
		}

		for d := first; d <= last; d++ {
			mask |= 1 << uint(d)
		}

		// stop unless a comma follows
		if i >= len(s) || s[i] != ',' {
			break
		}
		i++
	}

	if mask == 0 {
		return 0, pos, false, nil
	}

	return mask, i, true, nil
}

// handleCalDate parses a calendar date from s starting at position pos and returns the year,
// month, and day fields, the position after the date, and any error. Dates are specified as
// YYYY-MM-DD, YY-MM-DD, or MM-DD (the year matches any value). The day can be separated
// with "~" instead of "-" to count days from the end of the month.
func handleCalDate(s string, pos int) (calField, calField, calField, int, error) {
	// parse year, or month if the date has only two fields
	first, i, err := handleCalField(s, pos, false)
	if err != nil {
		return calField{}, calField{}, calField{}, pos, err
	}
	if i >= len(s) || s[i] != '-' {
		return calField{}, calField{}, calField{}, pos, fmt.Errorf("expected '-' in date, got %q", s)
	}
	i++

	// parse month, or day if the date has only two fields
	second, i, err := handleCalField(s, i, false)
	if err != nil {
		return calField{}, calField{}, calField{}, pos, err
	}

	// two-field date, the year matches any value
	if i >= len(s) || (s[i] != '-' && s[i] != '~') {
		if err := checkCalField(s, first, "month", 1, 12, 1); err != nil {
			return calField{}, calField{}, calField{}, pos, err
		}
		if err := checkCalField(s, second, "day", 1, 31, 1); err != nil {
			return calField{}, calField{}, calField{}, pos, err
		}
		return calAny(), first, second, i, nil
	}

	// parse day, "~" counts days from the end of the month
	fromEnd := s[i] == '~'
	i++
	day, i, err := handleCalField(s, i, false)
	if err != nil {
		return calField{}, calField{}, calField{}, pos, err
	}
	if fromEnd {
		if day.wildcard {
			return calField{}, calField{}, calField{}, pos, fmt.Errorf("expected day after '~' in %q", s)
		}
		day.fromEnd = true
	}

	year := first
	expandCalYear(&year)
	if err := checkCalField(s, year, "year", minCalYear, maxCalYear, 1); err != nil {
		return calField{}, calField{}, calField{}, pos, err
	}
	if err := checkCalField(s, second, "month", 1, 12, 1); err != nil {
		return calField{}, calField{}, calField{}, pos, err
	}
	maxDay := int64(31) // 31 is max days in any month
	if fromEnd {
		maxDay = 28 // 28 is min days in any month
	}
	if err := checkCalField(s, day, "day", 1, maxDay, 1); err != nil {
		return calField{}, calField{}, calField{}, pos, err
	}

	return year, second, day, i, nil
}

// handleCalTime parses a calendar time from s starting at position pos and returns the hour,
// minute, and second fields, the position after the time, and any error. Times are specified
// as HH:MM:SS or HH:MM (seconds default to 0). Fractional seconds are supported.
func handleCalTime(s string, pos int) (calField, calField, calField, int, error) {
	// parse hour
	hour, i, err := handleCalField(s, pos, false)
	if err != nil {
		return calField{}, calField{}, calField{}, pos, err
	}
	if err := checkCalField(s, hour, "hour", 0, 23, 1); err != nil {
		return calField{}, calField{}, calField{}, pos, err
	}
	if i >= len(s) || s[i] != ':' {
		return calField{}, calField{}, calField{}, pos, fmt.Errorf("expected time (HH:MM or HH:MM:SS), got %q", s)
	}
	i++

	// parse minute
	minute, i, err := handleCalField(s, i, false)
	if err != nil {
		return calField{}, calField{}, calField{}, pos, err
	}
	if err := checkCalField(s, minute, "minute", 0, 59, 1); err != nil {
		return calField{}, calField{}, calField{}, pos, err
	}

	// parse (optional) second
	second := calVals(0)
	if i < len(s) && s[i] == ':' {
		i++
		second, i, err = handleCalField(s, i, true) // seconds are read as nanoseconds
		if err != nil {
			return calField{}, calField{}, calField{}, pos, err
		}
		if err := checkCalField(s, second, "second", 0, 59, int64(Second)); err != nil {
			return calField{}, calField{}, calField{}, pos, err
		}
	}

	return hour, minute, second, i, nil
}

// handleCalKeyword parses special shorthands ("minutely", "hourly", "daily", "weekly",
// "monthly", "yearly", "annually", "quarterly", or "semiannually") with optional timezone and
// returns the calendar event, whether a shorthand was found, and any error. Shorthands are
// case-sensitive (must be lowercase).
func handleCalKeyword(s string) (CalendarEvent, bool, error) {
	keyword, i := readAlpha(s, 0)

	ce := newCalEvent()

	switch keyword {
	case "minutely":
		ce.hour, ce.minute = calAny(), calAny()
	case "hourly":
		ce.hour = calAny()
	case "daily": // default fields already match
	case "weekly":
		ce.weekdays = 1 << uint(monFirst(time.Monday))
	case "monthly":
		ce.day = calVals(1)
	case "yearly", "annually":
		ce.month, ce.day = calVals(1), calVals(1)
	case "quarterly":
		ce.month, ce.day = calVals(1, 4, 7, 10), calVals(1)
	case "semiannually":
		ce.month, ce.day = calVals(1, 7), calVals(1)
	default:
		return ce, false, nil
	}

	// parse (optional) timezone after shorthand
	for i < len(s) && s[i] == ' ' {
		i++
	}
	if i < len(s) {
		loc, j, err := handleTimezone(s, i)
		if err != nil {
			return CalendarEvent{}, true, err
		}
		if j < len(s) {
			return CalendarEvent{}, true, fmt.Errorf("expected end of input, got %q in %q", s[j:], s)
		}
		ce.loc = loc
	}

	return ce, true, nil
}

// nextCalValue returns the smallest value matched by v in the range from-last, or -1 if there
// is none. from is the value the search starts at and last is the highes value the component
// can have before it rolls over into the next larger one, e.g. 23 for the hour.
func nextCalValue(v calValue, from, last int64) int64 {
	step, end := v.step, v.end
	if step == 0 { // no repetition, so every value in the range matches
		step = 1
	}
	if end < 0 { // not a range
		end = v.start
		if v.step > 0 { // a repetition without a range end continues until last
			end = last
		}
	}
	if end > last {
		end = last
	}
	if v.start > end || from > end {
		return -1
	}
	if v.start >= from {
		return v.start
	}

	next := v.start + (from-v.start+step-1)/step*step // round up to the next multiple of step
	if next > end {
		return -1
	}

	return next
}

// nextVal returns the smallest value matched by f in the range from-last, and whether such a
// value was found. from and last are the same as in nextCalValue, so a value that was not
// found means the next larger component has to be advanced.
func (f calField) nextVal(from, last int64) (int64, bool) {
	if f.wildcard {
		if from > last {
			return 0, false
		}
		return from, true
	}

	best := int64(-1)
	for _, v := range f.values {
		if next := nextCalValue(v, from, last); next >= 0 && (best < 0 || next < best) {
			best = next
		}
	}

	return best, best >= 0
}

// nextDay returns the smallest day matched by f in the given month of the given year that is
// not before the given day, or -1 if there is none.
func (f calField) nextDay(year int, month time.Month, day int) int {
	lastDay := int64(daysInMonth(year, month))

	if !f.fromEnd {
		next, found := f.nextVal(int64(day), lastDay)
		if !found {
			return -1
		}
		return int(next)
	}

	// days are counted from the end of the month, so flip them into actual days first
	best := int64(-1)
	for _, v := range f.values {
		flipped := calValue{start: lastDay - v.start + 1, end: -1, step: v.step}
		if v.end >= 0 { // the range end is the first day and the range start is the last one
			flipped.start, flipped.end = lastDay-v.end+1, lastDay-v.start+1
		} else if v.step > 0 { // a repetition continues until the end of the month
			flipped.end = lastDay
		}
		if next := nextCalValue(flipped, int64(day), lastDay); next >= 0 && (best < 0 || next < best) {
			best = next
		}
	}

	return int(best)
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
// Relative times are time spans (see [ParseTimespan]) prefixed with "+" or "-", or
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
			return time.Time{}, fmt.Errorf("expected number after %q in %q", string(c), s)
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
				expectedWeekday, t.Format("2006-01-02"), t.Weekday(), s)
		}

		return t, nil
	}

	return time.Time{}, fmt.Errorf("expected timestamp, got %q", s)
}

// ParseCalendarEvent parses a calendar event string and returns the calendar event.
//
// Calendar events refer to one or more points in time and consist of optional weekdays,
// date, time, and timezone. Dates are specified as YYYY-MM-DD, YY-MM-DD (0-68 is 2000-2068,
// 69-99 is 1969-1999), or MM-DD (the year defaults to *). Years must be in range 1970-2199.
// Times are specified as HH:MM:SS or HH:MM (seconds default to 0). Fractional seconds can be
// specified.
//
// Any date or time component may be "*", which matches any value, a comma-separated list of
// values, a range of values (v..w), or a repetition (v/s), which matches the value itself and
// the value plus all multiples of s. Ranges and repetitions may be combined (v..w/s). The day
// may be separated with "~" instead of "-" to count days from the end of the month, e.g.
// "*-02~03" is the third last day in February.
//
// A calendar event can start with weekdays in abbreviated ("Wed") or full ("Wednesday")
// English form (case-insensitive), separated by commas. Two weekdays separated by ".." are a
// range of continuous weekdays.
//
// The timezone defaults to the local timezone if not specified. It may be given after a space
// as: "UTC", an IANA timezone database entry (e.g. "Asia/Tokyo"), or an offset in ±HH:MM,
// ±HHMM, or ±HH format.
//
// If the date is omitted, it defaults to *-*-*. If the time is omitted, it defaults to
// 00:00:00.
//
// The following special shorthands may be used, optionally followed by a timezone:
//
//	minutely     = *-*-* *:*:00
//	hourly       = *-*-* *:00:00
//	daily        = *-*-* 00:00:00
//	monthly      = *-*-01 00:00:00
//	weekly       = Mon *-*-* 00:00:00
//	yearly       = *-01-01 00:00:00
//	annually     = *-01-01 00:00:00
//	quarterly    = *-01,04,07,10-01 00:00:00
//	semiannually = *-01,07-01 00:00:00
//
// Examples for valid calendar events:
//
//	daily
//	weekly Pacific/Auckland
//	Mon
//	Mon..Fri
//	Sat,Sun 08:05:40
//	05:40
//	*:2/3
//	12..14:10,20,30
//	05:40:23.420000/3.170001
//	03-05
//	2003-02..04-05
//	*-01,04,07,10-01
//	*-*~1 18:00:00
//	Mon *-05~07/1
//	Mon,Sun 12-*-* 2,1:23
//	mon,fri *-1/2-1,3 *:30:45
//	Thu,Fri 2012-*-1,5 11:12:13
//	2003-03-05 05:40
//	*-*-* 00:00:00 UTC
//	*-*-* 12:00:00 +05:30
//
// Use [CalendarEvent.Next] to get the next time a calendar event matches.
func ParseCalendarEvent(s string) (CalendarEvent, error) {
	if s == "" {
		return CalendarEvent{}, errors.New("expected calendar event, got empty string")
	}

	c := s[0]

	// starts with letter (shorthand or weekday)
	if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
		if ce, matched, err := handleCalKeyword(s); matched {
			return ce, err
		}
	}

	// parse full calendar event: date and/or time with optional weekdays/timezone
	if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '*' {
		ce := newCalEvent()
		foundWeekday, foundDate, foundTime := false, false, false

		i := 0

		// try to parse optional weekdays
		mask, i, found, err := handleCalWeekdays(s, i)
		if err != nil {
			return CalendarEvent{}, err
		}
		if found {
			if mask != 127 { // 127 is all seven weekday bits, which is the same as no filter
				ce.weekdays = mask
			}
			foundWeekday = true

			// skip spaces after weekdays
			for i < len(s) && s[i] == ' ' {
				i++
			}
		}

		// try to parse date (if it starts with a digit or '*' and a dash comes first)
		if i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '*') && firstCalSep(s, i) == '-' {
			ce.year, ce.month, ce.day, i, err = handleCalDate(s, i)
			if err != nil {
				return CalendarEvent{}, err
			}
			foundDate = true

			// skip spaces after date
			for i < len(s) && s[i] == ' ' {
				i++
			}
		}

		// try to parse time (if it starts with a digit or '*' and a colon comes first)
		if i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '*') && firstCalSep(s, i) == ':' {
			ce.hour, ce.minute, ce.second, i, err = handleCalTime(s, i)
			if err != nil {
				return CalendarEvent{}, err
			}
			foundTime = true

			// skip spaces after time
			for i < len(s) && s[i] == ' ' {
				i++
			}
		}

		if !foundWeekday && !foundDate && !foundTime {
			return CalendarEvent{}, fmt.Errorf("expected calendar event, got %q", s)
		}

		// try to parse timezone
		if i < len(s) {
			ce.loc, i, err = handleTimezone(s, i)
			if err != nil {
				return CalendarEvent{}, err
			}
		}

		if i < len(s) {
			return CalendarEvent{}, fmt.Errorf("expected end of input, got %q in %q", s[i:], s)
		}

		return ce, nil
	}

	return CalendarEvent{}, fmt.Errorf("expected calendar event, got %q", s)
}

// Next returns the next time matched by the calendar event that is strictly after t and
// whether such a time was found. Events that do not match again before the last accepted
// year, like a date in the past, are reported as not found. If the calendar event has no
// timezone, the timezone of t is used.
func (c CalendarEvent) Next(t time.Time) (time.Time, bool) {
	loc := c.loc
	if loc == nil {
		loc = t.Location()
	}

	next := t.In(loc).Add(Nanosecond)
	limit := time.Date(maxCalYear+1, time.January, 1, 0, 0, 0, 0, loc)

	for !next.After(limit) {
		year, month, day := next.Date()
		hour, minute, sec := next.Clock()
		nsec := int64(sec)*int64(Second) + int64(next.Nanosecond())

		// match year
		nextYear, found := c.year.nextVal(int64(year), maxCalYear)
		if !found {
			return time.Time{}, false
		}
		if int(nextYear) != year {
			next = time.Date(int(nextYear), time.January, 1, 0, 0, 0, 0, loc)
			continue
		}

		// match month
		nextMonth, found := c.month.nextVal(int64(month), 12)
		if !found {
			next = time.Date(year+1, time.January, 1, 0, 0, 0, 0, loc)
			continue
		}
		if time.Month(nextMonth) != month {
			next = time.Date(year, time.Month(nextMonth), 1, 0, 0, 0, 0, loc)
			continue
		}

		// match day
		nextDay := c.day.nextDay(year, month, day)
		if nextDay < 0 {
			next = time.Date(year, month+1, 1, 0, 0, 0, 0, loc)
			continue
		}
		if nextDay != day {
			next = time.Date(year, month, nextDay, 0, 0, 0, 0, loc)
			continue
		}

		// match weekday
		if c.weekdays != 0 && c.weekdays>>uint(monFirst(next.Weekday()))&1 == 0 {
			next = time.Date(year, month, day+1, 0, 0, 0, 0, loc)
			continue
		}

		// match hour
		nextHour, found := c.hour.nextVal(int64(hour), 23)
		if !found {
			next = time.Date(year, month, day+1, 0, 0, 0, 0, loc)
			continue
		}
		if int(nextHour) != hour {
			next = time.Date(year, month, day, int(nextHour), 0, 0, 0, loc)
			continue
		}

		// match minute
		nextMinute, found := c.minute.nextVal(int64(minute), 59)
		if !found {
			next = time.Date(year, month, day, hour+1, 0, 0, 0, loc)
			continue
		}
		if int(nextMinute) != minute {
			next = time.Date(year, month, day, hour, int(nextMinute), 0, 0, loc)
			continue
		}

		// match second (in nanoseconds)
		nextNsec, found := c.second.nextVal(nsec, int64(Minute)-1) // 1ns below a minute is max valid second
		if !found {
			next = time.Date(year, month, day, hour, minute+1, 0, 0, loc)
			continue
		}
		if nextNsec != nsec {
			second := int(nextNsec / int64(Second))
			next = time.Date(year, month, day, hour, minute, second, int(nextNsec%int64(Second)), loc)
			continue
		}

		// when daylight saving time ends the clock repeats an hour and time.Date resolves it
		// to the earlier offset, so a match in that hour can be before t
		if !next.After(t) {
			next = time.Date(year, month, day, hour, minute+1, 0, 0, loc)
			continue
		}

		return next, true
	}

	return time.Time{}, false
}
