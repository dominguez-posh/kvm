package watchface

import (
	"testing"
	"time"
)

func TestTapAllowed(t *testing.T) {
	base := time.Unix(1000, 0)
	cases := []struct {
		name      string
		last, now time.Time
		want      bool
	}{
		{"first tap", time.Time{}, base, true},
		{"immediately again", base, base, false},
		{"100 ms later", base, base.Add(100 * time.Millisecond), false},
		{"just under window", base, base.Add(TapDebounce - time.Millisecond), false},
		{"exactly window", base, base.Add(TapDebounce), true},
		{"seconds later", base, base.Add(3 * time.Second), true},
		{"clock went back", base, base.Add(-time.Second), false},
	}
	for _, c := range cases {
		if got := TapAllowed(c.last, c.now); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestHeartbeat(t *testing.T) {
	if !HeartbeatDue(0) {
		t.Error("heartbeat should be due with an empty queue")
	}
	if HeartbeatDue(1) || HeartbeatDue(4) {
		t.Error("heartbeat must yield to waiting taps")
	}
	if HeartbeatLine != "{\"v\":2,\"event\":\"hb\"}\n" {
		t.Errorf("unexpected heartbeat line %q", HeartbeatLine)
	}
}
