//go:build unit

package service

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSendEmailSkipsPlaceholderRecipientsWithoutDialing(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	var accepted atomic.Int32
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			accepted.Add(1)
			_ = conn.Close()
		}
	}()
	cfg := &SMTPConfig{Host: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, From: "noreply@hivegpt.cn"}
	svc := &EmailService{}

	for _, to := range []string{
		"wechat-o0po06abc@wechat-connect.invalid",
		"WeChat User <wechat-o0po06abc@WECHAT-CONNECT.INVALID>",
		"linuxdo-123@linuxdo-connect.invalid",
		"someone@whatever.invalid",
	} {
		require.NoError(t, svc.SendEmailWithConfig(cfg, to, "hi", "<p>hi</p>"), to)
	}
	time.Sleep(50 * time.Millisecond)
	require.Zero(t, accepted.Load(), "no SMTP connection for placeholder recipients")

	// A real address does dial (and fails against this fake, which hangs up at once).
	require.Error(t, svc.SendEmailWithConfig(cfg, "user@example.com", "hi", "<p>hi</p>"))
	require.Eventually(t, func() bool { return accepted.Load() == 1 }, time.Second, 10*time.Millisecond)

	for addr, want := range map[string]bool{
		"a@wechat-connect.invalid": true, "a@x.invalid": true, "a@invalid": true,
		"a@example.com": false, "a@invalid.com": false, "a@hivegpt.cn": false,
	} {
		_, got := reservedEmailRecipient(addr)
		require.Equal(t, want, got, addr)
	}
}
