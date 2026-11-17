package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

// GameRoomHandler is the 联机双排 relay for the 小游戏: two players share a room code and every JSON
// object one sends is passed verbatim to the other. The games run host-authoritative netcode on top
// (seat 0 simulates, seat 1 sends inputs), so the server never looks inside the game messages.
//
//	GET /api/v1/games/rooms/ws?game=<slug>              create a room (you are seat 0, the host)
//	GET /api/v1/games/rooms/ws?game=<slug>&room=<CODE>  join it (seat 1)
//
// Server messages carry a top-level "sys" key, which game messages must not use:
//
//	{"sys":"room","code":"K7QX","seat":0}   after connecting
//	{"sys":"peer","on":true|false}          the other player joined / left
//	{"sys":"error","code":"…"}              then the socket closes
//
// When the host leaves the room ends; a guest who drops can rejoin with the same code. Rooms live in
// memory (prod runs one instance), so a deploy ends every room.
type GameRoomHandler struct {
	mu    sync.Mutex
	rooms map[string]*gameRoom
	perIP map[string]int

	maxRooms     int
	maxConnsIP   int
	maxMsgBytes  int
	ratePerSec   float64
	rateBurst    float64
	maxLifetime  time.Duration
	pingInterval time.Duration
}

type gameRoom struct {
	code  string
	game  string
	seats [2]*gameRoomConn
}

type gameRoomConn struct {
	conn       *coderws.Conn
	ip         string
	mu         sync.Mutex
	send       chan []byte
	closed     bool          // send is closed; nothing more is queued
	writerDone chan struct{} // closed when writeLoop returns
}

const gameRoomCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789" // no I/L/O/0/1

var (
	gameRoomSlugRe = regexp.MustCompile(`^[a-z0-9-]{1,24}$`)
	gameRoomCodeRe = regexp.MustCompile(`^[A-Z0-9]{4}$`)
)

func NewGameRoomHandler() *GameRoomHandler {
	return &GameRoomHandler{
		rooms:        map[string]*gameRoom{},
		perIP:        map[string]int{},
		maxRooms:     2000,
		maxConnsIP:   8,
		maxMsgBytes:  8 << 10,
		ratePerSec:   90,
		rateBurst:    180,
		maxLifetime:  3 * time.Hour,
		pingInterval: 25 * time.Second,
	}
}

func gameRoomSys(kind string, fields map[string]any) []byte {
	m := map[string]any{"sys": kind}
	for k, v := range fields {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	return b
}

// ServeWS upgrades the request and relays messages until either side leaves.
func (h *GameRoomHandler) ServeWS(c *gin.Context) {
	game := c.Query("game")
	code := strings.ToUpper(strings.TrimSpace(c.Query("room")))
	clientIP := ip.GetClientIP(c)

	conn, err := coderws.Accept(c.Writer, c.Request, &coderws.AcceptOptions{
		// Same-origin by default; local dev serves the page from vite on another port.
		OriginPatterns: []string{"localhost:*", "127.0.0.1:*"},
	})
	if err != nil {
		return
	}
	conn.SetReadLimit(int64(h.maxMsgBytes) * 2)
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.maxLifetime)
	defer cancel()

	fail := func(reason string) {
		wctx, wcancel := context.WithTimeout(ctx, 2*time.Second)
		_ = conn.Write(wctx, coderws.MessageText, gameRoomSys("error", map[string]any{"code": reason}))
		wcancel()
		_ = conn.Close(coderws.StatusPolicyViolation, reason)
	}
	if !gameRoomSlugRe.MatchString(game) {
		fail("bad_game")
		return
	}
	if code != "" && !gameRoomCodeRe.MatchString(code) {
		fail("not_found")
		return
	}

	me := &gameRoomConn{conn: conn, send: make(chan []byte, 256), ip: clientIP, writerDone: make(chan struct{})}
	room, seat, reason := h.join(game, code, me)
	if reason != "" {
		fail(reason)
		return
	}
	defer h.leave(room, seat, me)

	go h.writeLoop(ctx, me)
	me.enqueue(gameRoomSys("room", map[string]any{"code": room.code, "seat": seat}))
	if peer := h.peerOf(room, seat); peer != nil {
		me.enqueue(gameRoomSys("peer", map[string]any{"on": true}))
		peer.enqueue(gameRoomSys("peer", map[string]any{"on": true}))
	}
	h.readLoop(ctx, room, seat, me)
}

func (h *GameRoomHandler) join(game, code string, me *gameRoomConn) (*gameRoom, int, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.perIP[me.ip] >= h.maxConnsIP {
		return nil, 0, "rate"
	}
	if code == "" {
		if len(h.rooms) >= h.maxRooms {
			return nil, 0, "rate"
		}
		for i := 0; i < 20; i++ {
			cand := randomGameRoomCode()
			if _, taken := h.rooms[cand]; !taken {
				code = cand
				break
			}
		}
		if code == "" {
			return nil, 0, "rate"
		}
		room := &gameRoom{code: code, game: game}
		room.seats[0] = me
		h.rooms[code] = room
		h.perIP[me.ip]++
		return room, 0, ""
	}
	room := h.rooms[code]
	if room == nil || room.game != game || room.seats[0] == nil {
		return nil, 0, "not_found"
	}
	if room.seats[1] != nil {
		return nil, 0, "room_full"
	}
	room.seats[1] = me
	h.perIP[me.ip]++
	return room, 1, ""
}

func (h *GameRoomHandler) leave(room *gameRoom, seat int, me *gameRoomConn) {
	h.mu.Lock()
	if room.seats[seat] == me {
		room.seats[seat] = nil
	}
	if h.perIP[me.ip]--; h.perIP[me.ip] <= 0 {
		delete(h.perIP, me.ip)
	}
	var peer *gameRoomConn
	if seat == 0 {
		// The host simulates the game, so the room ends with it.
		if h.rooms[room.code] == room {
			delete(h.rooms, room.code)
		}
		peer = room.seats[1]
	} else {
		peer = room.seats[0]
	}
	h.mu.Unlock()

	// Let the writer flush what is queued (an error message, say), then drop the socket.
	me.closeAfterFlush()
	select {
	case <-me.writerDone:
	case <-time.After(2 * time.Second):
	}
	_ = me.conn.CloseNow()
	if peer != nil {
		peer.enqueue(gameRoomSys("peer", map[string]any{"on": false}))
		if seat == 0 {
			peer.enqueue(gameRoomSys("error", map[string]any{"code": "host_left"}))
			peer.closeAfterFlush()
		}
	}
}

func (h *GameRoomHandler) peerOf(room *gameRoom, seat int) *gameRoomConn {
	h.mu.Lock()
	defer h.mu.Unlock()
	return room.seats[1-seat]
}

func (h *GameRoomHandler) readLoop(ctx context.Context, room *gameRoom, seat int, me *gameRoomConn) {
	tokens, last := h.rateBurst, time.Now()
	for {
		typ, data, err := me.conn.Read(ctx)
		if err != nil {
			return
		}
		now := time.Now()
		tokens += now.Sub(last).Seconds() * h.ratePerSec
		if tokens > h.rateBurst {
			tokens = h.rateBurst
		}
		last = now
		if tokens < 1 {
			me.enqueue(gameRoomSys("error", map[string]any{"code": "rate"}))
			me.closeAfterFlush()
			return
		}
		tokens--
		if len(data) > h.maxMsgBytes {
			me.enqueue(gameRoomSys("error", map[string]any{"code": "too_big"}))
			me.closeAfterFlush()
			return
		}
		if typ != coderws.MessageText || !isGameMessage(data) {
			continue // ignore anything that isn't a plain JSON object without "sys"
		}
		if peer := h.peerOf(room, seat); peer != nil {
			peer.enqueue(data)
		}
	}
}

func isGameMessage(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) < 2 || trimmed[0] != '{' {
		return false
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(trimmed, &obj) != nil {
		return false
	}
	_, hasSys := obj["sys"]
	return !hasSys
}

func (h *GameRoomHandler) writeLoop(ctx context.Context, me *gameRoomConn) {
	defer close(me.writerDone)
	ping := time.NewTicker(h.pingInterval)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-me.send:
			if !ok {
				_ = me.conn.Close(coderws.StatusNormalClosure, "")
				return
			}
			wctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := me.conn.Write(wctx, coderws.MessageText, msg)
			cancel()
			if err != nil {
				_ = me.conn.CloseNow()
				return
			}
		case <-ping.C:
			pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := me.conn.Ping(pctx)
			cancel()
			if err != nil {
				_ = me.conn.CloseNow()
				return
			}
		}
	}
}

// enqueue never blocks: a client too slow to drain 256 queued messages is cut off.
func (c *gameRoomConn) enqueue(msg []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	select {
	case c.send <- msg:
	default:
		c.closed = true
		close(c.send)
		_ = c.conn.CloseNow()
	}
}

// closeAfterFlush lets the writer send what is queued, then close the socket.
func (c *gameRoomConn) closeAfterFlush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

func randomGameRoomCode() string {
	b := make([]byte, 4)
	n := big.NewInt(int64(len(gameRoomCodeAlphabet)))
	for i := range b {
		v, err := rand.Int(rand.Reader, n)
		if err != nil {
			return ""
		}
		b[i] = gameRoomCodeAlphabet[v.Int64()]
	}
	return string(b)
}
