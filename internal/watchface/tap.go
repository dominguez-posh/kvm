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
