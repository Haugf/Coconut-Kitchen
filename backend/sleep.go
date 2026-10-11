package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Sleep mode: overnight the screen turns off (the monitor goes to
// standby) and comes back on in the morning. The Pi keeps running, so
// updates and status checks carry on.

type SleepConfig struct {
	// Local times, "HH:MM". Set both to "" to keep the screen on all night.
	From string `json:"from"`
	To   string `json:"to"`
}

func defaultSleep() SleepConfig { return SleepConfig{From: "01:00", To: "06:30"} }

type sleeper struct {
	cfg SleepConfig
	off bool // what we last told the display
}

func newSleeper(cfg *SleepConfig) *sleeper {
	c := defaultSleep()
	if cfg != nil {
		c = *cfg
	}
	return &sleeper{cfg: c}
}

// asleep reports whether t falls in the sleep window, which may wrap
// past midnight.
func (s *sleeper) asleep(t time.Time) bool {
	from, ok1 := clockMinutes(s.cfg.From)
	to, ok2 := clockMinutes(s.cfg.To)
	if !ok1 || !ok2 || from == to {
		return false
	}
	now := t.Hour()*60 + t.Minute()
	if from < to {
		return now >= from && now < to
	}
	return now >= from || now < to
}

func clockMinutes(s string) (int, bool) {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

func (s *sleeper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"asleep": s.asleep(time.Now()), "from": s.cfg.From, "to": s.cfg.To})
}

// run switches the display at the edges of the window. It only acts on
// changes, so turning the screen on by hand overnight sticks until the
// next morning.
func (s *sleeper) run() {
	if _, err := exec.LookPath("wlopm"); err != nil {
		// Without wlopm the page still goes black overnight; the monitor
		// just doesn't drop into standby. (sudo apt install wlopm)
		log.Printf("sleep: wlopm isn't installed, so overnight the page goes black but the monitor stays on")
		return
	}
	first := true
	for {
		want := s.asleep(time.Now())
		if first || want != s.off {
			if err := setDisplay(!want); err != nil {
				log.Printf("sleep: %v", err)
			} else {
				if !first || want {
					log.Printf("sleep: screen %s", map[bool]string{true: "off", false: "on"}[want])
				}
				s.off = want
			}
			first = false
		}
		time.Sleep(30 * time.Second)
	}
}

// setDisplay puts the screen into standby or wakes it, through the
// desktop's Wayland session (same user as this server). It uses power
// management (wlopm) rather than disabling the output: the screen stays
// connected, so the browser window is untouched. Disabling the output
// (wlr-randr --off) took the kiosk browser down with it.
func setDisplay(on bool) error {
	runtime := fmt.Sprintf("/run/user/%d", os.Getuid())
	socks, _ := filepath.Glob(filepath.Join(runtime, "wayland-*"))
	display := ""
	for _, s := range socks {
		if !strings.HasSuffix(s, ".lock") {
			display = filepath.Base(s)
			break
		}
	}
	if display == "" {
		return fmt.Errorf("no desktop session to talk to yet")
	}
	state := "--off"
	if on {
		state = "--on"
	}
	cmd := exec.Command("wlopm", state, "*")
	cmd.Env = append(os.Environ(), "XDG_RUNTIME_DIR="+runtime, "WAYLAND_DISPLAY="+display)
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("wlopm %s: %v %s", state, err, strings.TrimSpace(string(b)))
	}
	return nil
}
