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
		// decimal
		{"1.5sec", 1500 * systemdtime.Millisecond, false},
		{"1.5days", time.Duration(1.5 * float64(systemdtime.Day)), false},
		{"2.5hr", 2*systemdtime.Hour + 30*systemdtime.Minute, false},
		{"0.5week", time.Duration(0.5 * float64(systemdtime.Week)), false},
		// complex
		{"3 days 12hours", 3*systemdtime.Day + 12*systemdtime.Hour, false},
		{"1year 12M", systemdtime.Year + 12*systemdtime.Month, false},
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
		{"1.5", 1500 * systemdtime.Millisecond, false},
		{"60 5min", 60*systemdtime.Second + 5*systemdtime.Minute, false},
		// zero
		{"0", 0, false},
		{"0s", 0, false},
		{"0h", 0, false},
		{"0y", 0, false},
		// error
		{"", 0, true},
		{"  ", 0, true},
		{"hello", 0, true},
		{"weeks", 0, true},
		{"5xyz", 0, true},
		{"abc123min", 0, true},
		{".", 0, true},
		{"1.", 0, true},
		{"1.2.3days", 0, true},
		{"5H", 0, true},
		{"5S", 0, true},
		{"5D", 0, true},
		{"5W", 0, true},
		{"5Months", 0, true},
		{"5Years", 0, true},
		// edge
		{" 10min", 10 * systemdtime.Minute, false},
		{"5sec ", 5 * systemdtime.Second, false},
		{" 5days  ", 5 * systemdtime.Day, false},
		{"2w    10s", 2*systemdtime.Week + 10*systemdtime.Second, false},
		{".5s", 500 * systemdtime.Millisecond, false},
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
	fmt.Printf("There are %.0f seconds in %q.\n", d.Seconds(), ts)
	// Output:
	// There are 9040 seconds in "2h30min40seconds".
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
		{"today UTC", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"yesterday UTC", time.Date(2009, 11, 9, 0, 0, 0, 0, time.UTC), false},
		{"tomorrow UTC", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
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
		{"23", time.Time{}, true},
		{"2009-13-01", time.Time{}, true},
		{"2009-00-01", time.Time{}, true},
		{"2009-01-00", time.Time{}, true},
		{"2009-11-32", time.Time{}, true},
		// time
		{"18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"18:15", time.Date(2009, 11, 10, 18, 15, 0, 0, time.UTC), false},
		{"11:12", time.Date(2009, 11, 10, 11, 12, 0, 0, time.UTC), false},
		{"0:0:0", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"0:0", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"25:00:00", time.Time{}, true},
		{"18:60:00", time.Time{}, true},
		{"18:60", time.Time{}, true},
		// date&time
		{"2009-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10 18:15", time.Date(2009, 11, 10, 18, 15, 0, 0, time.UTC), false},
		{"09-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10 25:00:00", time.Time{}, true},
		{"2009-11-10 18:60:00", time.Time{}, true},
		{"2009-11-10 18:15:60", time.Time{}, true},
		{"2009-11-10 18 :15:22", time.Time{}, true},
		{"2009-11-10 2009-11-10", time.Time{}, true},
		{"18:15:22 18:15:22", time.Time{}, true},
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
		{"Tuesday 2009-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"Saturday 2009-11-14", time.Date(2009, 11, 14, 0, 0, 0, 0, time.UTC), false},
		{"Sunday 2009-11-15", time.Date(2009, 11, 15, 0, 0, 0, 0, time.UTC), false},
		{"tue 2009-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"TUE 2009-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"Tue 2009-11-10 11:12:13", time.Date(2009, 11, 10, 11, 12, 13, 0, time.UTC), false},
		{"Tue 2009-11-10 18:15:22", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"Tue 2009-11-10 UTC", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"Mon 2009-11-10 18:15:22", time.Time{}, true},
		{"Thursday 2009-11-10", time.Time{}, true},
		{"Friday", time.Time{}, true},
		{"Fri 18:15:22", time.Time{}, true},
		{"Tue Tue 2009-11-10", time.Time{}, true},
		// fractional
		{"2009-11-10 18:15:22.5", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.UTC), false},
		{"2009-11-10 18:15:22.123456", time.Date(2009, 11, 10, 18, 15, 22, 123456000, time.UTC), false},
		{"2009-11-10 18:15:22.5 UTC", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.UTC), false},
		{"2009-11-10 18:15:22.5Z", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.UTC), false},
		{"2009-11-10 18:15:22.5+05:30", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.FixedZone("", 5*3600+30*60)), false},
		{"18:15:22.5 UTC", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.UTC), false},
		{"18:15:22.5Z", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.UTC), false},
		{"Tue 2009-11-10 18:15:22.5", time.Date(2009, 11, 10, 18, 15, 22, 500000000, time.UTC), false},
		{"2009-11-10 18:15:22.", time.Time{}, true},
		// timezone
		{"2009-11-10 UTC", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"2009-11-10 +01:00", time.Date(2009, 11, 10, 0, 0, 0, 0, time.FixedZone("", 3600)), false},
		{"18:15:22 UTC", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"11:12 UTC", time.Date(2009, 11, 10, 11, 12, 0, 0, time.UTC), false},
		{"18:15:22 +05:30", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", 5*3600+30*60)), false},
		{"18:15:22Z", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10 18:15:22 UTC", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10 18:15:22 Z", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10 18:15:22+05:30", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", 5*3600+30*60)), false},
		{"2009-11-10 18:15:22-05:00", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", -5*3600)), false},
		{"2009-11-10 18:15:22 +05", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", 5*3600)), false},
		{"2009-11-10 18:15:22 -05", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", -5*3600)), false},
		{"2009-11-10 18:15:22 +0530", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", 5*3600+30*60)), false},
		{"2009-11-10 18:15:22 -0530", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", -5*3600-30*60)), false},
		{"2009-11-10 18:15:22 America/New_York", time.Date(2009, 11, 10, 18, 15, 22, 0, tzNewYork), false},
		{"2009-11-10 18:15:22 Europe/London", time.Date(2009, 11, 10, 18, 15, 22, 0, tzLondon), false},
		{"2009-11-10 18:15:22 Asia/Tokyo", time.Date(2009, 11, 10, 18, 15, 22, 0, tzTokyo), false},
		{"2009-11-10 18:15:22 Australia/Sydney", time.Date(2009, 11, 10, 18, 15, 22, 0, tzSydney), false},
		{"2009-11-10 22:02:15Z", time.Date(2009, 11, 10, 22, 2, 15, 0, time.UTC), false},
		{"2009-11-10Z", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"2009-11-10+01:00", time.Date(2009, 11, 10, 0, 0, 0, 0, time.FixedZone("", 3600)), false},
		{"Tue 2009-11-10Z", time.Date(2009, 11, 10, 0, 0, 0, 0, time.UTC), false},
		{"Tue 2009-11-10+01:00", time.Date(2009, 11, 10, 0, 0, 0, 0, time.FixedZone("", 3600)), false},
		{"2009-11-10 18:15:22 +05:60", time.Time{}, true},
		{"2009-11-10 18:15:22 +99:00", time.Time{}, true},
		{"2009-11-10 18:15:22 Not/TZ", time.Time{}, true},
		// rfc3339
		{"2009-11-10T23:02:15", time.Date(2009, 11, 10, 23, 2, 15, 0, time.UTC), false},
		{"2009-11-10T23:02:15+01:00", time.Date(2009, 11, 10, 23, 2, 15, 0, time.FixedZone("", 3600)), false},
		{"2009-11-10T22:02:15Z", time.Date(2009, 11, 10, 22, 2, 15, 0, time.UTC), false},
		{"2009-11-10T11:12+02:00", time.Date(2009, 11, 10, 11, 12, 0, 0, time.FixedZone("", 2*3600)), false},
		{"2009-11-10T11:12:13Z", time.Date(2009, 11, 10, 11, 12, 13, 0, time.UTC), false},
		{"2009-11-10T18:15:22+00:00", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"2009-11-10T18:15:22-05:30", time.Date(2009, 11, 10, 18, 15, 22, 0, time.FixedZone("", -5*3600-30*60)), false},
		{"2009-11-10T11:12 UTC", time.Date(2009, 11, 10, 11, 12, 0, 0, time.UTC), false},
		{"2009-11-10T23:02:15 UTC", time.Date(2009, 11, 10, 23, 2, 15, 0, time.UTC), false},
		{"2009-11-10T18:15:22.654321+01:00", time.Date(2009, 11, 10, 18, 15, 22, 654321000, time.FixedZone("", 3600)), false},
		{"Tue 2009-11-10T11:12:13+01:00", time.Date(2009, 11, 10, 11, 12, 13, 0, time.FixedZone("", 3600)), false},
		{"Tue 2009-11-10T18:15:22Z", time.Date(2009, 11, 10, 18, 15, 22, 0, time.UTC), false},
		{"Tue 2009-11-10T11:12:13.5Z", time.Date(2009, 11, 10, 11, 12, 13, 500000000, time.UTC), false},
		{"Tue 2009-11-10T11:12:13.654321+01:00", time.Date(2009, 11, 10, 11, 12, 13, 654321000, time.FixedZone("", 3600)), false},
		{"09-11-10T18:15:22", time.Time{}, true},
		{"09-11-10T18:15:22Z", time.Time{}, true},
		// relative
		{"+3h30min", time.Date(2009, 11, 11, 2, 30, 0, 0, time.UTC), false},
		{"-5s", time.Date(2009, 11, 10, 22, 59, 55, 0, time.UTC), false},
		{"11min ago", time.Date(2009, 11, 10, 22, 49, 0, 0, time.UTC), false},
		{"1h left", time.Date(2009, 11, 11, 0, 0, 0, 0, time.UTC), false},
		{"+", time.Time{}, true},
		{"-", time.Time{}, true},
		{"-abc", time.Time{}, true},
		{"+abc", time.Time{}, true},
		{"abc ago", time.Time{}, true},
		{"abc left", time.Time{}, true},
		{"+5s -5s", time.Time{}, true},
		{"+5s UTC", time.Time{}, true},
		// unix
		{"@1395716396", time.Unix(1395716396, 0), false},
		{"@1395716396.11111", time.Unix(1395716396, 111110000), false},
		{"@1395716396.654321", time.Unix(1395716396, 654321000), false},
		{"@0", time.Unix(0, 0), false},
		{"@0.5", time.Unix(0, 500000000), false},
		{" @1395716396", time.Time{}, true},
		{"  @0", time.Time{}, true},
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
				systemdtime.ParseTimestamp(bc.input)
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
