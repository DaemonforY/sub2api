package service

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Activation reminder: one email, once, to people who signed up 24–72 hours ago and haven't made
// an API call yet — two ways to get going (draw on the canvas, or set up Codex), with links. Off until an admin turns it on in
// 后台「数据看板」.
const (
	SettingKeyActivationReminderEnabled = "activation_reminder_enabled"

	activationReminderAfter    = 24 * time.Hour
	activationReminderUntil    = 72 * time.Hour
	activationReminderBatch    = 50
	activationReminderInterval = time.Hour
	activationReminderSubject  = "3 分钟用上 HiveGPT：画第一张图，或接入 Codex"
	activationReminderCanvas   = "https://canvas.hivegpt.cn/"
)

// ActivationReminderUser is someone the reminder may go to.
type ActivationReminderUser struct {
	ID       int64
	Email    string
	Username string
}

type ActivationReminderRepository interface {
	// DueUsers: active non-admin users created in [from, to) with no API call and no reminder yet.
	DueUsers(ctx context.Context, from, to time.Time, limit int) ([]ActivationReminderUser, error)
	MarkSent(ctx context.Context, userID int64) error
	SentCount(ctx context.Context) (int64, error)
}

type activationReminderMailer interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

type ActivationReminderService struct {
	repo     ActivationReminderRepository
	settings SettingRepository
	mailer   activationReminderMailer
	now      func() time.Time
	mu       sync.Mutex
	stop     chan struct{}
	stopOnce sync.Once
}

func NewActivationReminderService(repo ActivationReminderRepository, settings SettingRepository, mailer activationReminderMailer) *ActivationReminderService {
	return &ActivationReminderService{repo: repo, settings: settings, mailer: mailer, now: time.Now, stop: make(chan struct{})}
}

// ProvideActivationReminderService starts the hourly run.
func ProvideActivationReminderService(repo ActivationReminderRepository, settings SettingRepository, email *EmailService) *ActivationReminderService {
	var mailer activationReminderMailer
	if email != nil {
		mailer = email
	}
	s := NewActivationReminderService(repo, settings, mailer)
	go func() {
		t := time.NewTicker(activationReminderInterval)
		defer t.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-t.C:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				if _, err := s.RunOnce(ctx); err != nil {
					logger.LegacyPrintf("service.activation_reminder", "[ActivationReminder] run failed: %v", err)
				}
				cancel()
			}
		}
	}()
	return s
}

func (s *ActivationReminderService) Stop() { s.stopOnce.Do(func() { close(s.stop) }) }

func (s *ActivationReminderService) Enabled(ctx context.Context) bool {
	if s == nil || s.settings == nil {
		return false
	}
	v, err := s.settings.GetValue(ctx, SettingKeyActivationReminderEnabled)
	return err == nil && v == "true"
}

func (s *ActivationReminderService) SetEnabled(ctx context.Context, on bool) error {
	return s.settings.Set(ctx, SettingKeyActivationReminderEnabled, strconv.FormatBool(on))
}

// RunOnce sends the reminders that are due. Returns how many went out.
func (s *ActivationReminderService) RunOnce(ctx context.Context) (int, error) {
	if s == nil || s.mailer == nil || !s.Enabled(ctx) {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	users, err := s.repo.DueUsers(ctx, now.Add(-activationReminderUntil), now.Add(-activationReminderAfter), activationReminderBatch)
	if err != nil {
		return 0, err
	}
	base := s.siteURL(ctx)
	sent := 0
	for _, u := range users {
		if !reminderAddressOK(u.Email) {
			_ = s.repo.MarkSent(ctx, u.ID) // never retry placeholder addresses
			continue
		}
		if err := s.mailer.SendEmail(ctx, u.Email, activationReminderSubject, ActivationReminderBody(base, u.Username)); err != nil {
			// Mail server trouble: stop here and try the rest next hour.
			logger.LegacyPrintf("service.activation_reminder", "[ActivationReminder] send to user %d failed: %v", u.ID, err)
			return sent, nil
		}
		if err := s.repo.MarkSent(ctx, u.ID); err != nil {
			return sent, err
		}
		sent++
	}
	if sent > 0 {
		logger.LegacyPrintf("service.activation_reminder", "[ActivationReminder] sent %d reminders", sent)
	}
	return sent, nil
}

// Status for the admin page.
type ActivationReminderStatus struct {
	Enabled bool   `json:"enabled"`
	Sent    int64  `json:"sent"`
	Due     int    `json:"due"` // would go out on the next run
	Subject string `json:"subject"`
	Preview string `json:"preview"` // the email's HTML
}

func (s *ActivationReminderService) Status(ctx context.Context) (*ActivationReminderStatus, error) {
	sent, err := s.repo.SentCount(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()
	due, err := s.repo.DueUsers(ctx, now.Add(-activationReminderUntil), now.Add(-activationReminderAfter), activationReminderBatch)
	if err != nil {
		return nil, err
	}
	n := 0
	for _, u := range due {
		if reminderAddressOK(u.Email) {
			n++
		}
	}
	return &ActivationReminderStatus{
		Enabled: s.Enabled(ctx), Sent: sent, Due: n, Subject: activationReminderSubject,
		Preview: ActivationReminderBody(s.siteURL(ctx), ""),
	}, nil
}

func (s *ActivationReminderService) siteURL(ctx context.Context) string {
	if s.settings != nil {
		if v, err := s.settings.GetValue(ctx, SettingKeyFrontendURL); err == nil && strings.HasPrefix(v, "http") {
			return strings.TrimRight(v, "/")
		}
	}
	return "https://hivegpt.cn"
}

// OAuth sign-ups without a mailbox get placeholder addresses; skip anything that isn't real.
func reminderAddressOK(email string) bool {
	e := strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(e, "@")
	if at <= 0 || at == len(e)-1 {
		return false
	}
	domain := e[at+1:]
	for _, bad := range []string{".invalid", ".local", ".localhost", ".test", ".example"} {
		if strings.HasSuffix(domain, bad) {
			return false
		}
	}
	return strings.Contains(domain, ".")
}

// ActivationReminderBody is the reminder email (HTML).
func ActivationReminderBody(base, username string) string {
	hello := "你好"
	if name := strings.TrimSpace(username); name != "" {
		hello = "你好，" + html.EscapeString(name)
	}
	link := func(path, utm string) string {
		return html.EscapeString(fmt.Sprintf("%s%s%sutm_source=email&utm_medium=activation&utm_campaign=%s", base, path, sepFor(path), utm))
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="margin:0;padding:20px;background:#f5f5f7;font-family:-apple-system,BlinkMacSystemFont,'PingFang SC','Microsoft YaHei',sans-serif;color:#1f2937">
<div style="max-width:600px;margin:0 auto;background:#fff;border-radius:12px;overflow:hidden">
  <div style="background:linear-gradient(135deg,#6366f1,#8b5cf6);color:#fff;padding:24px 28px">
    <div style="font-size:20px;font-weight:700">HiveGPT</div>
    <div style="font-size:14px;opacity:.9;margin-top:4px">3 分钟用上第一次</div>
  </div>
  <div style="padding:24px 28px;font-size:15px;line-height:1.75">
    <p>%s：</p>
    <p>你前两天注册了 HiveGPT，还没有用过。按你想做的事选一条，大约 3 分钟：</p>
    <p style="margin-bottom:4px"><b>🎨 想用 AI 画图</b></p>
    <p style="margin-top:0">打开<a href="%s" style="color:#4f46e5">无限画布</a>，用 HiveGPT 账号一键连接，输入一句话就能出图，不用配置任何东西。</p>
    <p style="margin-bottom:4px"><b>💻 想在 Codex 里写代码</b></p>
    <ol style="padding-left:20px;margin-top:0">
      <li><b>创建一个 API Key</b>：<a href="%s" style="color:#4f46e5">去创建</a></li>
      <li><b>复制接入配置</b>：在 Key 的「使用」里选 Codex，按步骤复制到电脑上。<a href="%s" style="color:#4f46e5">查看接入配置</a></li>
      <li><b>发出第一次请求</b>：在终端运行 codex，随便问一句；控制台的「三步开始使用」会自动打勾。详细步骤见 <a href="%s" style="color:#4f46e5">Codex 接入教程</a>。</li>
    </ol>
    <p>账户余额为 0 且没有订阅时，画图和调用都会失败，可以先<a href="%s" style="color:#4f46e5">充值或购买订阅</a>。</p>
    <p>遇到问题，可以在首页右下角问智能客服。</p>
    <p style="text-align:center;margin:28px 0 8px"><a href="%s" style="display:inline-block;background:#4f46e5;color:#fff;text-decoration:none;padding:10px 22px;border-radius:8px;font-weight:600">打开控制台</a></p>
  </div>
  <div style="background:#f9fafb;padding:14px 28px;font-size:12px;color:#9ca3af;line-height:1.6">
    你收到这封邮件，是因为用这个邮箱注册了 HiveGPT。上手提醒只发这一封，不会再发。
  </div>
</div>
</body></html>`, hello, html.EscapeString(activationReminderCanvas+"?utm_source=email&utm_medium=activation&utm_campaign=canvas"),
		link("/keys?action=create", "key"), link("/keys?action=use", "config"), link("/learn/codex/hivegpt", "tutorial"),
		link("/purchase", "purchase"), link("/dashboard", "dashboard"))
}

func sepFor(path string) string {
	if strings.Contains(path, "?") {
		return "&"
	}
	return "?"
}
