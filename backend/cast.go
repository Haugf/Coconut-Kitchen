package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// The Google speaker, over the Cast protocol it already speaks on the home
// network: the mirror keeps one connection open to see what's playing,
// and opens another to say a message out loud. Nothing here talks to
// Google's cloud except the voice for spoken messages (see voiceURL).

type CastConfig struct {
	// The speaker's name in the Google Home app ("Kitchen speaker").
	// Empty means the first Google speaker that answers.
	Name string `json:"name"`
	// Optional fixed address (192.168.1.20 or 192.168.1.20:8009) for
	// networks where discovery doesn't work. Never commit it.
	Host string `json:"host"`
	// Turns the speaker off entirely: no now-playing, no spoken messages.
	Off bool `json:"off"`
	// The voice for spoken messages: a URL the speaker fetches as MP3,
	// with {lang} and {text} filled in. Defaults to Google Translate's.
	VoiceURL string `json:"voiceUrl"`
	Language string `json:"language"`
}

const defaultVoiceURL = "https://translate.google.com/translate_tts?ie=UTF-8&client=tw-ob&tl={lang}&q={text}"

func defaultCast() CastConfig { return CastConfig{VoiceURL: defaultVoiceURL, Language: "en"} }

// NowPlaying is what /api/cast serves. It never carries the speaker's
// address.
type NowPlaying struct {
	Speaker   string `json:"speaker,omitempty"`
	Connected bool   `json:"connected"`
	// Empty when nothing worth showing is playing.
	State    string    `json:"state,omitempty"` // playing, paused, buffering
	App      string    `json:"app,omitempty"`   // "Spotify", "TuneIn Radio", …
	Title    string    `json:"title,omitempty"`
	Subtitle string    `json:"subtitle,omitempty"`
	Live     bool      `json:"live,omitempty"`
	Duration float64   `json:"duration,omitempty"` // seconds
	Position float64   `json:"position,omitempty"` // seconds, as of At
	At       time.Time `json:"at"`
}

const (
	nsConnection = "urn:x-cast:com.google.cast.tp.connection"
	nsHeartbeat  = "urn:x-cast:com.google.cast.tp.heartbeat"
	nsReceiver   = "urn:x-cast:com.google.cast.receiver"
	nsMedia      = "urn:x-cast:com.google.cast.media"

	appDefaultMedia = "CC1AD845" // the plain player spoken messages use
	appBackdrop     = "E8C28D3C" // the idle screen on displays

	castSender = "sender-0"
	castPort   = "8009"
)

// How long a paused song stays on the mirror before it steps aside.
const pausedShowFor = 5 * time.Minute

type caster struct {
	cfg CastConfig

	mu        sync.Mutex
	device    castDevice
	connected bool
	st        castState
	lastErr   error
	errAt     time.Time
	// Our own spoken message plays through the speaker's plain player;
	// it shouldn't show up as "now playing".
	speakingUntil time.Time
}

type castState struct {
	appID, app, statusText string
	player                 string // PLAYING, PAUSED, BUFFERING, IDLE
	title, subtitle        string
	live                   bool
	duration, position     float64
	at, pausedAt           time.Time
}

func newCaster(cfg CastConfig) *caster {
	d := defaultCast()
	if cfg.VoiceURL == "" {
		cfg.VoiceURL = d.VoiceURL
	}
	if cfg.Language == "" {
		cfg.Language = d.Language
	}
	return &caster{cfg: cfg}
}

// run keeps a connection to the speaker for as long as the server runs,
// reconnecting with a growing pause when the speaker is off or away.
func (c *caster) run(ctx context.Context) {
	wait := 5 * time.Second
	for ctx.Err() == nil {
		started := time.Now()
		err := c.watch(ctx)
		c.mu.Lock()
		if c.connected {
			log.Printf("speaker: disconnected from %s: %v", c.device.Name, err)
		}
		c.connected = false
		c.st = castState{}
		c.lastErr, c.errAt = err, time.Now()
		c.mu.Unlock()
		if time.Since(started) > time.Minute {
			wait = 5 * time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		if wait *= 2; wait > 2*time.Minute {
			wait = 2 * time.Minute
		}
	}
}

// find returns the configured speaker, by fixed address or by name.
func (c *caster) find(ctx context.Context) (castDevice, error) {
	if h := strings.TrimSpace(c.cfg.Host); h != "" {
		if _, _, err := net.SplitHostPort(h); err != nil {
			h = net.JoinHostPort(h, castPort)
		}
		name := c.cfg.Name
		if name == "" {
			name = "speaker"
		}
		return castDevice{Name: name, Addr: h}, nil
	}
	found, err := discoverCast(ctx, 3*time.Second)
	if err != nil {
		return castDevice{}, fmt.Errorf("looking for speakers: %w", err)
	}
	return pickDevice(found, c.cfg.Name)
}

func pickDevice(found []castDevice, name string) (castDevice, error) {
	for _, d := range found {
		if name == "" || strings.EqualFold(strings.TrimSpace(d.Name), strings.TrimSpace(name)) {
			return d, nil
		}
	}
	if name == "" {
		return castDevice{}, errors.New("no Google speaker answered on this network")
	}
	names := []string{}
	for _, d := range found {
		names = append(names, fmt.Sprintf("%q", d.Name))
	}
	if len(names) == 0 {
		return castDevice{}, fmt.Errorf("%q didn't answer, and no other speakers did either", name)
	}
	return castDevice{}, fmt.Errorf("%q didn't answer; found %s", name, strings.Join(names, ", "))
}

// watch holds one connection open and follows the speaker's status until
// it drops.
func (c *caster) watch(ctx context.Context) error {
	dev, err := c.find(ctx)
	if err != nil {
		return err
	}
	cc, err := dialCast(ctx, dev.Addr)
	if err != nil {
		return err
	}
	defer cc.Close()

	c.mu.Lock()
	c.device, c.connected, c.lastErr = dev, true, nil
	c.mu.Unlock()
	log.Printf("speaker: connected to %s (%s)", dev.Name, orDash(dev.Model))

	if err := cc.connect("receiver-0"); err != nil {
		return err
	}
	if _, err := cc.send("receiver-0", nsReceiver, map[string]any{"type": "GET_STATUS"}); err != nil {
		return err
	}

	// The media session we're following, if any.
	var media atomic.Value
	media.Store("")

	done := make(chan struct{})
	defer close(done)
	go func() {
		ping := time.NewTicker(5 * time.Second)
		defer ping.Stop()
		for n := 1; ; n++ {
			select {
			case <-done:
				return
			case <-ctx.Done():
				cc.Close()
				return
			case <-ping.C:
			}
			_, _ = cc.send("receiver-0", nsHeartbeat, map[string]any{"type": "PING"})
			// The speaker announces changes itself; asking every 30
			// seconds catches a radio station's next song if it doesn't.
			if t := media.Load().(string); t != "" && n%6 == 0 {
				_, _ = cc.send(t, nsMedia, map[string]any{"type": "GET_STATUS"})
			}
		}
	}()

	for {
		m, err := cc.read(30 * time.Second)
		if err != nil {
			return err
		}
		p := parsePayload(m.Payload)
		switch m.Namespace {
		case nsHeartbeat:
			if p.Type == "PING" {
				_, _ = cc.send(m.Source, nsHeartbeat, map[string]any{"type": "PONG"})
			}
		case nsReceiver:
			if p.Type != "RECEIVER_STATUS" {
				continue
			}
			app := mainApp(p.Status)
			c.applyApp(app, time.Now())
			if app != nil && app.hasMedia() && app.TransportID != media.Load().(string) {
				media.Store(app.TransportID)
				if err := cc.connect(app.TransportID); err != nil {
					return err
				}
				if _, err := cc.send(app.TransportID, nsMedia, map[string]any{"type": "GET_STATUS"}); err != nil {
					return err
				}
			} else if app == nil || !app.hasMedia() {
				media.Store("")
			}
		case nsMedia:
			if p.Type == "MEDIA_STATUS" {
				c.applyMedia(p.Status, time.Now())
			}
		case nsConnection:
			if p.Type == "CLOSE" && m.Source == media.Load().(string) {
				media.Store("")
			}
		}
	}
}

// snapshot is what the mirror shows now.
func (c *caster) snapshot(now time.Time) NowPlaying {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := NowPlaying{Speaker: c.device.Name, Connected: c.connected, At: now}
	st := c.st
	if !c.connected || st.app == "" {
		return out
	}
	if st.appID == appDefaultMedia && now.Before(c.speakingUntil) {
		return out
	}
	title := st.title
	if title == "" {
		title = st.statusText
	}
	if title == "" {
		return out
	}
	switch st.player {
	case "PLAYING", "BUFFERING":
		out.State = strings.ToLower(st.player)
	case "PAUSED":
		if now.Sub(st.pausedAt) > pausedShowFor {
			return out
		}
		out.State = "paused"
	default:
		return out
	}
	out.App, out.Title, out.Subtitle, out.Live = st.app, title, st.subtitle, st.live
	out.Duration = st.duration
	out.Position = st.position
	if st.player == "PLAYING" && !st.at.IsZero() {
		out.Position += now.Sub(st.at).Seconds()
	}
	if out.Duration > 0 && out.Position > out.Duration {
		out.Position = out.Duration
	}
	return out
}

func (c *caster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, c.snapshot(time.Now()))
}

// health is for /api/status: whether the speaker is reachable and
// whether something is playing, never what.
func (c *caster) health() map[string]any {
	np := c.snapshot(time.Now())
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]any{"connected": c.connected, "speaker": c.device.Name, "model": c.device.Model, "playing": np.State}
	if c.lastErr != nil && !c.connected {
		out["error"], out["errorAt"] = c.lastErr.Error(), c.errAt
	}
	return out
}

type castApp struct {
	AppID        string `json:"appId"`
	DisplayName  string `json:"displayName"`
	StatusText   string `json:"statusText"`
	TransportID  string `json:"transportId"`
	SessionID    string `json:"sessionId"`
	IsIdleScreen bool   `json:"isIdleScreen"`
	Namespaces   []struct {
		Name string `json:"name"`
	} `json:"namespaces"`
}

func (a *castApp) hasMedia() bool {
	for _, n := range a.Namespaces {
		if n.Name == nsMedia {
			return a.TransportID != ""
		}
	}
	return false
}

// mainApp is the app the speaker is running, ignoring idle screens.
func mainApp(raw json.RawMessage) *castApp {
	var st struct {
		Applications []castApp `json:"applications"`
	}
	if json.Unmarshal(raw, &st) != nil {
		return nil
	}
	for i := range st.Applications {
		a := &st.Applications[i]
		if !a.IsIdleScreen && a.AppID != appBackdrop {
			return a
		}
	}
	return nil
}

func (c *caster) applyApp(app *castApp, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if app == nil {
		c.st = castState{}
		return
	}
	if app.AppID != c.st.appID {
		c.st = castState{appID: app.AppID}
	}
	c.st.app = strings.TrimSpace(app.DisplayName)
	c.st.statusText = cleanStatusText(app.StatusText, c.st.app)
}

// cleanStatusText drops the generic lines apps show when idle.
func cleanStatusText(s, app string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "", "ready to cast", "default media receiver", strings.ToLower(app):
		return ""
	}
	for _, p := range []string{"Now Casting: ", "Casting: ", "Now Playing: "} {
		s = strings.TrimPrefix(s, p)
	}
	return s
}

type mediaStatus struct {
	PlayerState string   `json:"playerState"`
	CurrentTime *float64 `json:"currentTime"`
	Media       *struct {
		StreamType string   `json:"streamType"`
		Duration   *float64 `json:"duration"`
		Metadata   struct {
			Title       string `json:"title"`
			Subtitle    string `json:"subtitle"`
			Artist      string `json:"artist"`
			AlbumArtist string `json:"albumArtist"`
			AlbumName   string `json:"albumName"`
		} `json:"metadata"`
	} `json:"media"`
}

func (c *caster) applyMedia(raw json.RawMessage, now time.Time) {
	var list []mediaStatus
	if json.Unmarshal(raw, &list) != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(list) == 0 {
		c.st.player = "IDLE"
		return
	}
	m := list[0]
	if m.PlayerState != "" {
		if m.PlayerState == "PAUSED" && c.st.player != "PAUSED" {
			c.st.pausedAt = now
		}
		c.st.player = m.PlayerState
	}
	if m.CurrentTime != nil {
		c.st.position, c.st.at = *m.CurrentTime, now
	}
	// Later updates often leave out the media block when only the
	// position or state changed; keep what we had.
	if m.Media != nil {
		md := m.Media.Metadata
		c.st.title = strings.TrimSpace(md.Title)
		c.st.subtitle = firstNonEmpty(md.Artist, md.AlbumArtist, md.Subtitle, md.AlbumName)
		c.st.live = m.Media.StreamType == "LIVE"
		c.st.duration = 0
		if m.Media.Duration != nil && *m.Media.Duration > 0 && !c.st.live {
			c.st.duration = *m.Media.Duration
		}
	}
}

// say speaks text through the speaker. It stops whatever is playing;
// the speaker's plain player can't mix over music.
func (c *caster) say(ctx context.Context, text string) error {
	c.mu.Lock()
	dev, ok := c.device, c.connected
	c.mu.Unlock()
	if !ok {
		var err error
		if dev, err = c.find(ctx); err != nil {
			return err
		}
	}
	cc, err := dialCast(ctx, dev.Addr)
	if err != nil {
		return err
	}
	defer cc.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = cc.conn.SetDeadline(dl)
	}

	c.mu.Lock()
	c.speakingUntil = time.Now().Add(45 * time.Second)
	c.mu.Unlock()

	if err := cc.connect("receiver-0"); err != nil {
		return err
	}
	if _, err := cc.send("receiver-0", nsReceiver, map[string]any{"type": "LAUNCH", "appId": appDefaultMedia}); err != nil {
		return err
	}
	var transport string
	for transport == "" {
		m, err := cc.read(15 * time.Second)
		if err != nil {
			return fmt.Errorf("waiting for the speaker's player: %w", err)
		}
		p := parsePayload(m.Payload)
		switch {
		case m.Namespace == nsHeartbeat && p.Type == "PING":
			_, _ = cc.send(m.Source, nsHeartbeat, map[string]any{"type": "PONG"})
		case p.Type == "LAUNCH_ERROR":
			return fmt.Errorf("the speaker wouldn't open its player: %s", orDash(p.Reason))
		case p.Type == "RECEIVER_STATUS":
			if a := mainApp(p.Status); a != nil && a.AppID == appDefaultMedia && a.TransportID != "" {
				transport = a.TransportID
			}
		}
	}
	if err := cc.connect(transport); err != nil {
		return err
	}
	if _, err := cc.send(transport, nsMedia, map[string]any{
		"type":     "LOAD",
		"autoplay": true,
		"media": map[string]any{
			"contentId":   voiceURL(c.cfg.VoiceURL, c.cfg.Language, text),
			"contentType": "audio/mpeg",
			"streamType":  "BUFFERED",
			"metadata":    map[string]any{"metadataType": 0, "title": "Coconut Kitchen"},
		},
	}); err != nil {
		return err
	}
	for {
		m, err := cc.read(15 * time.Second)
		if err != nil {
			return fmt.Errorf("waiting for the message to play: %w", err)
		}
		p := parsePayload(m.Payload)
		switch p.Type {
		case "PING":
			_, _ = cc.send(m.Source, nsHeartbeat, map[string]any{"type": "PONG"})
		case "LOAD_FAILED", "LOAD_CANCELLED", "INVALID_REQUEST":
			return fmt.Errorf("the speaker couldn't play the message (%s)", strings.ToLower(p.Type))
		case "MEDIA_STATUS":
			var list []mediaStatus
			if json.Unmarshal(p.Status, &list) == nil && len(list) > 0 {
				if s := list[0].PlayerState; s == "PLAYING" || s == "BUFFERING" {
					return nil
				}
			}
		}
	}
}

// voiceURL fills in the voice template. Google's voice reads at most 200
// characters, so longer messages are cut at a word.
func voiceURL(tmpl, lang, text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if r := []rune(text); len(r) > 200 {
		cut := string(r[:200])
		if i := strings.LastIndex(cut, " "); i > 120 {
			cut = cut[:i]
		}
		text = cut
	}
	return strings.NewReplacer("{lang}", url.QueryEscape(lang), "{text}", url.QueryEscape(text)).Replace(tmpl)
}

// The wire format: each message is a 4-byte length and a small protobuf
// (CastMessage) whose payload is JSON.

type castMsg struct {
	Source, Dest, Namespace, Payload string
}

func encodeCast(m castMsg) []byte {
	b := []byte{0x08, 0x00} // protocol_version = CASTV2_1_0
	b = appendPBString(b, 2, m.Source)
	b = appendPBString(b, 3, m.Dest)
	b = appendPBString(b, 4, m.Namespace)
	b = append(b, 0x28, 0x00) // payload_type = STRING
	b = appendPBString(b, 6, m.Payload)
	return append(binary.BigEndian.AppendUint32(nil, uint32(len(b))), b...)
}

func appendPBString(b []byte, field int, s string) []byte {
	b = binary.AppendUvarint(b, uint64(field<<3|2))
	b = binary.AppendUvarint(b, uint64(len(s)))
	return append(b, s...)
}

func decodeCast(b []byte) (castMsg, error) {
	var m castMsg
	err := eachField(b, func(field int, val []byte, _ uint64) error {
		switch field {
		case 2:
			m.Source = string(val)
		case 3:
			m.Dest = string(val)
		case 4:
			m.Namespace = string(val)
		case 6:
			m.Payload = string(val)
		}
		return nil
	})
	return m, err
}

type castPayload struct {
	Type   string          `json:"type"`
	Status json.RawMessage `json:"status"`
	Reason string          `json:"reason"`
}

func parsePayload(s string) castPayload {
	var p castPayload
	_ = json.Unmarshal([]byte(s), &p)
	return p
}

type castConn struct {
	conn net.Conn
	r    *bufio.Reader
	wmu  sync.Mutex
	id   atomic.Int64
}

// Speakers use a certificate of their own, not one a browser would
// trust, so there's nothing to verify against; the connection stays on
// the home network.
var castTLS = &tls.Config{InsecureSkipVerify: true} //nolint:gosec

func dialCast(ctx context.Context, addr string) (*castConn, error) {
	d := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 5 * time.Second}, Config: castTLS}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connecting to the speaker: %w", err)
	}
	return &castConn{conn: conn, r: bufio.NewReader(conn)}, nil
}

func (c *castConn) Close() error { return c.conn.Close() }

func (c *castConn) connect(dest string) error {
	_, err := c.send(dest, nsConnection, map[string]any{"type": "CONNECT"})
	return err
}

// send adds a request id (except to the connection and heartbeat
// channels, which don't take one) and writes the message.
func (c *castConn) send(dest, ns string, payload map[string]any) (int64, error) {
	var id int64
	if ns != nsConnection && ns != nsHeartbeat {
		id = c.id.Add(1)
		payload["requestId"] = id
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	c.wmu.Lock()
	defer c.wmu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = c.conn.Write(encodeCast(castMsg{Source: castSender, Dest: dest, Namespace: ns, Payload: string(b)}))
	return id, err
}

func (c *castConn) read(timeout time.Duration) (castMsg, error) {
	_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
	var size [4]byte
	if _, err := io.ReadFull(c.r, size[:]); err != nil {
		return castMsg{}, err
	}
	n := binary.BigEndian.Uint32(size[:])
	if n > 64<<10 {
		return castMsg{}, fmt.Errorf("cast: message of %d bytes is too large", n)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(c.r, b); err != nil {
		return castMsg{}, err
	}
	return decodeCast(b)
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

func orDash(s string) string {
	if s == "" {
		return "–"
	}
	return s
}
