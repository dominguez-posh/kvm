// Package watchface holds the hardware-independent logic shared by all watch faces of the
// JetKVM display: the line protocol envelope, data freshness and the clock face.
//
// Protocol: one JSON line per update. Every line names its face:
//
//	{"v":2,"face":"claude","ts":1794303553,"five_hour":{"pct":42.1,"resets_at":1800000000},"seven_day":null}
//	{"v":2,"face":"clock","ts":1794303553,"tz_min":60}
//
// Version 1 lines (no "face") are the original Claude usage format and mean face "claude".
// The device clock may be wrong while offline, so every face derives time from the sender's
// "ts" plus the time elapsed since the line was received.
package watchface

import (
	"encoding/json"
	"fmt"
	"time"
)

// Freshness is how long received data counts as current.
const Freshness = 15 * time.Second

const (
	FaceClaude = "claude"
	FaceClock  = "clock"
)

// Envelope is the part of a line every face shares.
type Envelope struct {
	V     int    `json:"v"`
	Face  string `json:"face"`
	Ts    int64  `json:"ts"`
	TzMin int    `json:"tz_min"`
}

// ParseEnvelope validates the envelope. v1 lines without a face are the Claude face.
func ParseEnvelope(line []byte) (Envelope, bool) {
	var e Envelope
	if err := json.Unmarshal(line, &e); err != nil {
		return Envelope{}, false
	}
	switch e.V {
	case 1:
		if e.Face == "" {
			e.Face = FaceClaude
		}
	case 2:
		if e.Face == "" {
			return Envelope{}, false
		}
	default:
		return Envelope{}, false
	}
	return e, true
}

// Fresh reports whether a face may be shown: USB connected to the host AND data recent.
func Fresh(usbConfigured bool, receivedAt, now time.Time) bool {
	return usbConfigured && !receivedAt.IsZero() && now.Sub(receivedAt) < Freshness
}

// Clock is the state of the clock face.
type Clock struct {
	Ts         int64
	TzMin      int
	ReceivedAt time.Time
}

func ParseClock(e Envelope, now time.Time) *Clock {
	if e.Face != FaceClock || e.Ts <= 0 {
		return nil
	}
	return &Clock{Ts: e.Ts, TzMin: e.TzMin, ReceivedAt: now}
}

// Local is the sender's local wall-clock time, advanced by the time since receipt.
func (c *Clock) Local(now time.Time) time.Time {
	elapsed := int64(now.Sub(c.ReceivedAt) / time.Second)
	return time.Unix(c.Ts+elapsed+int64(c.TzMin)*60, 0).UTC()
}

func (c *Clock) TimeText(now time.Time) string {
	return c.Local(now).Format("15:04")
}

var (
	weekdays = [7]string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}
	months   = [12]string{"Januar", "Februar", "Maerz", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
)

func (c *Clock) DateText(now time.Time) string {
	t := c.Local(now)
	return fmt.Sprintf("%s, %d. %s", weekdays[t.Weekday()], t.Day(), months[t.Month()-1])
}
