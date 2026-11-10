package kvm

import (
	"bufio"
	"os"
	"time"

	"github.com/jetkvm/kvm/internal/claudeusage"
	"github.com/jetkvm/kvm/internal/sync"
	"golang.org/x/sys/unix"
)

// Claude usage on the display.
//
// A PC-side app sends one JSON line per update over the CDC-ACM serial port
// (/dev/ttyGS0), e.g.
//
//	{"v":1,"ts":1794303553,"five_hour":{"pct":42.1,"resets_at":1800000000},"seven_day":null}
//
// The usage screen is shown only while USB is connected to the host AND data is fresh.
// Otherwise the normal screen is restored.

const (
	claudeUsageScreen  = "claude_usage_screen"
	claudeUsageMaxLine = 4096
)

var (
	claudeUsageLock sync.Mutex
	claudeUsageLast *claudeusage.Msg
	claudeShowing   bool
)

func claudeUsageEnabled() bool {
	return config == nil || !config.ClaudeUsageDisabled
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

func claudeUsageReadLoop() {
	logger := displayLogger.With().Str("component", "claude_usage").Logger()
	for {
		f, err := os.OpenFile(cdcACMDevicePath, os.O_RDWR, 0)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if err := setRawTTY(f); err != nil {
			logger.Warn().Err(err).Msg("failed to set tty raw mode")
		}
		r := bufio.NewReaderSize(f, claudeUsageMaxLine)
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
			if m := claudeusage.ParseLine(line, time.Now()); m != nil {
				claudeUsageLock.Lock()
				claudeUsageLast = m
				claudeUsageLock.Unlock()
				claudeUsageEvaluate()
			}
		}
		f.Close()
		time.Sleep(time.Second)
	}
}

// claudeUsageEvaluate applies the current state to the display. Safe to call often.
func claudeUsageEvaluate() {
	if nativeInstance == nil || !displayInited {
		return
	}
	claudeUsageLock.Lock()
	defer claudeUsageLock.Unlock()

	now := time.Now()
	show := claudeusage.ShouldShow(usbState == "configured", claudeUsageLast, now)

	if !show {
		if claudeShowing {
			claudeShowing = false
			mainScreen := "no_network_screen"
			if networkManager != nil && networkManager.IsUp() {
				mainScreen = "home_screen"
			}
			// only switches if the usage screen is the one currently shown
			nativeInstance.SwitchToScreenIf(mainScreen, []string{claudeUsageScreen})

		}
		return
	}

	m := claudeUsageLast
	nativeInstance.UISetVar("claude_5h_pct", claudeusage.PctVar(m.FiveHour))
	nativeInstance.UISetVar("claude_7d_pct", claudeusage.PctVar(m.SevenDay))
	nativeInstance.UpdateLabelIfChanged("claude_5h_value", claudeusage.Pct(m.FiveHour))
	nativeInstance.UpdateLabelIfChanged("claude_7d_value", claudeusage.Pct(m.SevenDay))
	nativeInstance.UpdateLabelIfChanged("claude_5h_reset", claudeusage.Reset(m.FiveHour, m, now))
	nativeInstance.UpdateLabelIfChanged("claude_7d_reset", claudeusage.Reset(m.SevenDay, m, now))

	if !claudeShowing {
		claudeShowing = true
		wakeDisplay(false, "claude_usage")
	}
	nativeInstance.SwitchToScreenIfDifferent(claudeUsageScreen)
}

// initClaudeUsage starts the serial reader and the freshness/USB watcher.
func initClaudeUsage() {
	if !claudeUsageEnabled() {
		return
	}
	go claudeUsageReadLoop()
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for range t.C {
			claudeUsageEvaluate()
		}
	}()
}
