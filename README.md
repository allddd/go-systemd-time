# go-systemd-time

Go implementation of [systemd time](https://www.freedesktop.org/software/systemd/man/latest/systemd.time.html).

I don't like maintaining docs in two places, so below are just a few examples and the rest is on [pkg.go.dev](https://pkg.go.dev/gitlab.com/allddd/go-systemd-time).

## Time spans

`ParseTimespan` parses a time span string and returns a `time.Duration`. It supports units from nanoseconds to years (and their abbreviations), decimal values, combinations, bare numbers, etc. See the [full docs](https://pkg.go.dev/gitlab.com/allddd/go-systemd-time#ParseTimespan) for more info.

```go
package main

import (
	"fmt"

	systemdtime "gitlab.com/allddd/go-systemd-time"
)

func main() {
	timespans := []string{
		"1.5days",
		"100ns",
		"1y 12month",
		"2h30min",
		"500 ms",
		"55s500ms",
		"60",
	}
	for _, ts := range timespans {
		d, err := systemdtime.ParseTimespan(ts)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("There are %.0f seconds in %q.\n", d.Seconds(), ts)
	}
}
```

## Timestamps

`ParseTimestamp` parses a timestamp string and returns a `time.Time`. It supports many more formats/variations than shown below, see the [full docs](https://pkg.go.dev/gitlab.com/allddd/go-systemd-time#ParseTimestamp).

```go
package main

import (
	"fmt"

	systemdtime "gitlab.com/allddd/go-systemd-time"
)

func main() {
	timestamps := []string{
		"today",
		"+13h30min",
		"10h11min ago",
		"2009-11-10 18:15:22.654321 UTC",
		"@1395716396",
		"Tue 2009-11-10T18:15:22+01:00",
		"tomorrow Asia/Tokyo",
	}
	for _, ts := range timestamps {
		t, err := systemdtime.ParseTimestamp(ts)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("%q is a %s.\n", ts, t.Weekday())
	}
}
```

## Calendar events

`ParseCalendarEvent` parses a calendar event string and returns a `CalendarEvent`. Its `Next` method returns the next time the event matches. It supports weekdays, wildcards, lists, ranges, repetitions, days counted from the end of the month, shorthands, etc. See the [full docs](https://pkg.go.dev/gitlab.com/allddd/go-systemd-time#ParseCalendarEvent) for more info.

```go
package main

import (
	"fmt"
	"time"

	systemdtime "gitlab.com/allddd/go-systemd-time"
)

func main() {
	events := []string{
		"daily",
		"Mon..Fri 09:00",
		"*-*-01 12:00:00",
		"*-*~1 23:59",
		"Thu,Fri *-*-1,5 11:12:13",
		"weekly Asia/Tokyo",
		"*:0/15",
	}
	for _, ev := range events {
		ce, err := systemdtime.ParseCalendarEvent(ev)
		if err != nil {
			fmt.Println(err)
			continue
		}
		next, ok := ce.Next(time.Now())
		if !ok {
			fmt.Printf("%q never elapses again.\n", ev)
			continue
		}
		fmt.Printf("%q elapses next on %s.\n", ev, next.Format("Mon 2006-01-02 15:04:05"))
	}
}
```

## Copyright

This project is licensed under the BSD 2-Clause License. See [LICENSE](./LICENSE) for more details.
