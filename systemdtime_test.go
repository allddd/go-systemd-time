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
		{"100ns", 100 * time.Nanosecond, false},
		{"100nsec", 100 * time.Nanosecond, false},
		{"100us", 100 * time.Microsecond, false},
		{"100usec", 100 * time.Microsecond, false},
		{"100µs", 100 * time.Microsecond, false},
		{"100μs", 100 * time.Microsecond, false},
		{"500ms", 500 * time.Millisecond, false},
		{"500msec", 500 * time.Millisecond, false},
		{"30s", 30 * time.Second, false},
		{"30sec", 30 * time.Second, false},
		{"30second", 30 * time.Second, false},
		{"30seconds", 30 * time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"5min", 5 * time.Minute, false},
		{"5minute", 5 * time.Minute, false},
		{"5minutes", 5 * time.Minute, false},
		{"999999h", 999999 * time.Hour, false},
		{"3hr", 3 * time.Hour, false},
		{"3hour", 3 * time.Hour, false},
		{"3hours", 3 * time.Hour, false},
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
		// complex
		{"2 h", 2 * time.Hour, false},
		{"3days 12hours", 3*systemdtime.Day + 12*time.Hour, false},
		{"1year 12M", systemdtime.Year + 12*systemdtime.Month, false},
		{"55sec500msec", 55*time.Second + 500*time.Millisecond, false},
		{"300ms20seconds 5d", 300*time.Millisecond + 20*time.Second + 5*systemdtime.Day, false},
		{"2weeks3day", 2*systemdtime.Week + 3*systemdtime.Day, false},
		{"1d 2hr 30s", 1*systemdtime.Day + 2*time.Hour + 30*time.Second, false},
		{"5min10sec500ms", 5*time.Minute + 10*time.Second + 500*time.Millisecond, false},
		{"1w 2days", 1*systemdtime.Week + 2*systemdtime.Day, false},
		// decimal
		{".5s", 500 * time.Millisecond, false},
		{"1.5sec", 1500 * time.Millisecond, false},
		{"1.5days", time.Duration(1.5 * float64(systemdtime.Day)), false},
		{"2.5hr", 2*time.Hour + 30*time.Minute, false},
		{"0.5week", time.Duration(0.5 * float64(systemdtime.Week)), false},
		{"2.5d 1.5hours", time.Duration(2.5*float64(systemdtime.Day)) + time.Duration(1.5*float64(time.Hour)), false},
		// default unit (seconds)
		{"60", 60 * time.Second, false},
		{"1.5", 1500 * time.Millisecond, false},
		{"60 5min", 60*time.Second + 5*time.Minute, false},
		// error
		{"", 0, true},
		{"5xyz", 0, true},
		{"hello", 0, true},
		{"weeks", 0, true},
		{"1.2.3days", 0, true},
		{".", 0, true},
		{"abc123min", 0, true},
		// edge
		{" 10min", 10 * time.Minute, false},
		{"5sec ", 5 * time.Second, false},
		{" 5days  ", 5 * systemdtime.Day, false},
		{"2w    10s", 2*systemdtime.Week + 10*time.Second, false},
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
		{"complex", "1y 12month 2w3d 5h30min 15sec"},
		{"decimal", "2.5h 1.5min 3.75s"},
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
