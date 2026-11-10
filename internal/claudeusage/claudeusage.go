// Package claudeusage holds the hardware-independent logic of the Claude usage display:
// protocol parsing, freshness decision and text formatting.
//
// Protocol: one JSON line per update, e.g.
//
//	{"v":1,"ts":1794303553,"five_hour":{"pct":42.1,"resets_at":1800000000},"seven_day":null}
package claudeusage

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Freshness is how long received data counts as current.
const Freshness = 15 * time.Second

type Window struct {
	Pct      float64 `json:"pct"`
	ResetsAt *int64  `json:"resets_at"`
}

type Msg struct {
	V          int       `json:"v"`
	Ts         int64     `json:"ts"`
	FiveHour   *Window   `json:"five_hour"`
	SevenDay   *Window   `json:"seven_day"`
	ReceivedAt time.Time `json:"-"`
}

// ParseLine validates one protocol line. Returns nil if it is not valid.
func ParseLine(line []byte, now time.Time) *Msg {
	var m Msg
	if err := json.Unmarshal(line, &m); err != nil || m.V != 1 {
		return nil
	}
	m.ReceivedAt = now
	return &m
}

// Pct renders a window as "42%", or "--" if the window is missing.
func Pct(w *Window) string {
	if w == nil {
		return "--"
	}
	return fmt.Sprintf("%d%%", int(w.Pct+0.5))
}

// PctVar is the 0-100 integer string for the bar, 0 if the window is missing.
func PctVar(w *Window) string {
	if w == nil {
		return "0"
	}
	return strconv.Itoa(int(w.Pct + 0.5))
}

// Reset renders the time until reset. The device clock may be wrong while offline, so the
// remaining time is computed from the PC's own timestamp (Ts) plus the time since receipt.
func Reset(w *Window, m *Msg, now time.Time) string {
	if w == nil || w.ResetsAt == nil {
		return ""
	}
	remaining := time.Duration(*w.ResetsAt-m.Ts)*time.Second - now.Sub(m.ReceivedAt)
	if remaining <= 0 {
		return "Reset gleich"
	}
	mins := int(remaining.Minutes())
	switch {
	case mins < 60:
		return fmt.Sprintf("Reset in %d Min", mins)
	case mins < 48*60:
		return fmt.Sprintf("Reset in %d Std %d Min", mins/60, mins%60)
	default:
		return fmt.Sprintf("Reset in %d Tg %d Std", mins/(24*60), (mins/60)%24)
	}
}

// ShouldShow is the decision: USB connected to the host AND data fresh.
func ShouldShow(usbConfigured bool, m *Msg, now time.Time) bool {
	return usbConfigured && m != nil && now.Sub(m.ReceivedAt) < Freshness
}
