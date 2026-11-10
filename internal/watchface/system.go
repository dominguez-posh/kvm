package watchface

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// FaceSystem is the id of the CPU/RAM face.
const FaceSystem = "system"

// System is the state of the system face. A nil field was missing or invalid in the line and
// is shown as "--".
//
//	{"v":2,"face":"system","ts":..,"cpu_pct":12.5,"ram_pct":40.1,"ram_used_mb":8192,"ram_total_mb":32768}
type System struct {
	CPU        *float64
	RAM        *float64
	RAMUsedMB  *float64
	RAMTotalMB *float64
	ReceivedAt time.Time
}

type systemLine struct {
	CPU        *float64 `json:"cpu_pct"`
	RAM        *float64 `json:"ram_pct"`
	RAMUsedMB  *float64 `json:"ram_used_mb"`
	RAMTotalMB *float64 `json:"ram_total_mb"`
}

// validPct keeps a percentage only if it is a finite number within 0..100.
func validPct(p *float64) *float64 {
	if p == nil || math.IsNaN(*p) || math.IsInf(*p, 0) || *p < 0 || *p > 100 {
		return nil
	}
	return p
}

// validMB keeps a memory size only if it is finite and not negative.
func validMB(p *float64) *float64 {
	if p == nil || math.IsNaN(*p) || math.IsInf(*p, 0) || *p < 0 {
		return nil
	}
	return p
}

// ParseSystem validates a system line. Missing or invalid fields become nil; a line that is
// not for the system face or not valid JSON gives nil.
func ParseSystem(line []byte, e Envelope, now time.Time) *System {
	if e.Face != FaceSystem {
		return nil
	}
	var l systemLine
	if err := json.Unmarshal(line, &l); err != nil {
		return nil
	}
	s := &System{CPU: validPct(l.CPU), RAM: validPct(l.RAM), RAMUsedMB: validMB(l.RAMUsedMB), RAMTotalMB: validMB(l.RAMTotalMB), ReceivedAt: now}
	// used > total cannot be right: show no memory text rather than nonsense
	if s.RAMUsedMB != nil && s.RAMTotalMB != nil && *s.RAMUsedMB > *s.RAMTotalMB {
		s.RAMUsedMB, s.RAMTotalMB = nil, nil
	}
	return s
}

// SystemPct renders a percentage as "42%", or "--" if missing.
func SystemPct(p *float64) string {
	if p == nil {
		return "--"
	}
	return fmt.Sprintf("%d%%", int(*p+0.5))
}

// SystemPctVar is the 0-100 integer string for the bar, "0" if missing.
func SystemPctVar(p *float64) string {
	if p == nil {
		return "0"
	}
	return strconv.Itoa(int(*p + 0.5))
}

// RAMText renders "8,2 von 31,5 GB" (German decimal comma, ASCII only), "" if a size is missing.
func (s *System) RAMText() string {
	if s.RAMUsedMB == nil || s.RAMTotalMB == nil {
		return ""
	}
	gb := func(mb float64) string {
		return strings.Replace(strconv.FormatFloat(mb/1024, 'f', 1, 64), ".", ",", 1)
	}
	return gb(*s.RAMUsedMB) + " von " + gb(*s.RAMTotalMB) + " GB"
}
