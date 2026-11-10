package watchface

import (
	"strings"
	"testing"
	"time"
)

func parseSys(line string) *System {
	e, ok := ParseEnvelope([]byte(line))
	if !ok {
		return nil
	}
	return ParseSystem([]byte(line), e, time.Unix(100, 0))
}

func TestParseSystemValid(t *testing.T) {
	s := parseSys(`{"v":2,"face":"system","ts":5,"cpu_pct":12.4,"cpu_ghz":3.44,"ram_pct":40.6,"ram_used_mb":8396.8,"ram_total_mb":32256}`)
	if s == nil || s.CPU == nil || s.RAM == nil {
		t.Fatalf("got %+v", s)
	}
	if SystemPct(s.CPU) != "12%" || SystemPct(s.RAM) != "41%" {
		t.Errorf("pct = %s %s", SystemPct(s.CPU), SystemPct(s.RAM))
	}
	if SystemPctVar(s.RAM) != "41" {
		t.Errorf("var = %s", SystemPctVar(s.RAM))
	}
	// free = 32256 - 8396.8 = 23859.2 MB = 23,3 GB; total 31,5 GB
	if got := s.RAMText(); got != "23,3 von 31,5 GB frei" {
		t.Errorf("text = %q", got)
	}
	if got := s.CPUText(); got != "Takt 3,4 GHz" {
		t.Errorf("cpu text = %q", got)
	}
}

func TestParseSystemMissingAndInvalid(t *testing.T) {
	cases := []struct {
		name, line         string
		cpuNil, ramNil, tx bool // tx: RAMText empty
	}{
		{"missing all", `{"v":2,"face":"system","ts":5}`, true, true, true},
		{"cpu null", `{"v":2,"face":"system","cpu_pct":null,"ram_pct":10,"ram_used_mb":1,"ram_total_mb":2}`, true, false, false},
		{"negative", `{"v":2,"face":"system","cpu_pct":-1,"ram_pct":10}`, true, false, true},
		{"over 100", `{"v":2,"face":"system","cpu_pct":100.5,"ram_pct":101}`, true, true, true},
		{"used above total", `{"v":2,"face":"system","cpu_pct":1,"ram_pct":1,"ram_used_mb":9,"ram_total_mb":3}`, false, false, true},
		{"negative mb", `{"v":2,"face":"system","cpu_pct":1,"ram_pct":1,"ram_used_mb":-9,"ram_total_mb":3}`, false, false, true},
		{"edge 0 and 100", `{"v":2,"face":"system","cpu_pct":0,"ram_pct":100,"ram_used_mb":2,"ram_total_mb":2}`, false, false, false},
	}
	for _, c := range cases {
		s := parseSys(c.line)
		if s == nil {
			t.Errorf("%s: nil", c.name)
			continue
		}
		if (s.CPU == nil) != c.cpuNil || (s.RAM == nil) != c.ramNil || (s.RAMText() == "") != c.tx {
			t.Errorf("%s: cpuNil=%v ramNil=%v text=%q", c.name, s.CPU == nil, s.RAM == nil, s.RAMText())
		}
	}
	if s := parseSys(`{"v":2,"face":"system","cpu_pct":-1}`); SystemPct(s.CPU) != "--" || SystemPctVar(s.CPU) != "0" {
		t.Error("invalid value must show -- and 0")
	}
}

func TestParseSystemRejects(t *testing.T) {
	for _, line := range []string{
		`{"v":2,"face":"clock","ts":5}`,
		`{"v":2,"face":"system","cpu_pct":"abc"}`, // wrong type
	} {
		if s := parseSys(line); s != nil {
			t.Errorf("%q: want nil, got %+v", line, s)
		}
	}
	if s := parseSys(`{"v":2,"face":"system"`); s != nil {
		t.Error("broken JSON must give nil")
	}
}

func TestCPUGHz(t *testing.T) {
	cases := []struct {
		json string
		want string
	}{
		{`"cpu_ghz":3.4`, "Takt 3,4 GHz"},
		{`"cpu_ghz":0.1`, "Takt 0,1 GHz"},
		{`"cpu_ghz":10`, "Takt 10,0 GHz"},
		{`"cpu_ghz":0.09`, "Takt --"},
		{`"cpu_ghz":10.1`, "Takt --"},
		{`"cpu_ghz":0`, "Takt --"},
		{`"cpu_ghz":-2`, "Takt --"},
		{`"cpu_ghz":null`, "Takt --"},
		{`"cpu_pct":5`, "Takt --"}, // missing
	}
	for _, c := range cases {
		s := parseSys(`{"v":2,"face":"system",` + c.json + `}`)
		if s == nil {
			t.Errorf("%s: nil", c.json)
			continue
		}
		if got := s.CPUText(); got != c.want {
			t.Errorf("%s: %q, want %q", c.json, got, c.want)
		}
	}
	if s := parseSys(`{"v":2,"face":"system","cpu_ghz":"fast"}`); s != nil {
		t.Error("wrong type must give nil")
	}
}

func TestRAMTextAscii(t *testing.T) {
	s := parseSys(`{"v":2,"face":"system","ram_used_mb":1024,"ram_total_mb":16384}`)
	if got := s.RAMText(); got != "15,0 von 16,0 GB frei" || strings.ContainsAny(got, ".ä") {
		t.Errorf("text = %q", got)
	}
}
