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

package systemdtime_test

import (
	"fmt"
	"testing"
	"time"

	systemdtime "gitlab.com/allddd/go-systemd-time"
)

var (
	tzLondon, _  = time.LoadLocation("Europe/London")
	tzNewYork, _ = time.LoadLocation("America/New_York")
	tzSydney, _  = time.LoadLocation("Australia/Sydney")
	tzTokyo, _   = time.LoadLocation("Asia/Tokyo")
)

func TestParseTimespan(t *testing.T) {
	cases := []struct {
		input     string
		expect    time.Duration
		expectErr bool
	}{
		// simple
		{"100ns", 100 * systemdtime.Nanosecond, false},
		{"100nsec", 100 * systemdtime.Nanosecond, false},
		{"200us", 200 * systemdtime.Microsecond, false},
		{"200usec", 200 * systemdtime.Microsecond, false},
		{"200µs", 200 * systemdtime.Microsecond, false},
		{"200μs", 200 * systemdtime.Microsecond, false},
		{"500ms", 500 * systemdtime.Millisecond, false},
		{"500msec", 500 * systemdtime.Millisecond, false},
		{"30s", 30 * systemdtime.Second, false},
		{"30sec", 30 * systemdtime.Second, false},
		{"30second", 30 * systemdtime.Second, false},
		{"30seconds", 30 * systemdtime.Second, false},
		{"5m", 5 * systemdtime.Minute, false},
		{"5min", 5 * systemdtime.Minute, false},
		{"5minute", 5 * systemdtime.Minute, false},
		{"5minutes", 5 * systemdtime.Minute, false},
		{"3h", 3 * systemdtime.Hour, false},
		{"3hr", 3 * systemdtime.Hour, false},
		{"3hour", 3 * systemdtime.Hour, false},
		{"3hours", 3 * systemdtime.Hour, false},
		{"7d", 7 * systemdtime.Day, false},
		{"7day", 7 * systemdtime.Day, false},
		{"7days", 7 * systemdtime.Day, false},
		{"2w", 2 * systemdtime.Week, false},
		{"2week", 2 * systemdtime.Week, false},
		{"2weeks", 2 * systemdtime.Week, false},
		{"3M", 3 * systemdtime.Month, false},
		{"3month", 3 * systemdtime.Month, false},
		{"3months", 3 * systemdtime.Month, false},
		{"2y", 2 * systemdtime.Year, false},
		{"2year", 2 * systemdtime.Year, false},
		{"2years", 2 * systemdtime.Year, false},
		{"weeks", 0, true},
		{"5xyz", 0, true},
		{"5H", 0, true},
		{"5S", 0, true},
		{"5D", 0, true},
		{"5W", 0, true},
		{"5Months", 0, true},
		{"5Years", 0, true},
		// decimal
		{"1.5sec", time.Duration(1.5 * float64(systemdtime.Second)), false},
		{"1.5days", time.Duration(1.5 * float64(systemdtime.Day)), false},
		{"2.5hr", time.Duration(2.5 * float64(systemdtime.Hour)), false},
		{"0.5week", time.Duration(0.5 * float64(systemdtime.Week)), false},
		{".5s", time.Duration(0.5 * float64(systemdtime.Second)), false},
		{".", 0, true},
		{"1.", 0, true},
		{"1.2.3days", 0, true},
		// complex
		{"3 days 12hours", 3*systemdtime.Day + 12*systemdtime.Hour, false},
		{"1year 12M", 1*systemdtime.Year + 12*systemdtime.Month, false},
		{"55sec500msec", 55*systemdtime.Second + 500*systemdtime.Millisecond, false},
		{"300ms20seconds 5d", 300*systemdtime.Millisecond + 20*systemdtime.Second + 5*systemdtime.Day, false},
		{"2weeks3day", 2*systemdtime.Week + 3*systemdtime.Day, false},
		{"1d 2 hr 30s", 1*systemdtime.Day + 2*systemdtime.Hour + 30*systemdtime.Second, false},
		{"5min10sec500 ms", 5*systemdtime.Minute + 10*systemdtime.Second + 500*systemdtime.Millisecond, false},
		{"1w 2days", 1*systemdtime.Week + 2*systemdtime.Day, false},
		{"2.5d 1.5hours", time.Duration(2.5*float64(systemdtime.Day)) + time.Duration(1.5*float64(systemdtime.Hour)), false},
		{"1.5h 30min", time.Duration(1.5*float64(systemdtime.Hour)) + 30*systemdtime.Minute, false},
		{"2.5 d 12h 30min", time.Duration(2.5*float64(systemdtime.Day)) + 12*systemdtime.Hour + 30*systemdtime.Minute, false},
		// default unit
		{"60", 60 * systemdtime.Second, false},
		{"1.5", time.Duration(1.5 * float64(systemdtime.Second)), false},
		{"60 5min", 60*systemdtime.Second + 5*systemdtime.Minute, false},
		// zero
		{"0", 0, false},
		{"0s", 0, false},
		{"0h", 0, false},
		{"0y", 0, false},
		// spacing
		{" 10min", 10 * systemdtime.Minute, false},
		{"5sec ", 5 * systemdtime.Second, false},
		{" 5days  ", 5 * systemdtime.Day, false},
		{"2w    10s", 2*systemdtime.Week + 10*systemdtime.Second, false},
		{"  ", 0, true},
		// error
		{"", 0, true},
		{"hello", 0, true},
	}
	for _, tc := range cases {
		got, err := systemdtime.ParseTimespan(tc.input)
		if tc.expectErr {
			if err == nil {
				t.Errorf("%q: expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.expect {
			t.Errorf("%q: expected %v, got %v", tc.input, tc.expect, got)
		}
	}
}

func BenchmarkParseTimespan(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{"simple", "2h"},
		{"decimal", "2.5h"},
		{"complex", "1y 12month 2w3d 5.5h 10min15sec"},
	}
	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				systemdtime.ParseTimespan(bc.input)
			}
		})
	}
}

func ExampleParseTimespan() {
	ts := "2h30min40seconds"
	d, _ := systemdtime.ParseTimespan(ts)
	fmt.Printf("%q is %.0f seconds.\n", ts, d.Seconds())
	// Output:
	// "2h30min40seconds" is 9040 seconds.
}

func TestParseTimestamp(t *testing.T) {
	now := time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC)
	cases := []struct {
		input     string
		expect    time.Time
		expectErr bool
	}{
		// token
		{"now", now, false},
		{"today", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"yesterday", time.Date(2009, 11, 9, 0, 0, 0, 0, time.UTC), false},
		{"tomorrow", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{" now", time.Time{}, true},
		{"now ", time.Time{}, true},
		{"now UTC", time.Time{}, true},
		{"today tomorrow", time.Time{}, true},
		{"tomorrow today", time.Time{}, true},
		// date
		{"2009-11-10", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"09-11-10", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"00-01-01", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"68-01-01", time.Date(2068, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"69-01-01", time.Date(1969, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"99-12-31", time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC), false},
		{"11-23", time.Time{}, true},
		{"2009-00-01", time.Time{}, true},
		{"2009-13-01", time.Time{}, true},
		{"2009-01-00", time.Time{}, true},
		{"2009-11-32", time.Time{}, true},
		{"2009-11-10 2009-11-10", time.Time{}, true},
		// time
		{"18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"18:15", time.Date(2009, 11, 10, 18, 15, 0, 0, time.UTC), false},
		{"0:0:0", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"0:0", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"18", time.Time{}, true},
		{"25:00:00", time.Time{}, true},
		{"18:60:00", time.Time{}, true},
		{"18:15:60", time.Time{}, true},
		{"18:60", time.Time{}, true},
		{"18:15:22 18:15:22", time.Time{}, true},
		// date and time
		{"2009-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10 18:15", time.Date(2009, 11, 10, 18, 15, 0, 0, time.UTC), false},
		{"09-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10 18 :15:22", time.Time{}, true},
		{"2009-11-10 18", time.Time{}, true},
		// weekday
		{"Mon 2009-11-09", time.Date(2009, 11, 9, 0, 0, 0, 0, time.UTC), false},
		{"Tue 2009-11-10", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"Wed 2009-11-11", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"Thu 2009-11-12", time.Date(2009, 11, 12, 0, 0, 0, 0, time.UTC), false},
		{"Fri 2009-11-13", time.Date(2009, 11, 13, 0, 0, 0, 0, time.UTC), false},
		{"Sat 2009-11-14", time.Date(2009, 11, 14, 0, 0, 0, 0, time.UTC), false},
		{"Sun 2009-11-15", time.Date(2009, 11, 15, 0, 0, 0, 0, time.UTC), false},
		{"Monday 2009-11-09", time.Date(2009, 11, 9, 0, 0, 0, 0, time.UTC), false},
		{"Tuesday 2009-11-10", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"Wednesday 2009-11-11", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"Thursday 2009-11-12", time.Date(2009, 11, 12, 0, 0, 0, 0, time.UTC), false},
		{"Friday 2009-11-13", time.Date(2009, 11, 13, 0, 0, 0, 0, time.UTC), false},
		{"Saturday 2009-11-14", time.Date(2009, 11, 14, 0, 0, 0, 0, time.UTC), false},
		{"Sunday 2009-11-15", time.Date(2009, 11, 15, 0, 0, 0, 0, time.UTC), false},
		{"Tue 2009-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"Tuesday 2009-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"tue 2009-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"TUE 2009-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"Mon 2009-11-10 18:15:22", time.Time{}, true},
		{"Thursday 2009-11-10", time.Time{}, true},
		{"Friday", time.Time{}, true},
		{"Fri 18:15:22", time.Time{}, true},
		{"Tue Tue 2009-11-10", time.Time{}, true},
		{"Tue,2009-11-10", time.Time{}, true},
		{"Tue-2009-11-10", time.Time{}, true},
		// fractional
		{"18:15:22.5", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.UTC), false},
		{"2009-11-10 18:15:22.5", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.UTC), false},
		{"2009-11-10 18:15:22.123456", time.Date(2009, 11, 10, 18, 15, 22, 123456000, time.UTC), false},
		{"Tue 2009-11-10 18:15:22.5", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.UTC), false},
		{"2009-11-10 18:15:22.5 UTC", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.UTC), false},
		{"2009-11-10 18:15:22.5Z", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.UTC), false},
		{"2009-11-10 18:15:22.5+05:30", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.FixedZone("", 5*3600+30*60)), false},
		{"2009-11-10 18:15:22.", time.Time{}, true},
		// timezone
		{"today UTC", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"yesterday UTC", time.Date(2009, 11, 9, 0, 0, 0, 0, time.UTC), false},
		{"tomorrow UTC", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"2009-11-10 UTC", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"2009-11-10 +01:00", time.Date(2009, 11, 10, 0, 0, 0, 0, time.FixedZone("", 1*3600)), false},
		{"2009-11-10Z", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"2009-11-10+01:00", time.Date(2009, 11, 10, 0, 0, 0, 0, time.FixedZone("", 1*3600)), false},
		{"Tue 2009-11-10 UTC", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"Tue 2009-11-10Z", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"Tue 2009-11-10+01:00", time.Date(2009, 11, 10, 0, 0, 0, 0, time.FixedZone("", 1*3600)), false},
		{"18:15:22 UTC", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"11:12 UTC", time.Date(2009, 11, 10, 11, 12, 0, 0, time.UTC), false},
		{"18:15:22 +05:30", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", 5*3600+30*60)), false},
		{"18:15:22Z", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10 18:15:22 UTC", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10 18:15:22 Z", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10 22:02:15Z", time.Date(2009, 11, 10, 22, 2, 15, 0, time.UTC), false},
		{"2009-11-10 18:15:22+05:30", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", 5*3600+30*60)), false},
		{"2009-11-10 18:15:22-05:30", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", -5*3600-30*60)), false},
		{"2009-11-10 18:15:22 +05", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", 5*3600)), false},
		{"2009-11-10 18:15:22 -05", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", -5*3600)), false},
		{"2009-11-10 18:15:22 +0530", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", 5*3600+30*60)), false},
		{"2009-11-10 18:15:22 -0530", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", -5*3600-30*60)), false},
		{"2009-11-10 18:15:22 America/New_York", time.Date(2009, 11, 10, 18, 15, 22, 0, tzNewYork), false},
		{"2009-11-10 18:15:22 Europe/London", time.Date(2009, 11, 10, 18, 15, 22, 0, tzLondon), false},
		{"2009-11-10 18:15:22 Asia/Tokyo", time.Date(2009, 11, 10, 18, 15, 22, 0, tzTokyo), false},
		{"2009-11-10 18:15:22 Australia/Sydney", time.Date(2009, 11, 10, 18, 15, 22, 0, tzSydney), false},
		{"2009-11-10 18:15:22 +05:60", time.Time{}, true},
		{"2009-11-10 18:15:22 +99:00", time.Time{}, true},
		{"2009-11-10 18:15:22 Not/TZ", time.Time{}, true},
		{"2009-11-10 18:15:22 UTC extra", time.Time{}, true},
		{"UTC", time.Time{}, true},
		{"Tue UTC", time.Time{}, true},
		// rfc3339
		{"2009-11-10T23:02:15", time.Date(2009, 11, 10, 23, 2, 15, 0, time.UTC), false},
		{"2009-11-10T22:02:15Z", time.Date(2009, 11, 10, 22, 2, 15, 0, time.UTC), false},
		{"2009-11-10T18:15:22+00:00", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10T23:02:15+01:00", time.Date(2009, 11, 10, 23, 2, 15, 0, time.FixedZone("", 1*3600)), false},
		{"2009-11-10T18:15:22-05:30", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", -5*3600-30*60)), false},
		{"2009-11-10T11:12+02:00", time.Date(2009, 11, 10, 11, 12, 0, 0, time.FixedZone("", 2*3600)), false},
		{"2009-11-10T23:02:15 UTC", time.Date(2009, 11, 10, 23, 2, 15, 0, time.UTC), false},
		{"2009-11-10T18:15:22.654321+01:00", time.Date(2009, 11, 10, 18, 15, 22, 654321000, time.FixedZone("", 1*3600)), false},
		{"Tue 2009-11-10T11:12:13+01:00", time.Date(2009, 11, 10, 11, 12, 13, 0, time.FixedZone("", 1*3600)), false},
		{"Tue 2009-11-10T18:15:22Z", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"Tue 2009-11-10T11:12:13.654321+01:00", time.Date(2009, 11, 10, 11, 12, 13, 654321000, time.FixedZone("", 1*3600)), false},
		{"09-11-10T18:15:22", time.Time{}, true},
		{"09-11-10T18:15:22Z", time.Time{}, true},
		{"2009-11-10T", time.Time{}, true},
		{"2009-11-10TZ", time.Time{}, true},
		{"2009-11-10T18", time.Time{}, true},
		// relative
		{"+3h30min", time.Date(2009, 11, 11, 2, 30, 0, 0, time.UTC), false},
		{"-5s", time.Date(2009, 11, 10, 22, 59, 55, 0, time.UTC), false},
		{"11min ago", time.Date(2009, 11, 10, 22, 49, 0, 0, time.UTC), false},
		{"1h left", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"+", time.Time{}, true},
		{"-", time.Time{}, true},
		{"+abc", time.Time{}, true},
		{"-abc", time.Time{}, true},
		{"abc ago", time.Time{}, true},
		{"abc left", time.Time{}, true},
		{"+5s -5s", time.Time{}, true},
		{"+5s UTC", time.Time{}, true},
		// unix
		{"@0", time.Unix(0, 0), false},
		{"@0.5", time.Unix(0, 500000000), false},
		{"@1395716396", time.Unix(1395716396, 0), false},
		{"@1395716396.11111", time.Unix(1395716396, 111110000), false},
		{"@1395716396.654321", time.Unix(1395716396, 654321000), false},
		{" @0", time.Time{}, true},
		{"@", time.Time{}, true},
		{"@abc", time.Time{}, true},
		{"@1234 @5678", time.Time{}, true},
		{"@1.", time.Time{}, true},
		{"@1.5abc", time.Time{}, true},
		// error
		{"", time.Time{}, true},
		{"invalid", time.Time{}, true},
	}
	for _, tc := range cases {
		got, err := systemdtime.ParseTimestamp(tc.input, now)
		if tc.expectErr {
			if err == nil {
				t.Errorf("%q: expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.input, err)
			continue
		}
		if !got.Equal(tc.expect) {
			t.Errorf("%q: expected %v, got %v", tc.input, tc.expect, got)
		}
	}
}

func BenchmarkParseTimestamp(b *testing.B) {
	now := time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		input string
	}{
		{"token", "today"},
		{"date", "2009-11-10"},
		{"time", "18:15:22"},
		{"datetime", "2009-11-10 18:15:22"},
		{"weekday", "Tue 2009-11-10 18:15:22"},
		{"fractional", "2009-11-10 18:15:22.654321"},
		{"timezone", "2009-11-10 18:15:22 America/New_York"},
		{"rfc3339", "2009-11-10T18:15:22+01:00"},
		{"relative", "+3h30min"},
		{"unix", "@1395716396"},
	}
	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				systemdtime.ParseTimestamp(bc.input, now)
			}
		})
	}
}

func ExampleParseTimestamp() {
	ts := "2009-11-10 23:00:00 UTC"
	t, _ := systemdtime.ParseTimestamp(ts)
	fmt.Printf("%q is a %s.\n", ts, t.Weekday())
	// Output:
	// "2009-11-10 23:00:00 UTC" is a Tuesday.
}

func TestParseCalendarEvent(t *testing.T) {
	cases := []struct {
		input     string
		expectErr bool
	}{
		// shorthand
		{"minutely", false},
		{"hourly", false},
		{"daily", false},
		{"monthly", false},
		{"weekly", false},
		{"yearly", false},
		{"annually", false},
		{"quarterly", false},
		{"semiannually", false},
		{" daily", true},
		{"daily extra", true},
		// date
		{"*-*-*", false},
		{"*-*-01", false},
		{"*-01-01", false},
		{"2003-03-05", false},
		{"*-*", false},
		{"03-05", false},
		{"12-*-*", false},
		{"1970-01-01", false},
		{"2199-12-31", false},
		{"*", true},
		{"1969-12-31", true},
		{"69-01-01", true},
		{"2200-01-01", true},
		{"*-00-01", true},
		{"*-13-01", true},
		{"*-*-00", true},
		{"*-*-32", true},
		{"13-01", true},
		{"10-32", true},
		// time
		{"08:05:40", false},
		{"05:40", false},
		{"0:0:0", false},
		{"*:*:*", false},
		{"*:*:00", false},
		{"*:00:00", false},
		{"05:", true},
		{"24:00:00", true},
		{"23:60:00", true},
		{"23:59:60", true},
		// date and time
		{"*-*-* 00:00:00", false},
		{"*-*-* *:*:00", false},
		{"*-*-* *:00:00", false},
		{"*-*-7 0:0:0", false},
		{"2003-03-05 05:40", false},
		{"2003-03-05 05:40:00", false},
		{"2009-11-10T18:15:22", true},
		{"  *-*-* 00:00:00", true},
		// weekday
		{"Mon", false},
		{"Monday", false},
		{"tuesday", false},
		{"WEDNESDAY", false},
		{"Mon,", false},
		{"Mon,Fri", false},
		{"MONDAY,friday", false},
		{"Mon..Wed", false},
		{"Mon..Sun", false},
		{"Mon..Wed,Fri", false},
		{"Wed..Wed,Wed", false},
		{"Sat,Thu,Mon..Wed,Sat..Sun", false},
		{"Wed *-1", false},
		{"Wed *-1,3", false},
		{"Wed, 17:48", false},
		{"Mon,Wed,Fri 17:48", false},
		{"monday *-12-* 17:00", false},
		{"Mon,Sun 12-*-* 2,1:23", false},
		{"Mon,Fri *-*-3,1,2 *:30:45", false},
		{"mon,fri *-1/2-1,3 *:30:45", false},
		{"Thu,Fri 2012-*-1,5 11:12:13", false},
		{"Wed..Sat,Tue 12-10-15 1:2:3", false},
		{"Mon..", true},
		{"Fri..Mon", true},
		{"Mon..Wed..Fri", true},
		{"Mon..Sat..Sun", true},
		// last day
		{"*-*~01", false},
		{"*-*~28", false},
		{"*-*~1,5", false},
		{"*-*~1..3", false},
		{"*-*~3/2", false},
		{"*-02~03", false},
		{"*-05~07/1", false},
		{"*-*~*", true},
		{"*-*~0", true},
		{"*-*~29", true},
		{"*-*~5..2", true},
		// range and repetition
		{"*-*-01,15", false},
		{"*-1/2-1,3", false},
		{"2003-02..04-05", false},
		{"2020..2025-*-*", false},
		{"*:2/3", false},
		{"*:0/30", false},
		{"*:*:00/5", false},
		{"*:0/15:00", false},
		{"12,14,13,12:20,10,30", false},
		{"12..14:10,20,30", false},
		{"*-,-*", true},
		{"1,-01-01", true},
		{"12..:00", true},
		{"12..10:00", true},
		{"*:0/:00", true},
		{"*:*:00/0", true},
		// fractional
		{"05:40:23.420000", false},
		{"05:40:23.4200004/3.1700005", false},
		{"1.5-01", true},
		{"1.5:00", true},
		// timezone
		{"daily UTC", false},
		{"hourly Europe/London", false},
		{"weekly Pacific/Auckland", false},
		{"minutely +05:30", false},
		{"2003-03-05 05:40 UTC", false},
		{"2003-03-05 05:40 -0500", false},
		{"*-*-* 00:00:00 UTC", false},
		{"*-*-* 00:00:00 Asia/Tokyo", false},
		{"*-*-* 12:00:00 +05:30", false},
		{"*-*-* 00:00:00 Not/TZ", true},
		{"*-*-* 00:00:00 extra", true},
		{"*-*-* 00:00:00 UTC extra", true},
		{"daily UTC extra", true},
		// error
		{"", true},
		{"invalid", true},
	}
	for _, tc := range cases {
		_, err := systemdtime.ParseCalendarEvent(tc.input)
		if tc.expectErr {
			if err == nil {
				t.Errorf("%q: expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.input, err)
		}
	}
}

func BenchmarkParseCalendarEvent(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{"shorthand", "daily"},
		{"date", "2003-03-05"},
		{"time", "08:05:40"},
		{"datetime", "*-*-* 00:00:00"},
		{"weekday", "Mon,Fri"},
		{"lastday", "*-*~1"},
		{"fractional", "05:40:23.4200004/3.1700005"},
		{"timezone", "Mon,Fri *-*-1,15 09:00:00 America/New_York"},
	}
	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				systemdtime.ParseCalendarEvent(bc.input)
			}
		})
	}
}

func ExampleParseCalendarEvent() {
	ev := "Mon *-*-* 09:00:00"
	ce, _ := systemdtime.ParseCalendarEvent(ev)
	next, _ := ce.Next(time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC))
	fmt.Printf("%q elapses next on %s.\n", ev, next.Format("Mon 2006-01-02 15:04:05"))
	// Output:
	// "Mon *-*-* 09:00:00" elapses next on Mon 2009-11-16 09:00:00.
}

func TestCalendarEventNext(t *testing.T) {
	now := time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC)
	cases := []struct {
		input       string
		expect      time.Time
		expectNever bool
	}{
		// shorthand
		{"minutely", time.Date(2009, 11, 10, 23, 1, 0, 0, time.UTC), false},
		{"hourly", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"daily", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"monthly", time.Date(2009, 12, 1, 0, 0, 0, 0, time.UTC), false},
		{"weekly", time.Date(2009, 11, 16, 0, 0, 0, 0, time.UTC), false},
		{"yearly", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"annually", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"quarterly", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"semiannually", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), false},
		// date
		{"*-*-11", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"*-*-10", time.Date(2009, 12, 10, 0, 0, 0, 0, time.UTC), false},
		{"2009-11-11", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"2150-01-01", time.Date(2150, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"2009-11-10", time.Time{}, true},
		{"*-02-30", time.Time{}, true},
		// time
		{"08:05:40", time.Date(2009, 11, 11, 8, 5, 40, 0, time.UTC), false},
		{"23:30:00", time.Date(2009, 11, 10, 23, 30, 0, 0, time.UTC), false},
		{"23:00:00", time.Date(2009, 11, 11, 23, 0, 0, 0, time.UTC), false},
		{"22:30:00", time.Date(2009, 11, 11, 22, 30, 0, 0, time.UTC), false},
		{"*:30:00", time.Date(2009, 11, 10, 23, 30, 0, 0, time.UTC), false},
		{"*:00:00", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		// weekday
		{"Mon", time.Date(2009, 11, 16, 0, 0, 0, 0, time.UTC), false},
		{"Tue", time.Date(2009, 11, 17, 0, 0, 0, 0, time.UTC), false},
		{"Mon,Fri", time.Date(2009, 11, 13, 0, 0, 0, 0, time.UTC), false},
		{"Mon..Wed", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"Sat,Sun 08:05:40", time.Date(2009, 11, 14, 8, 5, 40, 0, time.UTC), false},
		{"monday *-12-* 17:00", time.Date(2009, 12, 7, 17, 0, 0, 0, time.UTC), false},
		{"Wed *-1", time.Date(2010, 9, 1, 0, 0, 0, 0, time.UTC), false},
		{"Wed..Sat,Tue 12-10-15 1:2:3", time.Time{}, true},
		// last day
		{"*-11~1", time.Date(2009, 11, 30, 0, 0, 0, 0, time.UTC), false},
		{"*-11~7/1", time.Date(2009, 11, 24, 0, 0, 0, 0, time.UTC), false},
		{"*-02~03", time.Date(2010, 2, 26, 0, 0, 0, 0, time.UTC), false},
		{"*-*~1..3 06:00", time.Date(2009, 11, 28, 6, 0, 0, 0, time.UTC), false},
		{"Mon *-05~07/1", time.Date(2010, 5, 31, 0, 0, 0, 0, time.UTC), false},
		// range and repetition
		{"2020..2025-*-*", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"*-02-25..31", time.Date(2010, 2, 25, 0, 0, 0, 0, time.UTC), false},
		{"12-*-* 2,1:23", time.Date(2012, 1, 1, 1, 23, 0, 0, time.UTC), false},
		{"12,14,13,12:20,10,30", time.Date(2009, 11, 11, 12, 10, 0, 0, time.UTC), false},
		{"12..14:10,20,30", time.Date(2009, 11, 11, 12, 10, 0, 0, time.UTC), false},
		{"0/6:00:00", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"*:0/15:00", time.Date(2009, 11, 10, 23, 15, 0, 0, time.UTC), false},
		{"*:2/3", time.Date(2009, 11, 10, 23, 2, 0, 0, time.UTC), false},
		{"*:0/30", time.Date(2009, 11, 10, 23, 30, 0, 0, time.UTC), false},
		{"*:*:00/5", time.Date(2009, 11, 10, 23, 0, 5, 0, time.UTC), false},
		{"mon,fri *-1/2-1,3 *:30:45", time.Date(2010, 1, 1, 0, 30, 45, 0, time.UTC), false},
		{"Thu,Fri 2012-*-1,5 11:12:13", time.Date(2012, 1, 5, 11, 12, 13, 0, time.UTC), false},
		// fractional
		{"*:*:00.5", time.Date(2009, 11, 10, 23, 0, 0, 500000000, time.UTC), false},
		{"05:40:23.4200004/3.1700005", time.Date(2009, 11, 11, 5, 40, 23, 420000400, time.UTC), false},
		// timezone
		{"daily UTC", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"daily Australia/Sydney", time.Date(2009, 11, 12, 0, 0, 0, 0, tzSydney), false},
		{"*-*-* 00:00:00 Asia/Tokyo", time.Date(2009, 11, 12, 0, 0, 0, 0, tzTokyo), false},
		{"*-*-* 12:00:00 +05:30", time.Date(2009, 11, 11, 12, 0, 0, 0, time.FixedZone("", 5*3600+30*60)), false},
	}
	for _, tc := range cases {
		ce, err := systemdtime.ParseCalendarEvent(tc.input)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.input, err)
			continue
		}
		got, ok := ce.Next(now)
		if tc.expectNever {
			if ok {
				t.Errorf("%q: expected never, got %v", tc.input, got)
			}
			continue
		}
		if !ok {
			t.Errorf("%q: expected %v, got never", tc.input, tc.expect)
			continue
		}
		if !got.Equal(tc.expect) {
			t.Errorf("%q: expected %v, got %v", tc.input, tc.expect, got)
		}
	}
}

func BenchmarkCalendarEventNext(b *testing.B) {
	now := time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		input string
	}{
		{"shorthand", "daily"},
		{"date", "*-*-11"},
		{"time", "08:05:40"},
		{"weekday", "Mon,Fri *-*-1,15 09:00:00"},
		{"lastday", "*-*~1"},
		{"repetition", "*:0/15:00"},
		{"fractional", "*:*:00.5"},
		{"timezone", "*-*-* 00:00:00 Asia/Tokyo"},
	}
	for _, bc := range cases {
		ce, _ := systemdtime.ParseCalendarEvent(bc.input)
		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				ce.Next(now)
			}
		})
	}
}
