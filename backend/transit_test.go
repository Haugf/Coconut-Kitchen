package main

import (
	"encoding/binary"
	"testing"
	"time"
)

// Tiny protobuf encoder for building test feeds.
func pbVarint(field int, v uint64) []byte {
	b := binary.AppendUvarint(nil, uint64(field<<3|0))
	return binary.AppendUvarint(b, v)
}

func pbBytes(field int, v []byte) []byte {
	b := binary.AppendUvarint(nil, uint64(field<<3|2))
	b = binary.AppendUvarint(b, uint64(len(v)))
	return append(b, v...)
}

func cat(parts ...[]byte) []byte {
	var b []byte
	for _, p := range parts {
		b = append(b, p...)
	}
	return b
}

func stu(stop string, arrival int64) []byte {
	return pbBytes(2, cat(
		pbVarint(1, 7), // stop_sequence
		pbBytes(2, pbVarint(2, uint64(arrival))),
		pbBytes(4, []byte(stop)),
		pbBytes(1001, []byte{0x0a, 0x01, 'x'}), // NYCT extension, ignored
	))
}

func trip(route string, stus ...[]byte) []byte {
	tu := cat(pbBytes(1, cat(pbBytes(1, []byte("trip-1")), pbBytes(5, []byte(route)))))
	for _, s := range stus {
		tu = append(tu, s...)
	}
	return pbBytes(2, cat(pbBytes(1, []byte("e1")), pbBytes(3, tu)))
}

func TestParseFeedAndMinutes(t *testing.T) {
	now := time.Now()
	at := func(min float64) int64 { return now.Add(time.Duration(min * float64(time.Minute))).Unix() }
	feed := cat(
		pbBytes(1, pbBytes(1, []byte("2.0"))), // header
		trip("L", stu("L17N", at(3)), stu("L16N", at(5))),
		trip("L", stu("L17S", at(4))),
		trip("L", stu("L17N", at(9.2))),
		trip("L", stu("L17N", at(-5))), // already gone
		trip("L", stu("L17N", at(16)), stu("L15N", at(18))),
		trip("L", stu("L17N", at(25))),
	)
	trips, err := parseFeed(feed)
	if err != nil {
		t.Fatal(err)
	}
	if len(trips) != 6 || trips[0].Route != "L" || len(trips[0].Stops) != 2 {
		t.Fatalf("parsed %+v", trips)
	}
	var times []int64
	for _, tr := range trips {
		for _, s := range tr.Stops {
			if s.ID == "L17N" {
				times = append(times, s.Time)
			}
		}
	}
	got := nextMinutes(times, now)
	want := []int{3, 9, 16}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("minutes %v, want %v", got, want)
	}
	if _, err := parseFeed(feed[:len(feed)-3]); err == nil {
		t.Fatal("expected an error for a truncated feed")
	}
}

func TestTitleCase(t *testing.T) {
	if got := titleCase("WYCKOFF HOSP"); got != "Wyckoff Hosp" {
		t.Fatal(got)
	}
	if got := firstString([]byte(`["SPRING CREEK"]`)); got != "SPRING CREEK" {
		t.Fatal(got)
	}
}
