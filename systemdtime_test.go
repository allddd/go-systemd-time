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
