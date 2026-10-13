package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCastMessageRoundTrip(t *testing.T) {
	in := castMsg{Source: "sender-0", Dest: "receiver-0", Namespace: nsReceiver, Payload: `{"type":"GET_STATUS","requestId":1}`}
	frame := encodeCast(in)
	if n := binary.BigEndian.Uint32(frame); int(n) != len(frame)-4 {
		t.Fatalf("length prefix %d, body %d", n, len(frame)-4)
	}
	out, err := decodeCast(frame[4:])
	if err != nil || out != in {
		t.Fatalf("got %+v, %v", out, err)
	}
}

// fakeSpeaker answers like a Google speaker playing a radio station, and
// records what a sender asks it to play.
type fakeSpeaker struct {
	ln     net.Listener
	mu     sync.Mutex
	loaded []string
	app    string // what RECEIVER_STATUS reports
}

func newFakeSpeaker(t *testing.T) *fakeSpeaker {
	srv := httptest.NewUnstartedServer(nil)
	srv.StartTLS()
	cfg := srv.TLS.Clone()
	srv.Close()
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSpeaker{ln: ln, app: "radio"}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go f.serve(c)
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return f
}

func (f *fakeSpeaker) serve(c net.Conn) {
	defer c.Close()
	reply := func(src, ns string, v any) {
		b, _ := json.Marshal(v)
		_, _ = c.Write(encodeCast(castMsg{Source: src, Dest: castSender, Namespace: ns, Payload: string(b)}))
	}
	for {
		var size [4]byte
		if _, err := io.ReadFull(c, size[:]); err != nil {
			return
		}
		b := make([]byte, binary.BigEndian.Uint32(size[:]))
		if _, err := io.ReadFull(c, b); err != nil {
			return
		}
		m, _ := decodeCast(b)
		var p map[string]any
		_ = json.Unmarshal([]byte(m.Payload), &p)
		switch {
		case m.Namespace == nsHeartbeat && p["type"] == "PING":
			reply(m.Dest, nsHeartbeat, map[string]any{"type": "PONG"})
		case m.Namespace == nsReceiver && p["type"] == "GET_STATUS":
			f.mu.Lock()
			app := f.app
			f.mu.Unlock()
			reply("receiver-0", nsReceiver, receiverStatus(app))
		case m.Namespace == nsReceiver && p["type"] == "LAUNCH":
			f.mu.Lock()
			f.app = "player"
			f.mu.Unlock()
			reply("receiver-0", nsReceiver, receiverStatus("player"))
		case m.Namespace == nsMedia && p["type"] == "GET_STATUS":
			reply(m.Dest, nsMedia, map[string]any{"type": "MEDIA_STATUS", "status": []any{map[string]any{
				"playerState": "PLAYING", "currentTime": 12.0,
				"media": map[string]any{"streamType": "LIVE", "metadata": map[string]any{"title": "WNYC 93.9", "subtitle": "All Things Considered"}},
			}}})
		case m.Namespace == nsMedia && p["type"] == "LOAD":
			media := p["media"].(map[string]any)
			f.mu.Lock()
			f.loaded = append(f.loaded, media["contentId"].(string))
			f.mu.Unlock()
			reply(m.Dest, nsMedia, map[string]any{"type": "MEDIA_STATUS", "status": []any{map[string]any{"playerState": "BUFFERING"}}})
		}
	}
}

func receiverStatus(app string) map[string]any {
	apps := []any{}
	switch app {
	case "radio":
		apps = append(apps, map[string]any{"appId": "TUNEIN", "displayName": "TuneIn Radio", "statusText": "Now Casting: WNYC", "transportId": "t-radio", "namespaces": []any{map[string]any{"name": nsMedia}}})
	case "player":
		apps = append(apps, map[string]any{"appId": appDefaultMedia, "displayName": "Default Media Receiver", "transportId": "t-player", "namespaces": []any{map[string]any{"name": nsMedia}}})
	}
	return map[string]any{"type": "RECEIVER_STATUS", "status": map[string]any{"applications": apps}}
}

func TestCasterFollowsWhatsPlaying(t *testing.T) {
	f := newFakeSpeaker(t)
	c := newCaster(CastConfig{Name: "Kitchen speaker", Host: f.ln.Addr().String()})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.run(ctx)

	var np NowPlaying
	for i := 0; i < 50; i++ {
		if np = c.snapshot(time.Now()); np.Title != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !np.Connected || np.State != "playing" || np.App != "TuneIn Radio" || np.Title != "WNYC 93.9" || np.Subtitle != "All Things Considered" || !np.Live {
		t.Fatalf("now playing = %+v", np)
	}
	if np.Speaker != "Kitchen speaker" {
		t.Errorf("speaker = %q", np.Speaker)
	}
	h := c.health()
	if h["connected"] != true || h["playing"] != "playing" {
		t.Errorf("health = %v", h)
	}
	if _, ok := h["title"]; ok {
		t.Error("health must not include what's playing")
	}
}

func TestCasterSaysAMessage(t *testing.T) {
	f := newFakeSpeaker(t)
	c := newCaster(CastConfig{Host: f.ln.Addr().String()})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.say(ctx, "Ally says: dinner's ready"); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.loaded) != 1 || !strings.HasPrefix(f.loaded[0], "https://translate.google.com/translate_tts?") || !strings.Contains(f.loaded[0], "q=Ally+says%3A+dinner%27s+ready") {
		t.Fatalf("loaded %v", f.loaded)
	}
	// The spoken message itself isn't "now playing".
	c.applyApp(&castApp{AppID: appDefaultMedia, DisplayName: "Default Media Receiver"}, time.Now())
	c.connected = true
	c.applyMedia(json.RawMessage(`[{"playerState":"PLAYING","media":{"metadata":{"title":"Coconut Kitchen"}}}]`), time.Now())
	if np := c.snapshot(time.Now()); np.State != "" {
		t.Errorf("own message shown as %+v", np)
	}
}

func TestPausedStepsAside(t *testing.T) {
	c := newCaster(CastConfig{})
	c.connected = true
	now := time.Now()
	c.applyApp(&castApp{AppID: "SPOTIFY", DisplayName: "Spotify"}, now)
	c.applyMedia(json.RawMessage(`[{"playerState":"PLAYING","currentTime":30,"media":{"duration":200,"metadata":{"title":"Song","artist":"Band"}}}]`), now)
	if np := c.snapshot(now.Add(10 * time.Second)); np.State != "playing" || np.Position != 40 || np.Duration != 200 || np.Subtitle != "Band" {
		t.Fatalf("playing = %+v", np)
	}
	// A position-only update keeps the song.
	c.applyMedia(json.RawMessage(`[{"playerState":"PAUSED","currentTime":41}]`), now.Add(11*time.Second))
	if np := c.snapshot(now.Add(time.Minute)); np.State != "paused" || np.Title != "Song" || np.Position != 41 {
		t.Fatalf("paused = %+v", np)
	}
	if np := c.snapshot(now.Add(11*time.Second + pausedShowFor + time.Second)); np.State != "" {
		t.Fatalf("long pause still shown: %+v", np)
	}
	c.applyApp(nil, now)
	if np := c.snapshot(now); np.State != "" {
		t.Fatalf("after stop = %+v", np)
	}
}

func TestVoiceURL(t *testing.T) {
	long := strings.Repeat("word ", 60)
	u := voiceURL(defaultVoiceURL, "en", long)
	q := u[strings.Index(u, "q=")+2:]
	if n := len(strings.ReplaceAll(q, "+", " ")); n > 200 {
		t.Errorf("text is %d characters", n)
	}
	if !strings.Contains(u, "tl=en") {
		t.Errorf("url %s", u)
	}
}

func TestPickDevice(t *testing.T) {
	found := []castDevice{{Name: "Living Room TV", Addr: "a"}, {Name: "Kitchen speaker", Addr: "b"}}
	if d, err := pickDevice(found, "kitchen speaker"); err != nil || d.Addr != "b" {
		t.Fatalf("got %v %v", d, err)
	}
	if _, err := pickDevice(found, "Bedroom"); err == nil || !strings.Contains(err.Error(), `"Kitchen speaker"`) {
		t.Fatalf("err = %v", err)
	}
	if d, _ := pickDevice(found, ""); d.Addr != "a" {
		t.Fatalf("first = %v", d)
	}
}

func TestParseMDNS(t *testing.T) {
	// A response like a speaker's: PTR, then SRV, TXT and A, with the
	// instance name compressed against the PTR's.
	msg := make([]byte, 12)
	binary.BigEndian.PutUint16(msg[2:], 0x8400)
	binary.BigEndian.PutUint16(msg[6:], 4)
	rr := func(name []byte, typ uint16, data []byte) {
		msg = append(msg, name...)
		msg = binary.BigEndian.AppendUint16(msg, typ)
		msg = binary.BigEndian.AppendUint16(msg, 1)
		msg = binary.BigEndian.AppendUint32(msg, 120)
		msg = binary.BigEndian.AppendUint16(msg, uint16(len(data)))
		msg = append(msg, data...)
	}
	svc := appendName(nil, castService)
	svcAt := len(msg)
	instAt := svcAt + len(svc) + 10
	inst := append([]byte{byte(len("Google-Nest-Mini-abc"))}, "Google-Nest-Mini-abc"...)
	inst = append(inst, 0xC0|byte(svcAt>>8), byte(svcAt))
	rr(svc, 12, inst)
	ptr := []byte{0xC0 | byte(instAt>>8), byte(instAt)}
	host := appendName(nil, "abc.local")
	rr(ptr, 33, append([]byte{0, 0, 0, 0, 0x1F, 0x49}, host...))
	txt := []byte{}
	for _, kv := range []string{"id=abc", "md=Google Nest Mini", "fn=Kitchen speaker"} {
		txt = append(append(txt, byte(len(kv))), kv...)
	}
	rr(ptr, 16, txt)
	rr(host, 1, []byte{192, 168, 1, 52})

	recs, addrs := map[string]*mdnsInstance{}, map[string]net.IP{}
	if err := parseMDNS(msg, recs, addrs); err != nil {
		t.Fatal(err)
	}
	in := recs[strings.ToLower("Google-Nest-Mini-abc."+castService)]
	if in == nil || in.port != 8009 || in.txt["fn"] != "Kitchen speaker" || in.txt["md"] != "Google Nest Mini" {
		t.Fatalf("instance = %+v (all %v)", in, recs)
	}
	if ip := addrs[strings.ToLower(in.target)]; ip.String() != "192.168.1.52" {
		t.Fatalf("address %v for %q", ip, in.target)
	}
}

func TestSpokenMessage(t *testing.T) {
	if s := spoken(Message{Text: "Dinner's ready", Source: "Ally"}); s != "Ally says: Dinner's ready" {
		t.Error(s)
	}
	if s := spoken(Message{Text: "Hi"}); s != "Hi" {
		t.Error(s)
	}
}
