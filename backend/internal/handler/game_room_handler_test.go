//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newGameRoomServer(t *testing.T) (*GameRoomHandler, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewGameRoomHandler()
	r := gin.New()
	r.GET("/ws", h.ServeWS)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return h, "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
}

func dialGameRoom(t *testing.T, url string) *coderws.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _, err := coderws.Dial(ctx, url, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.CloseNow() })
	return c
}

func readGameRoom(t *testing.T, c *coderws.Conn) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, data, err := c.Read(ctx)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func writeGameRoom(t *testing.T, c *coderws.Conn, s string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	require.NoError(t, c.Write(ctx, coderws.MessageText, []byte(s)))
}

// createAndJoin opens a room as host and joins it as guest, consuming the handshake messages.
func createAndJoin(t *testing.T, url string) (host, guest *coderws.Conn, code string) {
	t.Helper()
	host = dialGameRoom(t, url+"?game=snow")
	m := readGameRoom(t, host)
	require.Equal(t, "room", m["sys"])
	require.EqualValues(t, 0, m["seat"])
	code = m["code"].(string)
	require.Regexp(t, `^[A-Z2-9]{4}$`, code)

	guest = dialGameRoom(t, url+"?game=snow&room="+strings.ToLower(code))
	m = readGameRoom(t, guest)
	require.Equal(t, "room", m["sys"])
	require.EqualValues(t, 1, m["seat"])
	require.Equal(t, code, m["code"])
	require.Equal(t, map[string]any{"sys": "peer", "on": true}, readGameRoom(t, guest))
	require.Equal(t, map[string]any{"sys": "peer", "on": true}, readGameRoom(t, host))
	return host, guest, code
}

func TestGameRoom_RelaysBothWaysAndDropsSysMessages(t *testing.T) {
	_, url := newGameRoomServer(t)
	host, guest, _ := createAndJoin(t, url)

	writeGameRoom(t, guest, `{"sys":"room","code":"FAKE"}`) // spoofed control message: dropped
	writeGameRoom(t, guest, `[1,2]`)                        // not an object: dropped
	writeGameRoom(t, guest, `{"in":{"left":true},"f":12}`)
	require.Equal(t, map[string]any{"in": map[string]any{"left": true}, "f": float64(12)}, readGameRoom(t, host))

	writeGameRoom(t, host, `{"snap":[1,2,3]}`)
	require.Equal(t, map[string]any{"snap": []any{float64(1), float64(2), float64(3)}}, readGameRoom(t, guest))
}

func TestGameRoom_HostLeavingEndsRoom(t *testing.T) {
	h, url := newGameRoomServer(t)
	host, guest, code := createAndJoin(t, url)

	require.NoError(t, host.Close(coderws.StatusNormalClosure, ""))
	require.Equal(t, map[string]any{"sys": "peer", "on": false}, readGameRoom(t, guest))
	require.Equal(t, map[string]any{"sys": "error", "code": "host_left"}, readGameRoom(t, guest))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _, err := guest.Read(ctx) // a client keeps reading, which completes the close handshake
	require.Equal(t, coderws.StatusNormalClosure, coderws.CloseStatus(err))

	require.Eventually(t, func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.rooms[code] == nil && len(h.perIP) == 0
	}, 3*time.Second, 20*time.Millisecond)

	late := dialGameRoom(t, url+"?game=snow&room="+code)
	require.Equal(t, map[string]any{"sys": "error", "code": "not_found"}, readGameRoom(t, late))
}

func TestGameRoom_GuestCanRejoinAndThirdIsRejected(t *testing.T) {
	_, url := newGameRoomServer(t)
	host, guest, code := createAndJoin(t, url)

	third := dialGameRoom(t, url+"?game=snow&room="+code)
	require.Equal(t, map[string]any{"sys": "error", "code": "room_full"}, readGameRoom(t, third))

	require.NoError(t, guest.Close(coderws.StatusNormalClosure, ""))
	require.Equal(t, map[string]any{"sys": "peer", "on": false}, readGameRoom(t, host))

	again := dialGameRoom(t, url+"?game=snow&room="+code)
	require.Equal(t, "room", readGameRoom(t, again)["sys"])
	require.Equal(t, map[string]any{"sys": "peer", "on": true}, readGameRoom(t, again))
	require.Equal(t, map[string]any{"sys": "peer", "on": true}, readGameRoom(t, host))
}

func TestGameRoom_RejectsBadInput(t *testing.T) {
	_, url := newGameRoomServer(t)

	bad := dialGameRoom(t, url+"?game=Snow!")
	require.Equal(t, map[string]any{"sys": "error", "code": "bad_game"}, readGameRoom(t, bad))

	host := dialGameRoom(t, url+"?game=snow")
	code := readGameRoom(t, host)["code"].(string)
	other := dialGameRoom(t, url+"?game=island&room="+code) // right code, wrong game
	require.Equal(t, map[string]any{"sys": "error", "code": "not_found"}, readGameRoom(t, other))

	writeGameRoom(t, host, `{"pad":"`+strings.Repeat("x", 9000)+`"}`)
	require.Equal(t, map[string]any{"sys": "error", "code": "too_big"}, readGameRoom(t, host))
}

func TestGameRoom_PerIPCap(t *testing.T) {
	h, url := newGameRoomServer(t)
	h.maxConnsIP = 2
	for i := 0; i < 2; i++ {
		c := dialGameRoom(t, url+"?game=snow")
		require.Equal(t, "room", readGameRoom(t, c)["sys"])
	}
	c := dialGameRoom(t, url+"?game=snow")
	require.Equal(t, map[string]any{"sys": "error", "code": "rate"}, readGameRoom(t, c))
}
