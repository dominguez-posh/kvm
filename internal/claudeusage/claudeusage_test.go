package claudeusage

import (
	"testing"
	"time"
)

func TestParseLine(t *testing.T) {
	now := time.Unix(1000, 0)

	m := ParseLine([]byte(`{"v":1,"ts":5,"five_hour":{"pct":42.4,"resets_at":100},"seven_day":null}`+"\n"), now)
	if m == nil || m.FiveHour == nil || m.SevenDay != nil {
		t.Fatalf("unexpected parse result: %+v", m)
	}
	if got := Pct(m.FiveHour); got != "42%" {
		t.Errorf("pct = %q", got)
	}
	if got := Pct(m.SevenDay); got != "--" {
		t.Errorf("nil window = %q", got)
	}
	if got := PctVar(m.SevenDay); got != "0" {
		t.Errorf("PctVar nil = %q", got)
	}
	if got := PctVar(m.FiveHour); got != "42" {
		t.Errorf("PctVar = %q", got)
	}

	for _, bad := range []string{"", "garbage", `{"v":3}`, `{"ts":1}`, `{"v":1`} {
		if ParseLine([]byte(bad), now) != nil {
			t.Errorf("accepted invalid line %q", bad)
		}
	}
}

func TestShouldShow(t *testing.T) {
	now := time.Unix(1000, 0)
	fresh := &Msg{ReceivedAt: now.Add(-5 * time.Second)}
	stale := &Msg{ReceivedAt: now.Add(-20 * time.Second)}

	cases := []struct {
		name string
		usb  bool
		m    *Msg
		want bool
	}{
		{"usb+fresh", true, fresh, true},
		{"usb unplugged", false, fresh, false},
		{"app stopped", true, stale, false},
		{"no data yet", true, nil, false},
	}
	for _, c := range cases {
		if got := ShouldShow(c.usb, c.m, now); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestReset(t *testing.T) {
	recv := time.Unix(5000, 0) // device clock is irrelevant, only elapsed time counts
	m := &Msg{Ts: 1000, ReceivedAt: recv}
	at := func(sec int64) *Window { return &Window{ResetsAt: &sec} }

	cases := []struct {
		w       *Window
		elapsed time.Duration
		want    string
	}{
		{at(1000 + 42*60), 0, "Reset in 42 Min"},
		{at(1000 + 2*3600 + 14*60), 0, "Reset in 2 Std 14 Min"},
		{at(1000 + 3*86400 + 4*3600), 0, "Reset in 3 Tg 4 Std"},
		{at(1000 + 42*60), 2 * time.Minute, "Reset in 40 Min"},
		{at(1000 + 60), 2 * time.Minute, "Reset gleich"},
		{nil, 0, ""},
		{&Window{}, 0, ""},
	}
	for i, c := range cases {
		if got := Reset(c.w, m, recv.Add(c.elapsed)); got != c.want {
			t.Errorf("case %d: got %q want %q", i, got, c.want)
		}
	}
}
