package main

import (
	"bufio"
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
	if _, err := exec.LookPath("wlr-randr"); err != nil {
		return // not on the Pi; the page goes dark on its own instead
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

// setDisplay turns the HDMI output on or off through the desktop's
// Wayland session, which runs as the same user as this server.
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
	env := append(os.Environ(), "XDG_RUNTIME_DIR="+runtime, "WAYLAND_DISPLAY="+display)

	list := exec.Command("wlr-randr")
	list.Env = env
	out, err := list.Output()
	if err != nil {
		return fmt.Errorf("wlr-randr: %w", err)
	}
	output := ""
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "HDMI") {
			output = strings.Fields(line)[0]
			break
		}
	}
	if output == "" {
		return fmt.Errorf("no HDMI screen found")
	}

	args := []string{"--output", output, "--off"}
	if on {
		args = []string{"--output", output, "--on"}
		// Coming back on, use the saved mode; the default one shimmers.
		if mode := displayMode(); mode != "" {
			args = append(args, "--mode", mode)
		}
	}
	cmd := exec.Command("wlr-randr", args...)
	cmd.Env = env
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("wlr-randr %s: %v %s", strings.Join(args, " "), err, strings.TrimSpace(string(b)))
	}
	return nil
}

// displayMode reads MODE from ~/mirror/display.env (the server runs there).
func displayMode() string {
	f, err := os.Open("display.env")
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "MODE="); ok {
			return strings.Trim(v, `"'`)
		}
	}
	return ""
}
