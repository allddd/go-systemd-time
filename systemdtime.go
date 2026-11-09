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

// Package systemdtime parses time spans, timestamps, and calendar events the way systemd
// does. The spec is the systemd.time(7) man page at
// https://www.freedesktop.org/software/systemd/man/latest/systemd.time.html
package systemdtime

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// The units of a time span. Month and Year are the averages systemd uses.
const (
	Nanosecond  = time.Nanosecond
	Microsecond = time.Microsecond
	Millisecond = time.Millisecond
	Second      = time.Second
	Minute      = time.Minute
	Hour        = time.Hour
	Day         = 24 * Hour
	Week        = 7 * Day
	Month       = Year / 12        // 30.4375 days
	Year        = 365*Day + 6*Hour // 365.25 days
)

// systemd refuses calendar events outside these years.
const (
	minCalYear = 1970
	maxCalYear = 2199
)

// calValue is one entry of a calendar field, either a plain number, a range, or a repetition.
// The numbers are what the user typed, except in the second field where they are nanoseconds.
type calValue struct {
	start int64
	end   int64 // -1 if not a range
	step  int64 // 0 if not a repetition
}

// calField is one component of a calendar event, from the year down to the second.
type calField struct {
	wildcard bool // "*", matches everything
	values   []calValue
	fromEnd  bool // the values count back from the end of the month (day field only)
}

// CalendarEvent is a parsed calendar event, which matches one or more points in time.
type CalendarEvent struct {
	weekdays uint8 // bitmask from Monday (bit 0) to Sunday (bit 6)
	year     calField
	month    calField
	day      calField
	hour     calField
	minute   calField
	second   calField
	loc      *time.Location // nil means the timezone of the time passed to Next
}

// ParseTimespan parses a time span like "2h 30min" into a [time.Duration].
//
// A time span is a list of numbers, each with an optional unit, that all get added up. So
// "2h 30min" is 150 minutes. The spaces in between may be left out. Numbers may have a decimal
// point. A number without a unit is seconds. Units are case-sensitive and only the English
// names work.
//
// These units are understood:
//
//	nsec, ns
//	usec, us, µs, μs
//	msec, ms
//	seconds, second, sec, s
//	minutes, minute, min, m
//	hours, hour, hr, h
//	days, day, d
//	weeks, week, w
//	months, month, M (defined as 30.4375 days)
//	years, year, y (defined as 365.25 days)
//
// Some valid time spans:
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
	i := skipSpaces(s, 0)
	if i == len(s) {
		return 0, fmt.Errorf("expected time span, got %q", s)
	}

	var d time.Duration
	for i < len(s) {
		// the number, a leading dot as in ".5" is allowed
		var num, nsec int
		var err error
		if isDigit(s[i]) {
			num, i, err = readNum(s, i)
			if err != nil {
				return 0, err
			}
		} else if s[i] != '.' {
			return 0, fmt.Errorf("expected number, got %q in %q", string(s[i]), s)
		}
		if i < len(s) && s[i] == '.' {
			nsec, i, err = readFrac(s, i+1)
			if err != nil {
				return 0, err
			}
		}

		var word string
		var unit time.Duration
		word, i = readWord(s, skipSpaces(s, i))
		switch word {
		case "ns", "nsec":
			unit = Nanosecond
		case "us", "µs", "μs", "usec": // micro sign (U+00B5) and Greek mu (U+03BC)
			unit = Microsecond
		case "ms", "msec":
			unit = Millisecond
		case "", "s", "sec", "second", "seconds": // no unit means seconds
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
			return 0, fmt.Errorf("expected unit, got %q in %q", word, s)
		}

		// nsec * unit overflows int64 for anything above seconds, so scale the fraction in two steps
		d += time.Duration(num) * unit
		if unit >= Second {
			d += time.Duration(nsec) * (unit / Second)
		} else {
			d += time.Duration(nsec) / (Second / unit)
		}
		i = skipSpaces(s, i)
	}
	return d, nil
}

// ParseTimestamp parses a timestamp like "Tue 2009-11-10 18:15:22 UTC" into a [time.Time].
//
// A timestamp is a weekday, a date, a time, and a timezone, each of them optional. The date
// is YYYY-MM-DD or YY-MM-DD, where 0-68 means 2000-2068 and 69-99 means 1969-1999. The time
// is HH:MM:SS or HH:MM, the seconds default to 0 and may have a fraction. Date and time are
// separated by a space, or by a "T" as in RFC 3339, but the "T" only works with a 4-digit
// year. No date means today, no time means 00:00:00.
//
// The weekday is English, "Wed" or "Wednesday", case does not matter. If it is given, it has
// to match the date.
//
// Without a timezone the timestamp is in the current timezone. Otherwise the timezone comes
// after a space and is "UTC", an entry of the IANA timezone database like "Asia/Tokyo", or an
// offset in ±HH:MM, ±HHMM, or ±HH format. "Z" and "±HH:MM" may also be glued directly to the
// timestamp, as in RFC 3339.
//
// "now" is the current time. "today", "yesterday", and "tomorrow" are 00:00:00 of that day
// and may be followed by a timezone.
//
// A time span (see [ParseTimespan]) with a "+" or "-" in front of it, or " ago" or " left"
// behind it, is relative to the current time.
//
// "@" followed by an integer counts seconds from the UNIX epoch (1970-01-01 00:00:00 UTC).
// The seconds may have a fraction.
//
// Some valid timestamps:
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
// now is the reference for "now", relative timestamps, today, and the current timezone. Leave
// it out to use the current time.
func ParseTimestamp(s string, now ...time.Time) (time.Time, error) {
	ref := time.Now()
	if len(now) > 0 {
		ref = now[0]
	}

	switch s {
	case "":
		return time.Time{}, fmt.Errorf("expected timestamp, got %q", s)
	case "now":
		return ref, nil
	}

	// unix
	if s[0] == '@' {
		return parseUnix(s[1:])
	}

	// relative
	span, sign := "", 0
	switch {
	case s[0] == '+':
		span, sign = s[1:], 1
	case s[0] == '-':
		span, sign = s[1:], -1
	case strings.HasSuffix(s, " left"):
		span, sign = strings.TrimSuffix(s, " left"), 1
	case strings.HasSuffix(s, " ago"):
		span, sign = strings.TrimSuffix(s, " ago"), -1
	}
	if sign != 0 {
		d, err := ParseTimespan(span)
		if err != nil {
			return time.Time{}, err
		}
		return ref.Add(time.Duration(sign) * d), nil
	}

	// token
	t, ok, err := parseToken(s, ref)
	if err != nil {
		return time.Time{}, err
	}
	if ok {
		return t, nil
	}

	// the long form, weekday, date, time, timezone, all of them optional
	year, month, day := ref.Date()
	hour, minute, sec, nsec := 0, 0, 0, 0
	loc := ref.Location()

	wd, i, hasWeekday := parseWeekday(s, 0)
	if hasWeekday {
		i = skipSpaces(s, i)
	}

	hasDate := firstSep(s, i) == '-'
	if hasWeekday && !hasDate {
		return time.Time{}, fmt.Errorf("expected date after weekday in %q", s)
	}
	if hasDate {
		year, month, day, i, err = parseDate(s, i)
		if err != nil {
			return time.Time{}, err
		}
		// "T" instead of the space only works with a 4-digit year and must be followed by a time
		if i < len(s) && s[i] == 'T' {
			if year < 100 {
				return time.Time{}, fmt.Errorf("expected 4-digit year before 'T' in %q", s)
			}
			i++
			if i >= len(s) || !isDigit(s[i]) {
				return time.Time{}, fmt.Errorf("expected time after 'T' in %q", s)
			}
		} else {
			i = skipSpaces(s, i)
		}
		year = expandYear(year)
	}

	if i < len(s) && isDigit(s[i]) {
		hour, minute, sec, nsec, i, err = parseTime(s, i)
		if err != nil {
			return time.Time{}, err
		}
	} else if !hasDate {
		return time.Time{}, fmt.Errorf("expected timestamp, got %q", s)
	}

	loc, err = parseTimezone(s, i, loc)
	if err != nil {
		return time.Time{}, err
	}

	t = time.Date(year, month, day, hour, minute, sec, nsec, loc)
	if hasWeekday && t.Weekday() != wd {
		return time.Time{}, fmt.Errorf("expected weekday %s for %s, got %s in %q",
			wd, t.Format("2006-01-02"), t.Weekday(), s)
	}
	return t, nil
}

// parseUnix reads seconds since the epoch, with an optional fraction. s is what follows the "@".
func parseUnix(s string) (time.Time, error) {
	sec, i, err := readNum(s, 0)
	if err != nil {
		return time.Time{}, err
	}
	nsec := 0
	if i < len(s) && s[i] == '.' {
		nsec, i, err = readFrac(s, i+1)
		if err != nil {
			return time.Time{}, err
		}
	}
	if i < len(s) {
		return time.Time{}, fmt.Errorf("expected end of input, got %q in %q", s[i:], s)
	}
	return time.Unix(int64(sec), int64(nsec)), nil
}

// parseToken takes care of "today", "yesterday", and "tomorrow", lowercase only, with an
// optional timezone behind them. All three mean 00:00:00 of that day. The bool is false when s
// starts with none of them.
func parseToken(s string, ref time.Time) (time.Time, bool, error) {
	var token string
	var days int
	switch {
	case strings.HasPrefix(s, "today"):
		token = "today"
	case strings.HasPrefix(s, "yesterday"):
		token, days = "yesterday", -1
	case strings.HasPrefix(s, "tomorrow"):
		token, days = "tomorrow", 1
	default:
		return time.Time{}, false, nil
	}
	loc, err := parseTimezone(s, len(token), ref.Location())
	if err != nil {
		return time.Time{}, true, err
	}
	year, month, day := ref.In(loc).Date()
	return time.Date(year, month, day+days, 0, 0, 0, 0, loc), true, nil
}

// parseWeekday accepts "Mon" as well as "Monday", case does not matter. The bool is false when
// there is no weekday at pos.
func parseWeekday(s string, pos int) (time.Weekday, int, bool) {
	word, i := readAlpha(s, pos)
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

// parseDate reads YYYY-MM-DD and YY-MM-DD. The year comes back exactly as written, so the
// caller can still tell a 2-digit year apart before it expands it.
func parseDate(s string, pos int) (int, time.Month, int, int, error) {
	year, i, err := readNum(s, pos)
	if err != nil {
		return 0, 0, 0, pos, err
	}
	if i-pos > 4 || i >= len(s) || s[i] != '-' {
		return 0, 0, 0, pos, fmt.Errorf("expected date (YYYY-MM-DD or YY-MM-DD), got %q", s)
	}
	month, i, err := readBounded(s, i+1, "month", 1, 12)
	if err != nil {
		return 0, 0, 0, pos, err
	}
	if i >= len(s) || s[i] != '-' {
		return 0, 0, 0, pos, fmt.Errorf("expected date (YYYY-MM-DD or YY-MM-DD), got %q", s)
	}
	day, i, err := readBounded(s, i+1, "day", 1, 31)
	if err != nil {
		return 0, 0, 0, pos, err
	}
	return year, time.Month(month), day, i, nil
}

// parseTime reads HH:MM:SS and HH:MM, the seconds may have a fraction. It returns hour, minute,
// second, nanosecond, and where it ends.
func parseTime(s string, pos int) (int, int, int, int, int, error) {
	hour, i, err := readBounded(s, pos, "hour", 0, 23)
	if err != nil {
		return 0, 0, 0, 0, pos, err
	}
	if i >= len(s) || s[i] != ':' {
		return 0, 0, 0, 0, pos, fmt.Errorf("expected time (HH:MM:SS or HH:MM), got %q", s)
	}
	minute, i, err := readBounded(s, i+1, "minute", 0, 59)
	if err != nil {
		return 0, 0, 0, 0, pos, err
	}
	sec, nsec := 0, 0
	if i < len(s) && s[i] == ':' {
		sec, i, err = readBounded(s, i+1, "second", 0, 59)
		if err != nil {
			return 0, 0, 0, 0, pos, err
		}
		if i < len(s) && s[i] == '.' {
			nsec, i, err = readFrac(s, i+1)
			if err != nil {
				return 0, 0, 0, 0, pos, err
			}
		}
	}
	return hour, minute, sec, nsec, i, nil
}

// parseTimezone handles the timezone, which is always the last thing in the input. Whatever is
// left at pos, minus leading spaces, must be "UTC", "Z", an IANA name like "Asia/Tokyo", or an
// offset in ±HH:MM, ±HHMM, or ±HH format. Nothing left means def. systemd only allows "Z" and
// ±HH:MM directly after the timestamp without a space, here any timezone may follow that way.
func parseTimezone(s string, pos int, def *time.Location) (*time.Location, error) {
	i := skipSpaces(s, pos)
	if i == len(s) {
		return def, nil
	}
	if s[i] == '+' || s[i] == '-' {
		loc, j, err := parseOffset(s, i)
		if err != nil {
			return nil, err
		}
		if j < len(s) {
			return nil, fmt.Errorf("expected end of input, got %q in %q", s[j:], s)
		}
		return loc, nil
	}
	name := s[i:]
	if name == "Z" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("expected timezone, got %q in %q: %w", name, s, err)
	}
	return loc, nil
}

// parseOffset reads a ±HH:MM, ±HHMM, or ±HH timezone offset. s[pos] must be the sign.
func parseOffset(s string, pos int) (*time.Location, int, error) {
	sign := 1
	if s[pos] == '-' {
		sign = -1
	}
	num, i, err := readNum(s, pos+1)
	if err != nil {
		return nil, pos, err
	}
	hours, minutes := num, 0
	switch digits := i - pos - 1; digits {
	case 4:
		hours, minutes = num/100, num%100
	case 2:
		if i < len(s) && s[i] == ':' {
			var j int
			minutes, j, err = readNum(s, i+1)
			if err != nil {
				return nil, pos, err
			}
			if j-i-1 != 2 {
				return nil, pos, fmt.Errorf("expected 2-digit offset minutes, got %d digits in %q", j-i-1, s)
			}
			i = j
		}
	default:
		return nil, pos, fmt.Errorf("expected 2- or 4-digit offset, got %d digits in %q", digits, s)
	}
	if minutes > 59 {
		return nil, pos, fmt.Errorf("expected offset minutes in range 0-59, got %d in %q", minutes, s)
	}
	offset := hours*3600 + minutes*60
	if offset > 24*3600 {
		return nil, pos, fmt.Errorf("expected offset in range 0-24 hours, got %d seconds in %q", offset, s)
	}
	return time.FixedZone("", sign*offset), i, nil
}

// ParseCalendarEvent parses a calendar event like "Mon..Fri *-*-* 09:00:00", which stands for
// one or more points in time.
//
// A calendar event is weekdays, a date, a time, and a timezone, each of them optional. The
// date is YYYY-MM-DD, YY-MM-DD, where 0-68 means 2000-2068 and 69-99 means 1969-1999, or
// MM-DD, which leaves the year at "*". Years must be in range 1970-2199. The time is HH:MM:SS
// or HH:MM, the seconds default to 0 and may have a fraction. No date means *-*-*, no time
// means 00:00:00.
//
// Every date and time component can be a "*" for any value, a comma-separated list of values,
// a range (v..w), or a repetition (v/s), which matches v and then every s steps after it. Both
// combine as v..w/s. A "~" instead of the "-" in front of the day counts the days from the end
// of the month, so "*-02~03" is the third last day in February.
//
// The weekdays come first, separated by commas. They are English, "Wed" or "Wednesday", case
// does not matter. "Mon..Fri" is a range of weekdays.
//
// Without a timezone the event uses the timezone of the time passed to [CalendarEvent.Next].
// Otherwise the timezone comes after a space and is "UTC", an entry of the IANA timezone
// database like "Asia/Tokyo", or an offset in ±HH:MM, ±HHMM, or ±HH format.
//
// These shorthands work too, with an optional timezone behind them:
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
// Some valid calendar events:
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
// To find out when an event matches next, call [CalendarEvent.Next].
func ParseCalendarEvent(s string) (CalendarEvent, error) {
	ce := newCalEvent()

	// shorthand
	ok, err := parseCalShorthand(s, &ce)
	if err != nil {
		return CalendarEvent{}, err
	}
	if ok {
		return ce, nil
	}

	// the long form, weekdays, date, time, timezone, all of them optional
	mask, i, err := parseCalWeekdays(s, 0)
	if err != nil {
		return CalendarEvent{}, err
	}
	if mask != 0 {
		ce.weekdays = mask
		i = skipSpaces(s, i)
	}

	if firstSep(s, i) == '-' {
		i, err = parseCalDate(s, i, &ce)
		if err != nil {
			return CalendarEvent{}, err
		}
		i = skipSpaces(s, i)
	}

	if firstSep(s, i) == ':' {
		i, err = parseCalTime(s, i, &ce)
		if err != nil {
			return CalendarEvent{}, err
		}
	}

	if i == 0 {
		return CalendarEvent{}, fmt.Errorf("expected calendar event, got %q", s)
	}

	ce.loc, err = parseTimezone(s, i, nil)
	if err != nil {
		return CalendarEvent{}, err
	}
	return ce, nil
}

// newCalEvent is where parsing starts from, an event for every day at 00:00:00 with no
// timezone of its own.
func newCalEvent() CalendarEvent {
	return CalendarEvent{
		weekdays: 127, // all seven weekdays
		year:     calAny(),
		month:    calAny(),
		day:      calAny(),
		hour:     calVals(0),
		minute:   calVals(0),
		second:   calVals(0),
		loc:      nil,
	}
}

// parseCalShorthand recognizes "minutely", "hourly", "daily", "monthly", "weekly", "yearly",
// "annually", "quarterly", and "semiannually", lowercase only, plus an optional timezone. ce
// must still have its defaults from newCalEvent. The bool is false when s is no shorthand.
func parseCalShorthand(s string, ce *CalendarEvent) (bool, error) {
	word, i := readAlpha(s, 0)
	switch word {
	case "minutely":
		ce.hour, ce.minute = calAny(), calAny()
	case "hourly":
		ce.hour = calAny()
	case "daily": // nothing to do, that is the default
	case "monthly":
		ce.day = calVals(1)
	case "weekly":
		ce.weekdays = 1 // Monday
	case "yearly", "annually":
		ce.month, ce.day = calVals(1), calVals(1)
	case "quarterly":
		ce.month, ce.day = calVals(1, 4, 7, 10), calVals(1)
	case "semiannually":
		ce.month, ce.day = calVals(1, 7), calVals(1)
	default:
		return false, nil
	}
	var err error
	ce.loc, err = parseTimezone(s, i, nil)
	return true, err
}

// parseCalWeekdays reads the weekday list in front of a calendar event, like "Mon,Wed..Fri".
// It returns a bitmask, which is 0 when there are no weekdays at pos.
func parseCalWeekdays(s string, pos int) (uint8, int, error) {
	var mask uint8
	i := pos
	for {
		first, j, ok := parseWeekday(s, i)
		if !ok {
			return mask, i, nil
		}
		last := first
		if strings.HasPrefix(s[j:], "..") {
			if last, j, ok = parseWeekday(s, j+2); !ok {
				return 0, pos, fmt.Errorf("expected weekday after '..' in %q", s)
			}
			if monFirst(last) < monFirst(first) {
				return 0, pos, fmt.Errorf("expected weekday range in order, got %s..%s in %q", first, last, s)
			}
		}
		for bit := monFirst(first); bit <= monFirst(last); bit++ {
			mask |= 1 << uint(bit)
		}
		if j >= len(s) || s[j] != ',' {
			return mask, j, nil
		}
		i = j + 1
	}
}

// parseCalDate fills in the year, month, and day of ce. Dates come as YYYY-MM-DD, YY-MM-DD, or
// MM-DD, which leaves the year at "*". A "~" instead of the "-" in front of the day counts the
// days from the end of the month.
func parseCalDate(s string, pos int, ce *CalendarEvent) (int, error) {
	f1, i, err := parseCalField(s, pos, false)
	if err != nil {
		return pos, err
	}
	if i >= len(s) || s[i] != '-' {
		return pos, fmt.Errorf("expected date (YYYY-MM-DD, YY-MM-DD, or MM-DD), got %q", s)
	}
	f2, i, err := parseCalField(s, i+1, false)
	if err != nil {
		return pos, err
	}

	// with only two fields they are month and day
	year, month, day := calAny(), f1, f2
	if i < len(s) && (s[i] == '-' || s[i] == '~') {
		fromEnd := s[i] == '~'
		day, i, err = parseCalField(s, i+1, false)
		if err != nil {
			return pos, err
		}
		if fromEnd && day.wildcard {
			return pos, fmt.Errorf("expected day after '~' in %q", s)
		}
		day.fromEnd = fromEnd
		for j := range f1.values { // expand 2-digit years
			v := &f1.values[j]
			v.start = int64(expandYear(int(v.start)))
			if v.end >= 0 {
				v.end = int64(expandYear(int(v.end)))
			}
		}
		year, month = f1, f2
	}

	if err := checkCalField(s, year, "year", minCalYear, maxCalYear, 1); err != nil {
		return pos, err
	}
	if err := checkCalField(s, month, "month", 1, 12, 1); err != nil {
		return pos, err
	}
	maxDay := int64(31)
	if day.fromEnd {
		maxDay = 28 // February is the shortest month
	}
	if err := checkCalField(s, day, "day", 1, maxDay, 1); err != nil {
		return pos, err
	}
	ce.year, ce.month, ce.day = year, month, day
	return i, nil
}

// parseCalTime fills in the hour, minute, and second of ce. Times come as HH:MM:SS or HH:MM,
// the seconds default to 0 and may have a fraction.
func parseCalTime(s string, pos int, ce *CalendarEvent) (int, error) {
	hour, i, err := parseCalField(s, pos, false)
	if err != nil {
		return pos, err
	}
	if i >= len(s) || s[i] != ':' {
		return pos, fmt.Errorf("expected time (HH:MM:SS or HH:MM), got %q", s)
	}
	minute, i, err := parseCalField(s, i+1, false)
	if err != nil {
		return pos, err
	}
	second := calVals(0)
	if i < len(s) && s[i] == ':' {
		second, i, err = parseCalField(s, i+1, true) // seconds are read as nanoseconds
		if err != nil {
			return pos, err
		}
	}

	if err := checkCalField(s, hour, "hour", 0, 23, 1); err != nil {
		return pos, err
	}
	if err := checkCalField(s, minute, "minute", 0, 59, 1); err != nil {
		return pos, err
	}
	if err := checkCalField(s, second, "second", 0, 59, int64(Second)); err != nil {
		return pos, err
	}
	ce.hour, ce.minute, ce.second = hour, minute, second
	return i, nil
}

// parseCalField reads a whole field, either "*" or values separated by commas.
func parseCalField(s string, pos int, isSec bool) (calField, int, error) {
	if pos < len(s) && s[pos] == '*' {
		return calAny(), pos + 1, nil
	}
	var values []calValue
	i := pos
	for {
		v, j, err := parseCalValue(s, i, isSec)
		if err != nil {
			return calField{}, pos, err
		}
		values = append(values, v)
		if j >= len(s) || s[j] != ',' {
			return calField{wildcard: false, values: values, fromEnd: false}, j, nil
		}
		i = j + 1
	}
}

// parseCalValue reads one entry of a field. That can be "1", a range like "1..5", a repetition
// like "1/2", or both at once as in "1..5/2".
func parseCalValue(s string, pos int, isSec bool) (calValue, int, error) {
	start, i, err := readCalNum(s, pos, isSec)
	if err != nil {
		return calValue{}, pos, err
	}
	v := calValue{start: start, end: -1, step: 0}
	if strings.HasPrefix(s[i:], "..") {
		v.end, i, err = readCalNum(s, i+2, isSec)
		if err != nil {
			return calValue{}, pos, err
		}
		if v.end < v.start {
			return calValue{}, pos, fmt.Errorf("expected range in order, got %d..%d in %q", v.start, v.end, s)
		}
	}
	if i < len(s) && s[i] == '/' {
		v.step, i, err = readCalNum(s, i+1, isSec)
		if err != nil {
			return calValue{}, pos, err
		}
		if v.step == 0 {
			return calValue{}, pos, fmt.Errorf("expected repetition greater than 0 in %q", s)
		}
	}
	return v, i, nil
}

// calAny is the "*" field.
func calAny() calField {
	return calField{wildcard: true, values: nil, fromEnd: false}
}

// calVals builds a field that matches exactly the given values.
func calVals(nums ...int64) calField {
	values := make([]calValue, len(nums))
	for i, num := range nums {
		values[i] = calValue{start: num, end: -1, step: 0}
	}
	return calField{wildcard: false, values: values, fromEnd: false}
}

// checkCalField makes sure every value in f lies within lo-hi. unit is 1, except for the
// second field where the values are nanoseconds and unit is Second.
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

// Next returns the first time after t that matches the calendar event, t itself does not
// count. The bool is false when there is no such time before the end of 2199, which happens
// for a date in the past. An event without a timezone of its own uses the timezone of t.
func (ce CalendarEvent) Next(t time.Time) (time.Time, bool) {
	loc := ce.loc
	if loc == nil {
		loc = t.Location()
	}

	// walk down from the year to the second, and whenever a component does not match jump
	// to its next matching value and start over, or to the start of the next larger
	// component if nothing matches anymore
	next := t.In(loc).Add(Nanosecond)
	for {
		year, month, day := next.Date()
		hour, minute, sec := next.Clock()
		nsec := int64(sec)*int64(Second) + int64(next.Nanosecond())

		nextYear, ok := ce.year.nextVal(int64(year), maxCalYear)
		if !ok { // nothing left before maxCalYear, which ends the search
			return time.Time{}, false
		}
		if int(nextYear) != year {
			next = time.Date(int(nextYear), time.January, 1, 0, 0, 0, 0, loc)
			continue
		}

		nextMonth, ok := ce.month.nextVal(int64(month), 12)
		if !ok {
			next = time.Date(year+1, time.January, 1, 0, 0, 0, 0, loc)
			continue
		}
		if time.Month(nextMonth) != month {
			next = time.Date(year, time.Month(nextMonth), 1, 0, 0, 0, 0, loc)
			continue
		}

		nextDay, ok := ce.day.nextDay(year, month, day)
		if !ok {
			next = time.Date(year, month+1, 1, 0, 0, 0, 0, loc)
			continue
		}
		if nextDay != day {
			next = time.Date(year, month, nextDay, 0, 0, 0, 0, loc)
			continue
		}

		if ce.weekdays>>uint(monFirst(next.Weekday()))&1 == 0 {
			next = time.Date(year, month, day+1, 0, 0, 0, 0, loc)
			continue
		}

		nextHour, ok := ce.hour.nextVal(int64(hour), 23)
		if !ok {
			next = time.Date(year, month, day+1, 0, 0, 0, 0, loc)
			continue
		}
		if int(nextHour) != hour {
			next = time.Date(year, month, day, int(nextHour), 0, 0, 0, loc)
			continue
		}

		nextMinute, ok := ce.minute.nextVal(int64(minute), 59)
		if !ok {
			next = time.Date(year, month, day, hour+1, 0, 0, 0, loc)
			continue
		}
		if int(nextMinute) != minute {
			next = time.Date(year, month, day, hour, int(nextMinute), 0, 0, loc)
			continue
		}

		nextNsec, ok := ce.second.nextVal(nsec, int64(Minute)-1)
		if !ok {
			next = time.Date(year, month, day, hour, minute+1, 0, 0, loc)
			continue
		}
		if nextNsec != nsec {
			nextSec := int(nextNsec / int64(Second))
			next = time.Date(year, month, day, hour, minute, nextSec, int(nextNsec%int64(Second)), loc)
			continue
		}

		// when daylight saving time ends an hour repeats itself and time.Date picks the
		// earlier offset, which can land before t
		if !next.After(t) {
			next = time.Date(year, month, day, hour, minute+1, 0, 0, loc)
			continue
		}
		return next, true
	}
}

// nextVal is nextCalValue over all values of f. When it finds nothing, the caller has to bump
// the next larger component and try again.
func (f calField) nextVal(from, last int64) (int64, bool) {
	if f.wildcard {
		return from, from <= last
	}
	best := int64(-1)
	for _, v := range f.values {
		if next := nextCalValue(v, from, last); next >= 0 && (best < 0 || next < best) {
			best = next
		}
	}
	return best, best >= 0
}

// nextDay is nextVal for the day field, which has to know how long the month is.
func (f calField) nextDay(year int, month time.Month, day int) (int, bool) {
	lastDay := int64(daysInMonth(year, month))
	if !f.fromEnd {
		next, ok := f.nextVal(int64(day), lastDay)
		return int(next), ok
	}

	// "~" days count backwards, turn them into normal days first
	best := int64(-1)
	for _, v := range f.values {
		flipped := calValue{start: lastDay - v.start + 1, end: -1, step: v.step}
		if v.end >= 0 { // a range flips around, "~1..3" is the last three days
			flipped.start, flipped.end = lastDay-v.end+1, lastDay-v.start+1
		}
		if next := nextCalValue(flipped, int64(day), lastDay); next >= 0 && (best < 0 || next < best) {
			best = next
		}
	}
	return int(best), best >= 0
}

// nextCalValue finds the first value of v that is at least from and at most last, or -1 if
// there is none. last is where the component rolls over, 23 for hours or 59 for minutes.
func nextCalValue(v calValue, from, last int64) int64 {
	step, end := v.step, v.end
	if step == 0 { // no repetition means every value counts
		step = 1
	}
	if end < 0 { // a single value
		end = v.start
		if v.step > 0 { // "5/2" runs until the component rolls over
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
	next := v.start + (from-v.start+step-1)/step*step // first step at or past from
	if next > end {
		return -1
	}
	return next
}

// Lexers and arithmetic helpers shared by the parsers above.

// isDigit reports whether c is an ASCII digit.
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// skipSpaces moves pos past any spaces.
func skipSpaces(s string, pos int) int {
	i := pos
	for i < len(s) && s[i] == ' ' {
		i++
	}
	return i
}

// readNum returns the digits at pos as a number, and where they end.
func readNum(s string, pos int) (int, int, error) {
	i := pos
	for i < len(s) && isDigit(s[i]) {
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

// readFrac parses the digits behind a decimal point and returns them as nanoseconds, so ".5"
// gives 500000000. Anything past the ninth digit is cut off.
func readFrac(s string, pos int) (int, int, error) {
	i := pos
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	if i == pos {
		return 0, pos, fmt.Errorf("expected number in %q", s)
	}
	nsec := 0
	for j := pos; j < pos+9; j++ { // pad or cut to 9 digits
		nsec *= 10
		if j < i {
			nsec += int(s[j] - '0')
		}
	}
	return nsec, i, nil
}

// readBounded is readNum with a range check. name only shows up in the error message.
func readBounded(s string, pos int, name string, lo, hi int) (int, int, error) {
	n, i, err := readNum(s, pos)
	if err != nil {
		return 0, pos, err
	}
	if n < lo || n > hi {
		return 0, pos, fmt.Errorf("expected %s in range %d-%d, got %d in %q", name, lo, hi, n, s)
	}
	return n, i, nil
}

// readCalNum parses a number for a calendar field. isSec marks the second field, where the
// number may have a fraction and comes back as nanoseconds.
func readCalNum(s string, pos int, isSec bool) (int64, int, error) {
	n, i, err := readNum(s, pos)
	if err != nil {
		return 0, pos, err
	}
	if !isSec {
		return int64(n), i, nil
	}
	nsec := int64(n) * int64(Second)
	if i < len(s) && s[i] == '.' {
		var frac int
		frac, i, err = readFrac(s, i+1)
		if err != nil {
			return 0, pos, err
		}
		nsec += int64(frac)
	}
	return nsec, i, nil
}

// readAlpha returns the run of ASCII letters at pos, and where it ends.
func readAlpha(s string, pos int) (string, int) {
	i := pos
	for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z')) {
		i++
	}
	return s[pos:i], i
}

// readWord returns everything at pos up to the next digit or space, and where it ends.
func readWord(s string, pos int) (string, int) {
	i := pos
	for i < len(s) && s[i] != ' ' && !isDigit(s[i]) {
		i++
	}
	return s[pos:i], i
}

// firstSep tells a date from a time. It returns whichever of ':' and '-' comes first in the
// word at pos, or 0 if neither does. A word that does not start with a digit or '*' always
// gives 0, otherwise a timezone like "Etc/GMT-5" would pass for a date.
func firstSep(s string, pos int) byte {
	if pos >= len(s) || (!isDigit(s[pos]) && s[pos] != '*') {
		return 0
	}
	for i := pos; i < len(s) && s[i] != ' '; i++ {
		if s[i] == ':' || s[i] == '-' {
			return s[i]
		}
	}
	return 0
}

// expandYear turns a 2-digit year into a real one. 0-68 becomes 2000-2068, 69-99 becomes
// 1969-1999. Three or more digits pass through untouched. systemd does the same for
// timestamps (but rejects everything before the epoch), while in calendar events it maps 69
// to 2069.
func expandYear(year int) int {
	switch {
	case year >= 100:
		return year
	case year <= 68:
		return year + 2000
	default:
		return year + 1900
	}
}

// daysInMonth returns how many days month has. The year only matters for February.
func daysInMonth(year int, month time.Month) int {
	if month == time.February {
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	}
	if month == time.April || month == time.June || month == time.September || month == time.November {
		return 30
	}
	return 31
}

// monFirst turns Go's Sunday-first weekday into a Monday-first index, so Monday is 0 and
// Sunday is 6.
func monFirst(wd time.Weekday) int {
	return (int(wd) + 6) % 7
}
