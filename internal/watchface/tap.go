package watchface

import "time"

// TapDebounce is the minimum gap between two taps that count. One physical tap produces several
// indev events (and the old screen handler may fire as well), so anything closer than this is
// the same tap.
const TapDebounce = 400 * time.Millisecond

// TapAllowed reports whether a tap at now counts, given the time of the last counted tap
// (zero time: none yet).
func TapAllowed(last, now time.Time) bool {
	return last.IsZero() || now.Sub(last) >= TapDebounce
}

// HeartbeatInterval is how often the device writes a heartbeat line to the host while the tty is
// open. It lets the PC app notice a dead device -> PC direction (and re-open the port).
const HeartbeatInterval = 5 * time.Second

// HeartbeatLine is the heartbeat event (one JSON line).
const HeartbeatLine = `{"v":2,"event":"hb"}` + "\n"

// HeartbeatDue reports whether a heartbeat may be written now: only when no event (tap) is
// waiting, so a heartbeat never delays or displaces a tap.
func HeartbeatDue(queuedEvents int) bool {
	return queuedEvents == 0
}
