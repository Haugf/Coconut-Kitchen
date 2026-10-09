package main

import (
	"errors"
	"fmt"
)

// A minimal reader for GTFS-realtime protobuf feeds: just enough to get
// arrival times per stop for each trip. Hand-rolled so the server keeps
// a single dependency.
//
// Field numbers, from gtfs-realtime.proto:
//   FeedMessage:    entity = 2
//   FeedEntity:     trip_update = 3, vehicle = 4
//   TripUpdate:     trip = 1, stop_time_update = 2
//   TripDescriptor: trip_id = 1, route_id = 5
//   VehiclePosition: trip = 1, current_status = 4, stop_id = 7
//   StopTimeUpdate: arrival = 2, departure = 3, stop_id = 4
//   StopTimeEvent:  time = 2

type rtTrip struct {
	ID    string
	Route string
	Stops []rtStop
	// From the matching vehicle entity. A trip with no vehicle hasn't
	// left its first stop yet.
	Assigned bool
	Status   int    // 0 incoming at, 1 stopped at, 2 in transit to
	AtStop   string // the stop that status refers to
}

const (
	statusIncoming  = 0
	statusStopped   = 1
	statusInTransit = 2
)

type rtStop struct {
	ID   string
	Time int64 // unix seconds; arrival, or departure if no arrival
}

func parseFeed(b []byte) ([]rtTrip, error) {
	var trips []rtTrip
	type veh struct {
		status int
		stop   string
	}
	vehicles := map[string]veh{}
	err := eachField(b, func(f int, v []byte, _ uint64) error {
		if f != 2 {
			return nil
		}
		return eachField(v, func(f int, v []byte, _ uint64) error {
			switch f {
			case 3:
				t, err := parseTripUpdate(v)
				if err == nil {
					trips = append(trips, t)
				}
				return err
			case 4:
				id, vh := "", veh{status: statusInTransit}
				err := eachField(v, func(f int, v []byte, n uint64) error {
					switch f {
					case 1:
						return eachField(v, func(f int, v []byte, _ uint64) error {
							if f == 1 {
								id = string(v)
							}
							return nil
						})
					case 4:
						vh.status = int(n)
					case 7:
						vh.stop = string(v)
					}
					return nil
				})
				if err == nil && id != "" {
					vehicles[id] = vh
				}
				return err
			}
			return nil
		})
	})
	for i := range trips {
		if vh, ok := vehicles[trips[i].ID]; ok && trips[i].ID != "" {
			trips[i].Assigned = true
			trips[i].Status = vh.status
			trips[i].AtStop = vh.stop
		}
	}
	return trips, err
}

func parseTripUpdate(b []byte) (rtTrip, error) {
	var t rtTrip
	err := eachField(b, func(f int, v []byte, _ uint64) error {
		switch f {
		case 1:
			return eachField(v, func(f int, v []byte, _ uint64) error {
				switch f {
				case 1:
					t.ID = string(v)
				case 5:
					t.Route = string(v)
				}
				return nil
			})
		case 2:
			var s rtStop
			var arrival, departure int64
			err := eachField(v, func(f int, v []byte, _ uint64) error {
				switch f {
				case 4:
					s.ID = string(v)
				case 2, 3:
					return eachField(v, func(g int, _ []byte, n uint64) error {
						if g == 2 {
							if f == 2 {
								arrival = int64(n)
							} else {
								departure = int64(n)
							}
						}
						return nil
					})
				}
				return nil
			})
			if err != nil {
				return err
			}
			s.Time = arrival
			if s.Time == 0 {
				s.Time = departure
			}
			if s.ID != "" && s.Time != 0 {
				t.Stops = append(t.Stops, s)
			}
		}
		return nil
	})
	return t, err
}

var errTruncated = errors.New("gtfs-rt: truncated message")

// eachField walks one protobuf message. For length-delimited fields fn
// gets the bytes; for varints it gets the number.
func eachField(b []byte, fn func(field int, val []byte, num uint64) error) error {
	for i := 0; i < len(b); {
		key, n := uvarint(b[i:])
		if n <= 0 {
			return errTruncated
		}
		i += n
		field, wire := int(key>>3), int(key&7)
		switch wire {
		case 0:
			v, n := uvarint(b[i:])
			if n <= 0 {
				return errTruncated
			}
			i += n
			if err := fn(field, nil, v); err != nil {
				return err
			}
		case 1:
			i += 8
		case 2:
			l, n := uvarint(b[i:])
			if n <= 0 || i+n+int(l) > len(b) {
				return errTruncated
			}
			i += n
			if err := fn(field, b[i:i+int(l)], 0); err != nil {
				return err
			}
			i += int(l)
		case 5:
			i += 4
		default:
			return fmt.Errorf("gtfs-rt: unsupported wire type %d", wire)
		}
		if i > len(b) {
			return errTruncated
		}
	}
	return nil
}

func uvarint(b []byte) (uint64, int) {
	var x uint64
	for i := 0; i < len(b) && i < 10; i++ {
		x |= uint64(b[i]&0x7f) << (7 * i)
		if b[i] < 0x80 {
			return x, i + 1
		}
	}
	return 0, 0
}
