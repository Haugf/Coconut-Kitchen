package main

import (
	"testing"
	"time"
)

func TestSleepWindow(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 10, 9, h, m, 0, 0, time.Local) }
	s := newSleeper(nil) // 01:00 to 06:30
	for _, c := range []struct {
		h, m int
		want bool
	}{{0, 59, false}, {1, 0, true}, {3, 0, true}, {6, 29, true}, {6, 30, false}, {12, 0, false}} {
		if got := s.asleep(at(c.h, c.m)); got != c.want {
			t.Errorf("%02d:%02d asleep=%v, want %v", c.h, c.m, got, c.want)
		}
	}
	wrap := newSleeper(&SleepConfig{From: "23:30", To: "06:00"})
	if !wrap.asleep(at(23, 45)) || !wrap.asleep(at(2, 0)) || wrap.asleep(at(12, 0)) {
		t.Error("window across midnight is wrong")
	}
	if newSleeper(&SleepConfig{}).asleep(at(3, 0)) {
		t.Error("empty times should mean never asleep")
	}
	if plainTitle("Workout 💪🏽 at the gym") != "Workout at the gym" {
		t.Errorf("plainTitle: %q", plainTitle("Workout 💪🏽 at the gym"))
	}
}
