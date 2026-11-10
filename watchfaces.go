package kvm

import (
	"bufio"
	"os"
	"sync/atomic"
	"time"

	"github.com/jetkvm/kvm/internal/claudeusage"
	"github.com/jetkvm/kvm/internal/sync"
	"github.com/jetkvm/kvm/internal/watchface"
	"golang.org/x/sys/unix"
)

// Watch faces on the display.
//
// A PC-side app sends one JSON line per update over the CDC-ACM serial port (/dev/ttyGS0);
// see internal/watchface for the protocol. The line names the face to show. A face is shown
// only while USB is connected to the host AND data is fresh; otherwise the normal screen is
// restored. While a face is shown the display is dimmed by the normal timers but never turned
// off (see tick_displayOff).
//
// To add a face: add a screen to the UI (screens.c/h), a case in handleWatchFaceLine and an
// entry in watchFaces below.

const watchFaceMaxLine = 4096

// watchFaceState is the latest valid frame of the face the sender currently wants.
type watchFaceState struct {
	face       string
	receivedAt time.Time
	claude     *claudeusage.Msg
	clock      *watchface.Clock
	system     *watchface.System
}

type watchFaceDef struct {
	screen string
	apply  func(s *watchFaceState, now time.Time)
}

var watchFaces = map[string]watchFaceDef{
	watchface.FaceClaude: {screen: "claude_usage_screen", apply: applyClaudeFace},
	watchface.FaceClock:  {screen: "clock_screen", apply: applyClockFace},
	watchface.FaceSystem: {screen: "system_screen", apply: applySystemFace},
}

var (
	watchFaceLock    sync.Mutex
	watchFaceLast    *watchFaceState
	watchFaceCurrent string // screen name currently driven by us, "" if none
	watchFaceActive  atomic.Bool

	// events for the PC app (e.g. a tap asking for the next face), written to the serial port
	watchFaceOut = make(chan []byte, 4)
)

// watchFaceTapped is called when the display is tapped while a face is shown. The PC app owns
// the choice of face, so the tap is passed on as an event line; the app switches and sends the
// new face. Taps only happen while a face is shown, i.e. while the app is sending, so no stale
// events pile up.
func watchFaceTapped() {
	select {
	case watchFaceOut <- append([]byte(`{"v":2,"event":"next_face"}`), 10):
	default: // writer busy: drop
	}
}

// watchFaceShowing reports whether a watch face is on the display (keeps the display on).
func watchFaceShowing() bool {
	return watchFaceActive.Load()
}

func claudeUsageEnabled() bool {
	return config == nil || !config.ClaudeUsageDisabled
}

func applyClaudeFace(s *watchFaceState, now time.Time) {
	m := s.claude
	nativeInstance.UISetVar("claude_5h_pct", claudeusage.PctVar(m.FiveHour))
	nativeInstance.UISetVar("claude_7d_pct", claudeusage.PctVar(m.SevenDay))
	nativeInstance.UpdateLabelIfChanged("claude_5h_value", claudeusage.Pct(m.FiveHour))
	nativeInstance.UpdateLabelIfChanged("claude_7d_value", claudeusage.Pct(m.SevenDay))
	nativeInstance.UpdateLabelIfChanged("claude_5h_reset", claudeusage.Reset(m.FiveHour, m, now))
	nativeInstance.UpdateLabelIfChanged("claude_7d_reset", claudeusage.Reset(m.SevenDay, m, now))
}

func applyClockFace(s *watchFaceState, now time.Time) {
	nativeInstance.UpdateLabelIfChanged("clock_time", s.clock.TimeText(now))
	nativeInstance.UpdateLabelIfChanged("clock_date", s.clock.DateText(now))
}

func applySystemFace(s *watchFaceState, now time.Time) {
	m := s.system
	nativeInstance.UISetVar("system_cpu_pct", watchface.SystemPctVar(m.CPU))
	nativeInstance.UISetVar("system_ram_pct", watchface.SystemPctVar(m.RAM))
	nativeInstance.UpdateLabelIfChanged("system_cpu_value", watchface.SystemPct(m.CPU))
	nativeInstance.UpdateLabelIfChanged("system_ram_value", watchface.SystemPct(m.RAM))
	nativeInstance.UpdateLabelIfChanged("system_ram_detail", m.RAMText())
}

func setRawTTY(f *os.File) error {
	fd := int(f.Fd())
	t, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return err
	}
	t.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	t.Oflag &^= unix.OPOST
	t.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	t.Cflag &^= unix.CSIZE | unix.PARENB
	t.Cflag |= unix.CS8
	t.Cc[unix.VMIN] = 1
	t.Cc[unix.VTIME] = 0
	return unix.IoctlSetTermios(fd, unix.TCSETS, t)
}

// handleWatchFaceLine validates one protocol line and stores it as the current state.
func handleWatchFaceLine(line []byte, now time.Time) {
	env, ok := watchface.ParseEnvelope(line)
	if !ok {
		return
	}
	s := &watchFaceState{face: env.Face, receivedAt: now}
	switch env.Face {
	case watchface.FaceClaude:
		if s.claude = claudeusage.ParseLine(line, now); s.claude == nil {
			return
		}
	case watchface.FaceClock:
		if s.clock = watchface.ParseClock(env, now); s.clock == nil {
			return
		}
	case watchface.FaceSystem:
		if s.system = watchface.ParseSystem(line, env, now); s.system == nil {
			return
		}
	default:
		return // unknown face: ignore
	}
	watchFaceLock.Lock()
	watchFaceLast = s
	watchFaceLock.Unlock()
	watchFaceEvaluate()
}

func watchFaceReadLoop() {
	logger := displayLogger.With().Str("component", "watchface").Logger()
	for {
		f, err := os.OpenFile(cdcACMDevicePath, os.O_RDWR, 0)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if err := setRawTTY(f); err != nil {
			logger.Warn().Err(err).Msg("failed to set tty raw mode")
		}
		done := make(chan struct{})
		go func(out *os.File) {
			for {
				select {
				case b := <-watchFaceOut:
					_, _ = out.Write(b)
				case <-done:
					return
				}
			}
		}(f)
		r := bufio.NewReaderSize(f, watchFaceMaxLine)
		for {
			line, err := r.ReadSlice('\n')
			if err == bufio.ErrBufferFull {
				// overlong garbage line: drop the rest of it
				for err == bufio.ErrBufferFull {
					_, err = r.ReadSlice('\n')
				}
				continue
			}
			if err != nil {
				break
			}
			handleWatchFaceLine(line, time.Now())
		}
		close(done)
		f.Close()
		time.Sleep(time.Second)
	}
}

// watchFaceEvaluate applies the current state to the display. Safe to call often.
func watchFaceEvaluate() {
	if nativeInstance == nil || !displayInited {
		return
	}
	watchFaceLock.Lock()
	defer watchFaceLock.Unlock()

	now := time.Now()
	var def watchFaceDef
	show := false
	if watchFaceLast != nil {
		var known bool
		def, known = watchFaces[watchFaceLast.face]
		show = known && watchface.Fresh(usbState == "configured", watchFaceLast.receivedAt, now)
	}

	if !show {
		if watchFaceCurrent != "" {
			watchFaceCurrent = ""
			watchFaceActive.Store(false)
			mainScreen := "no_network_screen"
			if networkManager != nil && networkManager.IsUp() {
				mainScreen = "home_screen"
			}
			// only switches if one of the watch face screens is the one currently shown
			var ours []string
			for _, d := range watchFaces {
				ours = append(ours, d.screen)
			}
			nativeInstance.SwitchToScreenIf(mainScreen, ours)
		}
		return
	}

	def.apply(watchFaceLast, now)

	if watchFaceCurrent == "" {
		watchFaceActive.Store(true)
		wakeDisplay(false, "watchface")
	}
	watchFaceCurrent = def.screen
	nativeInstance.SwitchToScreenIfDifferent(def.screen)
}

// initWatchFaces starts the serial reader and the freshness/USB watcher.
func initWatchFaces() {
	if !claudeUsageEnabled() {
		return
	}
	go watchFaceReadLoop()
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for range t.C {
			watchFaceEvaluate()
		}
	}()
}
