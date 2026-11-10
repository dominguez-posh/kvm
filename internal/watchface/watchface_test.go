package watchface

import (
	"testing"
	"time"
)

func TestParseEnvelope(t *testing.T) {
	cases := []struct {
		line string
		face string
		ok   bool
	}{
		{`{"v":1,"ts":5,"five_hour":null}`, FaceClaude, true}, // v1 has no face: Claude
		{`{"v":2,"face":"claude","ts":5}`, FaceClaude, true},
		{`{"v":2,"face":"clock","ts":5,"tz_min":60}`, FaceClock, true},
		{`{"v":2,"ts":5}`, "", false}, // v2 must name its face
		{`{"v":3,"face":"clock"}`, "", false},
		{``, "", false},
		{`garbage`, "", false},
		{`{"v":2,"face":"clock"`, "", false},
	}
	for _, c := range cases {
		e, ok := ParseEnvelope([]byte(c.line))
		if ok != c.ok || (ok && e.Face != c.face) {
			t.Errorf("%q: got face=%q ok=%v, want face=%q ok=%v", c.line, e.Face, ok, c.face, c.ok)
		}
	}
}

func TestFresh(t *testing.T) {
	now := time.Unix(1000, 0)
	if !Fresh(true, now.Add(-5*time.Second), now) {
		t.Error("usb+fresh should show")
	}
	if Fresh(false, now.Add(-5*time.Second), now) {
		t.Error("usb unplugged must not show")
	}
	if Fresh(true, now.Add(-20*time.Second), now) {
		t.Error("stale data must not show")
	}
	if Fresh(true, time.Time{}, now) {
		t.Error("no data yet must not show")
	}
}

func TestClock(t *testing.T) {
	// 2026-11-10 13:30:00 UTC is a Tuesday; the sender is at UTC+1 -> 14:30 local.
	ts := time.Date(2026, 11, 10, 13, 30, 0, 0, time.UTC).Unix()
	recv := time.Unix(50000, 0) // device clock is irrelevant, only elapsed time counts
	c := ParseClock(Envelope{V: 2, Face: FaceClock, Ts: ts, TzMin: 60}, recv)
	if c == nil {
		t.Fatal("ParseClock returned nil")
	}
	if got := c.TimeText(recv); got != "14:30" {
		t.Errorf("time = %q", got)
	}
	if got := c.DateText(recv); got != "Dienstag, 10. November" {
		t.Errorf("date = %q", got)
	}
	// 90 seconds later the minute advances
	if got := c.TimeText(recv.Add(90 * time.Second)); got != "14:31" {
		t.Errorf("advanced time = %q", got)
	}
	// crossing midnight advances the date
	late := ParseClock(Envelope{V: 2, Face: FaceClock, Ts: time.Date(2026, 11, 10, 22, 59, 0, 0, time.UTC).Unix(), TzMin: 60}, recv)
	if got := late.DateText(recv.Add(2 * time.Minute)); got != "Mittwoch, 11. November" {
		t.Errorf("date after midnight = %q", got)
	}
	if ParseClock(Envelope{Face: FaceClaude, Ts: 5}, recv) != nil {
		t.Error("wrong face must not parse as clock")
	}
	if ParseClock(Envelope{Face: FaceClock}, recv) != nil {
		t.Error("missing ts must not parse")
	}
}
