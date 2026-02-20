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
//	months, month, M (defined as 30.44 days)
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
//
// The function returns an error if the input is empty, contains invalid numbers,
// unrecognized units, or malformed syntax (e.g. multiple decimal points).
func ParseTimespan(s string) (time.Duration, error) {
	switch s {
	case "":
		return 0, errors.New("expected time span, got empty string")
	case "0":
		return 0, nil
	}

	var total time.Duration

	for i := 0; i < len(s); {
		// skip spaces
		for i < len(s) && s[i] == ' ' {
			i++
		}

		// break if we reached the end
		if i >= len(s) {
			break
		}

		// parse numbers
		foundDot := false
		foundNum := false
		start := i
		for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '.') {
			if s[i] == '.' {
				if foundDot {
					return 0, fmt.Errorf("expected single decimal point, got multiple at position %d in %q", i, s)
				}
				foundDot = true
			} else {
				foundNum = true
			}
			i++
		}
		if !foundNum {
			return 0, fmt.Errorf("expected number, got %q at position %d in %q", string(s[start]), start, s)
		}
		var num float64
		if foundDot { // ParseFloat can parse integers, but it's much slower than Atoi, hence the if-else
			var err error
			num, err = strconv.ParseFloat(s[start:i], 64)
			if err != nil {
				return 0, fmt.Errorf("expected valid number, got %q in %q: %w", s[start:i], s, err)
			}
		} else {
			v, err := strconv.Atoi(s[start:i])
			if err != nil {
				return 0, fmt.Errorf("expected valid number, got %q in %q: %w", s[start:i], s, err)
			}
			num = float64(v)
		}

		// skip spaces again
		for i < len(s) && s[i] == ' ' {
			i++
		}

		// parse unit
		var unit time.Duration
		start = i
		for i < len(s) && s[i] != ' ' && (s[i] < '0' || s[i] > '9') {
			i++
		}
		if start == i {
			unit = Second // no unit specified, default to seconds
		} else {
			// switch was ca. 20% faster than a map in my tests
			switch s[start:i] {
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
				return 0, fmt.Errorf("expected valid unit, got %q in %q", s[start:i], s)
			}
		}

		total += time.Duration(num * float64(unit))
	}

	return total, nil
}
