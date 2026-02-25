# go-systemd-time

Go implementation of [systemd time](https://www.freedesktop.org/software/systemd/man/latest/systemd.time.html).

> [!note]
> Calendar events are not yet supported.

I don't like maintaining docs in two places, so below are just a few examples and the rest is on [pkg.go.dev](https://pkg.go.dev/gitlab.com/allddd/go-systemd-time).

## Time spans

`ParseTimespan` parses a time span string and returns a `time.Duration`. It supports units from nanoseconds to years (and their abbreviations), decimal values, combinations, bare numbers, etc. See the [full docs](https://pkg.go.dev/gitlab.com/allddd/go-systemd-time#ParseTimespan) for more info.

```go
package main

import (
	"fmt"
	"log"

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
			log.Fatal(err)
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
	"log"

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
			log.Fatal(err)
		}
		fmt.Printf("%q is a %s.\n", ts, t.Weekday())
	}
}
```

This project is licensed under the BSD 2-Clause License. See [LICENSE](./LICENSE) for more details.
