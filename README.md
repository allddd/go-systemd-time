# go-systemd-time

Go implementation of [systemd time](https://www.freedesktop.org/software/systemd/man/latest/systemd.time.html) (calendar events not yet supported).

Documentation is available on [pkg.go.dev/gitlab.com/allddd/go-systemd-time](https://pkg.go.dev/gitlab.com/allddd/go-systemd-time).

```go
import (
	systemdtime "gitlab.com/allddd/go-systemd-time"
)

// parses a time span, returns time.Duration
d, err := systemdtime.ParseTimespan("1.5days")
d, err := systemdtime.ParseTimespan("100ns")
d, err := systemdtime.ParseTimespan("1y 12month")
d, err := systemdtime.ParseTimespan("2h30min")
d, err := systemdtime.ParseTimespan("500 ms")
d, err := systemdtime.ParseTimespan("55s500ms")
d, err := systemdtime.ParseTimespan("60")

// parses a timestamp, returns time.Time
t, err := systemdtime.ParseTimestamp("+3h30min")
t, err := systemdtime.ParseTimestamp("11min ago")
t, err := systemdtime.ParseTimestamp("18:15")
t, err := systemdtime.ParseTimestamp("2009-11-10 18:15:22.654321 UTC")
t, err := systemdtime.ParseTimestamp("@1395716396")
t, err := systemdtime.ParseTimestamp("Tue 2009-11-10T18:15:22+01:00")
t, err := systemdtime.ParseTimestamp("tomorrow Asia/Tokyo")
```

This project is licensed under the BSD 2-Clause License. See [LICENSE](./LICENSE) for more details.
