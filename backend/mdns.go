package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// A minimal mDNS browser, enough to find Google speakers on the home
// network by the name they have in the Google Home app. It asks once
// from an ordinary UDP port, so it works alongside avahi on the Pi.

type castDevice struct {
	Name  string // "Kitchen speaker" (the fn= TXT entry)
	Model string // "Google Nest Mini" (md=)
	Addr  string // host:port
}

const castService = "_googlecast._tcp.local"

var mdnsGroup = &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}

// discoverCast lists the Cast devices that answer within wait.
func discoverCast(ctx context.Context, wait time.Duration) ([]castDevice, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	q := mdnsQuery(castService)
	deadline := time.Now().Add(wait)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	// Ask twice in case the first packet is lost on Wi-Fi.
	if _, err := conn.WriteToUDP(q, mdnsGroup); err != nil {
		return nil, err
	}
	resend := time.AfterFunc(wait/3, func() { _, _ = conn.WriteToUDP(q, mdnsGroup) })
	defer resend.Stop()

	recs := map[string]*mdnsInstance{}
	addrs := map[string]net.IP{}
	buf := make([]byte, 9000)
	for {
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				break
			}
			return nil, err
		}
		if err := parseMDNS(buf[:n], recs, addrs); err != nil {
			continue
		}
		// A speaker that sends no A record is still reachable at the
		// address it answered from.
		for _, in := range recs {
			if in.from == nil {
				in.from = from.IP
			}
		}
	}

	var out []castDevice
	for _, in := range recs {
		if in.port == 0 {
			continue
		}
		ip := addrs[strings.ToLower(in.target)]
		if ip == nil {
			ip = in.from
		}
		if ip == nil {
			continue
		}
		name := in.txt["fn"]
		if name == "" {
			name = strings.TrimSuffix(in.instance, "."+castService)
		}
		out = append(out, castDevice{Name: name, Model: in.txt["md"], Addr: net.JoinHostPort(ip.String(), fmt.Sprint(in.port))})
	}
	return out, nil
}

// mdnsQuery builds a PTR question with the unicast-response bit set.
func mdnsQuery(name string) []byte {
	b := make([]byte, 12) // id 0, flags 0, one question
	binary.BigEndian.PutUint16(b[4:], 1)
	b = appendName(b, name)
	return binary.BigEndian.AppendUint16(binary.BigEndian.AppendUint16(b, 12), 0x8001) // PTR, IN|QU
}

func appendName(b []byte, name string) []byte {
	for _, label := range strings.Split(strings.TrimSuffix(name, "."), ".") {
		b = append(b, byte(len(label)))
		b = append(b, label...)
	}
	return append(b, 0)
}

type mdnsInstance struct {
	instance string
	target   string
	port     uint16
	txt      map[string]string
	from     net.IP
}

// parseMDNS reads every record in one response, keeping the ones that
// describe Cast devices.
func parseMDNS(msg []byte, recs map[string]*mdnsInstance, addrs map[string]net.IP) error {
	if len(msg) < 12 {
		return errTruncated
	}
	qd := int(binary.BigEndian.Uint16(msg[4:]))
	rr := int(binary.BigEndian.Uint16(msg[6:])) + int(binary.BigEndian.Uint16(msg[8:])) + int(binary.BigEndian.Uint16(msg[10:]))
	off := 12
	for i := 0; i < qd; i++ {
		_, n, err := readName(msg, off)
		if err != nil {
			return err
		}
		off = n + 4
	}
	inst := func(name string) *mdnsInstance {
		k := strings.ToLower(name)
		if recs[k] == nil {
			recs[k] = &mdnsInstance{instance: name, txt: map[string]string{}}
		}
		return recs[k]
	}
	for i := 0; i < rr; i++ {
		name, n, err := readName(msg, off)
		if err != nil {
			return err
		}
		if n+10 > len(msg) {
			return errTruncated
		}
		typ := binary.BigEndian.Uint16(msg[n:])
		size := int(binary.BigEndian.Uint16(msg[n+8:]))
		data := n + 10
		if data+size > len(msg) {
			return errTruncated
		}
		rdata := msg[data : data+size]
		off = data + size

		switch typ {
		case 12: // PTR: service → instance
			if strings.EqualFold(name, castService) {
				target, _, err := readName(msg, data)
				if err == nil {
					inst(target)
				}
			}
		case 33: // SRV: instance → host and port
			if strings.HasSuffix(strings.ToLower(name), castService) && size >= 7 {
				target, _, err := readName(msg, data+6)
				if err == nil {
					in := inst(name)
					in.port = binary.BigEndian.Uint16(rdata[4:])
					in.target = target
				}
			}
		case 16: // TXT: fn=Kitchen speaker, md=Google Nest Mini, …
			if strings.HasSuffix(strings.ToLower(name), castService) {
				in := inst(name)
				for j := 0; j < len(rdata); {
					l := int(rdata[j])
					j++
					if j+l > len(rdata) {
						break
					}
					if k, v, ok := strings.Cut(string(rdata[j:j+l]), "="); ok {
						in.txt[k] = v
					}
					j += l
				}
			}
		case 1: // A
			if size == 4 {
				addrs[strings.ToLower(name)] = net.IP(append([]byte(nil), rdata...))
			}
		}
	}
	return nil
}

// readName decodes a possibly compressed DNS name at off and returns the
// offset just past it.
func readName(msg []byte, off int) (string, int, error) {
	var labels []string
	end := -1
	for jumps := 0; ; {
		if off >= len(msg) {
			return "", 0, errTruncated
		}
		l := int(msg[off])
		switch {
		case l == 0:
			if end < 0 {
				end = off + 1
			}
			return strings.Join(labels, "."), end, nil
		case l&0xC0 == 0xC0:
			if off+1 >= len(msg) {
				return "", 0, errTruncated
			}
			if end < 0 {
				end = off + 2
			}
			if jumps++; jumps > 20 {
				return "", 0, errors.New("mdns: name loop")
			}
			off = int(binary.BigEndian.Uint16(msg[off:]) & 0x3FFF)
		default:
			if off+1+l > len(msg) {
				return "", 0, errTruncated
			}
			labels = append(labels, string(msg[off+1:off+1+l]))
			off += 1 + l
		}
	}
}
