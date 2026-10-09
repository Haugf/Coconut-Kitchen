package main

import (
	"testing"
	"time"
)

func vehicleEntity(tripID string, status uint64, stop string) []byte {
	vp := cat(pbBytes(1, pbBytes(1, []byte(tripID))), pbVarint(4, status), pbBytes(7, []byte(stop)))
	return pbBytes(2, cat(pbBytes(1, []byte("v-"+tripID)), pbBytes(4, vp)))
}

func tripWithID(id, route string, stus ...[]byte) []byte {
	tu := cat(pbBytes(1, cat(pbBytes(1, []byte(id)), pbBytes(5, []byte(route)))))
	for _, s := range stus {
		tu = append(tu, s...)
	}
	return pbBytes(2, cat(pbBytes(1, []byte("e-"+id)), pbBytes(3, tu)))
}

func TestPlaceTrains(t *testing.T) {
	now := time.Now()
	at := func(sec int64) int64 { return now.Unix() + sec }
	feed := cat(
		// Moving: one minute from Forest Av, coming from Fresh Pond Rd.
		tripWithID("moving", "M", stu("M05N", at(60)), stu("M06N", at(180))),
		vehicleEntity("moving", statusInTransit, "M05N"),
		// Stopped at Fresh Pond Rd, Forest Av next.
		tripWithID("stopped", "M", stu("M04N", at(10)), stu("M05N", at(150))),
		vehicleEntity("stopped", statusStopped, "M04N"),
		// Not assigned: still at Metropolitan Av, the start of the line.
		tripWithID("waiting", "M", stu("M01N", at(300)), stu("M04N", at(480)), stu("M05N", at(600))),
		// Coming from beyond the stations we draw.
		tripWithID("far", "L", stu("L22N", at(200)), stu("L21N", at(300)), stu("L17N", at(900))),
		vehicleEntity("far", statusInTransit, "L22N"),
	)
	trips, err := parseFeed(feed)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]MapVehicle{}
	for _, tr := range trips {
		stop := "M05N"
		if tr.Route == "L" {
			stop = "L17N"
		}
		if v, ok := placeTrain(tr, stop, now); ok {
			got[v.ID] = v
		}
	}
	want := map[string]string{"moving": "moving", "stopped": "stopped", "waiting": "waiting", "far": "far"}
	for id, state := range want {
		v, ok := got[id]
		if !ok {
			t.Fatalf("%s: not placed", id)
		}
		if v.State != state {
			t.Errorf("%s: state %q, want %q", id, v.State, state)
		}
	}
	// The moving train sits between Fresh Pond Rd and Forest Av.
	mv := got["moving"]
	fo, fr := stations["M05"], stations["M04"]
	if mv.Lon <= fo.Lon || mv.Lon >= fr.Lon {
		t.Errorf("moving train at lon %f, want between %f and %f", mv.Lon, fo.Lon, fr.Lon)
	}
	if mv.From == nil || mv.From.Lat != fr.Lat {
		t.Errorf("moving train should trail from Fresh Pond Rd")
	}
	if got["waiting"].Minutes != 10 || got["waiting"].Note != "hasn't left Metropolitan Av" {
		t.Errorf("waiting: %+v", got["waiting"])
	}
	if got["far"].Lat != stations["L21"].Lat {
		t.Errorf("far train should park at the end of the line")
	}
}
