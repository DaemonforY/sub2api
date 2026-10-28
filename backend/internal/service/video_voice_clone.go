package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 我的音色: users clone their own voice with 豆包声音复刻 2.0 and narrate their films with it. The
// operator buys voice slots (speaker IDs, S_xxx) in the 火山引擎 console and lists them in the video
// settings; a new clone takes a free slot, the recording goes to 豆包 for training and is not kept.
// The slot fee is taken from the balance once training succeeds (a failed training costs nothing);
// narration with a clone is billed per character like the paid preset voices. A project refers to a
// clone as "clone:<id>", never by speaker ID, because a deleted clone's slot is retrained for the
// next user: only the clone's owner can narrate with it, and only while it is ready.

const (
	VideoClonePrefix = "clone:"
	// Recording limits: 豆包 takes up to 10 MB and uses about the first 30 s of speech.
	VideoCloneMaxBytes   = 10 << 20
	videoCloneMinSeconds = 5.0
	videoCloneMaxSeconds = 40.0
	// Re-recordings per clone (each one retrains the slot; 豆包 allows 15 trainings per slot in all).
	videoCloneMaxTrainings = 5
	// How often a training clone's status is asked for while the user waits.
	videoClonePollEvery = 4 * time.Second
	// Free previews per user per day (a sentence each).
	videoClonePreviewDaily = 30
	// VideoCloneConsentText is what the user agrees to; it is stored with the clone.
	VideoCloneConsentText = "我确认：这段录音是我本人的声音，或我已取得声音所有者本人的单独同意；复刻的音色只用于我自己的作品，不会用来冒充他人、误导他人或从事违法活动。"
	// VideoCloneScript is read aloud for the recording. Its first sentence repeats the consent, so the
	// recording itself carries the speaker's agreement.
	VideoCloneScript = "我自愿用我的声音创建一个 AI 音色，只用在我自己的作品里。今天天气不错，我想用这段话录下自己的声音：说话的时候放松一点，像平时和朋友聊天一样，有快有慢，有轻有重，这样复刻出来的声音才会自然。"
)

const (
	VideoCloneTraining = "training"
	VideoCloneReady    = "ready"
	VideoCloneFailed   = "failed"
	VideoCloneDisabled = "disabled" // by an admin
)

var (
	ErrVideoCloneOff      = infraerrors.BadRequest("VIDEO_CLONE_OFF", "声音复刻暂未开通，请先用预置音色（Voice cloning is not available）")
	ErrVideoCloneFull     = infraerrors.Conflict("VIDEO_CLONE_FULL", "音色位已经用完了，请稍后再试或联系管理员（No free voice slot）")
	ErrVideoCloneLimit    = infraerrors.Forbidden("VIDEO_CLONE_LIMIT", "你的复刻音色已达上限，删掉一个不用的再新建（Too many voices）")
	ErrVideoCloneConsent  = infraerrors.BadRequest("VIDEO_CLONE_CONSENT", "请先勾选声音授权声明（Consent required）")
	ErrVideoCloneAudio    = infraerrors.BadRequest("VIDEO_CLONE_AUDIO", "录音格式不支持：请用页面上的录音，或上传 10 MB 以内的 wav / mp3 / m4a 文件（Unsupported recording）")
	ErrVideoCloneLength   = infraerrors.BadRequest("VIDEO_CLONE_LENGTH", "录音太短或太长：请录 10 到 30 秒的清晰人声（Recording length out of range）")
	ErrVideoCloneName     = infraerrors.BadRequest("VIDEO_CLONE_NAME", "请给音色起个名字（20 字以内）（Name required）")
	ErrVideoCloneNotFound = infraerrors.NotFound("VIDEO_CLONE_NOT_FOUND", "找不到这个音色，可能已被删除，请换一个音色（Voice not found）")
	ErrVideoCloneNotReady = infraerrors.BadRequest("VIDEO_CLONE_NOT_READY", "这个音色还没复刻好（或复刻失败），请到「我的音色」查看，或换一个音色（Voice not ready）")
	ErrVideoCloneRetrain  = infraerrors.Forbidden("VIDEO_CLONE_RETRAIN", "这个音色重录次数已用完，请删掉后新建一个（Too many re-recordings）")
	ErrVideoCloneBusy     = infraerrors.Conflict("VIDEO_CLONE_BUSY", "音色正在复刻中，请等它完成后再操作（Training in progress）")
	ErrVideoCloneTrain    = infraerrors.ServiceUnavailable("VIDEO_CLONE_TRAIN", "声音复刻服务暂时不可用，请稍后再试，这次不扣费（Voice training failed）")
	ErrVideoClonePreview  = infraerrors.TooManyRequests("VIDEO_CLONE_PREVIEW", "今天的试听次数用完了，明天再来（Preview limit reached）")
)

// VideoVoiceClone is one user's cloned voice.
type VideoVoiceClone struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	SpeakerID  string     `json:"-"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	Error      string     `json:"error,omitempty"`
	Trainings  int        `json:"trainings"`
	Price      float64    `json:"price"`
	Charged    bool       `json:"charged"`
	ConsentIP  string     `json:"-"`
	ConsentAt  time.Time  `json:"consent_at"`
	CheckedAt  *time.Time `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Voice      string     `json:"voice"` // clone:<id>, the value a project's options.voice takes
	RetrainsOK int        `json:"retrains_left"`
}

// VideoAdminClone adds what an admin needs to look into a clone.
type VideoAdminClone struct {
	VideoVoiceClone
	SpeakerID string `json:"speaker_id"`
	ConsentIP string `json:"consent_ip"`
	UserEmail string `json:"user_email"`
}

// VideoCloneRepository stores clones (videoRepository).
type VideoCloneRepository interface {
	// ClaimClone inserts c on the first of slots no live clone holds; ErrVideoCloneFull when none is free.
	ClaimClone(ctx context.Context, c *VideoVoiceClone, slots []string, consentText string) error
	ListClones(ctx context.Context, userID int64) ([]VideoVoiceClone, error)
	GetClone(ctx context.Context, id int64) (*VideoVoiceClone, error)
	UpdateClone(ctx context.Context, c *VideoVoiceClone) error
	DeleteClone(ctx context.Context, userID, id int64) (bool, error)
	// MarkCloneReady sets the clone ready and, if it was not charged yet, takes its price from the
	// balance (recorded in video_tts_charges) in the same transaction. It reports what was charged.
	MarkCloneReady(ctx context.Context, id int64) (float64, error)
	AdminListClones(ctx context.Context, limit int) ([]VideoAdminClone, error)
	SetCloneStatus(ctx context.Context, id int64, status, message string) error
}

// VideoVoiceTrainer trains and checks a voice slot (豆包声音复刻).
type VideoVoiceTrainer interface {
	// Train (re)trains the slot with a recording; ready reports a voice usable at once.
	Train(ctx context.Context, speakerID string, audio []byte, format string) (ready bool, err error)
	// Status reports ready (usable), failed (with 豆包's reason) or neither (still training).
	Status(ctx context.Context, speakerID string) (ready bool, failed string, err error)
}

// SetVoiceClones enables 我的音色. The paid narrators (SetPaidTTS) must be on as well.
func (s *VideoService) SetVoiceClones(repo VideoCloneRepository, trainer VideoVoiceTrainer) {
	if repo == nil || trainer == nil {
		return
	}
	s.clones, s.trainer = repo, trainer
}

// VideoCloneInfo is shown on the create page and 我的音色.
type VideoCloneInfo struct {
	Enabled    bool    `json:"enabled"`
	Price      float64 `json:"price"` // slot fee, balance dollars, taken once training succeeds
	MaxPerUser int     `json:"max_per_user"`
	Script     string  `json:"script"`
	Consent    string  `json:"consent"`
	Retrains   int     `json:"retrains"`
}

func (s *VideoService) CloneInfo(ctx context.Context) VideoCloneInfo {
	st := s.Settings(ctx)
	return VideoCloneInfo{Enabled: s.clonesEnabled(st), Price: st.clonePrice(), MaxPerUser: st.cloneMax(), Script: VideoCloneScript, Consent: VideoCloneConsentText, Retrains: videoCloneMaxTrainings}
}

func (s *VideoService) clonesEnabled(st VideoSettings) bool {
	return s.clones != nil && s.trainer != nil && s.paidTTS != nil && s.ttsBilling != nil && len(st.CloneSlots) > 0
}

func videoCloneVoice(id int64) string { return VideoClonePrefix + strconv.FormatInt(id, 10) }

// videoCloneID parses clone:<id>.
func videoCloneID(voice string) (int64, bool) {
	rest, ok := strings.CutPrefix(voice, VideoClonePrefix)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	return id, err == nil && id > 0
}

func decorateClone(c *VideoVoiceClone) {
	c.Voice = videoCloneVoice(c.ID)
	c.RetrainsOK = max(0, videoCloneMaxTrainings-c.Trainings)
}

// cloneSpeaker resolves a project's clone voice to its speaker ID for userID (the owner only).
func (s *VideoService) cloneSpeaker(ctx context.Context, userID int64, voice string) (string, error) {
	id, ok := videoCloneID(voice)
	if !ok || s.clones == nil {
		return "", ErrVideoCloneNotFound
	}
	c, err := s.clones.GetClone(ctx, id)
	if err != nil || c == nil || c.UserID != userID {
		return "", ErrVideoCloneNotFound
	}
	if c.Status != VideoCloneReady {
		return "", ErrVideoCloneNotReady
	}
	return c.SpeakerID, nil
}

// checkCloneVoice is the create / change-voice check: a clone voice must be the user's own and ready.
func (s *VideoService) checkCloneVoice(ctx context.Context, userID int64, voice string) error {
	if _, ok := videoCloneID(voice); !ok {
		return nil
	}
	if s.paidTTS == nil {
		return ErrVideoPaidVoiceOff
	}
	_, err := s.cloneSpeaker(ctx, userID, voice)
	return err
}

// ListClones returns the user's clones, asking 豆包 about those still training.
func (s *VideoService) ListClones(ctx context.Context, userID int64) ([]VideoVoiceClone, error) {
	if s.clones == nil {
		return []VideoVoiceClone{}, nil
	}
	list, err := s.clones.ListClones(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].Status == VideoCloneTraining {
			s.refreshClone(ctx, &list[i])
		}
		decorateClone(&list[i])
	}
	return list, nil
}

// refreshClone asks 豆包 how a training clone is doing (at most every videoClonePollEvery).
func (s *VideoService) refreshClone(ctx context.Context, c *VideoVoiceClone) {
	if c.CheckedAt != nil && s.now().Sub(*c.CheckedAt) < videoClonePollEvery {
		return
	}
	ready, failed, err := s.trainer.Status(ctx, c.SpeakerID)
	now := s.now()
	c.CheckedAt = &now
	switch {
	case err != nil:
		slog.Warn("video: voice clone status", "clone", c.ID, "err", err)
		_ = s.clones.UpdateClone(ctx, c)
	case ready:
		s.cloneReady(ctx, c)
	case failed != "":
		c.Status, c.Error = VideoCloneFailed, clipRunes("复刻失败："+failed+"。请换个安静的地方重新录一段，这次不扣费", 200)
		_ = s.clones.UpdateClone(ctx, c)
	default:
		_ = s.clones.UpdateClone(ctx, c)
	}
}

// cloneReady marks a trained clone ready and takes the slot fee the first time.
func (s *VideoService) cloneReady(ctx context.Context, c *VideoVoiceClone) {
	charged, err := s.clones.MarkCloneReady(context.WithoutCancel(ctx), c.ID)
	if err != nil {
		slog.Warn("video: voice clone ready", "clone", c.ID, "err", err)
		return
	}
	c.Status, c.Error = VideoCloneReady, ""
	if charged > 0 {
		c.Charged = true
		s.afterTTSCharge(c.UserID)
	}
}

// VideoCloneInput is a new recording.
type VideoCloneInput struct {
	Name    string
	Audio   []byte
	Consent bool
	IP      string
}

// CreateClone takes a free slot and starts training it with the user's recording.
func (s *VideoService) CreateClone(ctx context.Context, key *APIKey, in VideoCloneInput) (*VideoVoiceClone, error) {
	if key == nil {
		return nil, ErrVideoKey
	}
	st := s.Settings(ctx)
	if !s.clonesEnabled(st) {
		return nil, ErrVideoCloneOff
	}
	if !in.Consent {
		return nil, ErrVideoCloneConsent
	}
	name := clipRunes(in.Name, 20)
	if name == "" {
		return nil, ErrVideoCloneName
	}
	format, err := checkCloneAudio(in.Audio)
	if err != nil {
		return nil, err
	}
	list, err := s.clones.ListClones(ctx, key.UserID)
	if err != nil {
		return nil, err
	}
	if len(list) >= st.cloneMax() {
		return nil, ErrVideoCloneLimit
	}
	price := st.clonePrice()
	if balance, err := s.ttsBilling.Balance(ctx, key.UserID); err != nil {
		return nil, err
	} else if balance < price {
		return nil, infraerrors.Forbidden("VIDEO_CLONE_BALANCE", fmt.Sprintf("余额不足：复刻一个音色 $%.2f（复刻成功后才扣），当前余额 $%.2f。请到 hivegpt.cn 充值（Insufficient balance）", price, balance))
	}
	c := &VideoVoiceClone{UserID: key.UserID, Name: name, Status: VideoCloneTraining, Trainings: 1, Price: price, ConsentIP: clipRunes(in.IP, 64)}
	if err := s.clones.ClaimClone(ctx, c, st.CloneSlots, VideoCloneConsentText); err != nil {
		return nil, err
	}
	ready, err := s.trainer.Train(ctx, c.SpeakerID, in.Audio, format)
	if err != nil {
		slog.Warn("video: voice clone training", "clone", c.ID, "user", key.UserID, "err", err)
		// Give the slot back: nothing was trained for this user.
		_, _ = s.clones.DeleteClone(context.WithoutCancel(ctx), key.UserID, c.ID)
		return nil, cloneTrainError(err)
	}
	if ready {
		s.cloneReady(ctx, c)
	}
	decorateClone(c)
	return c, nil
}

// RetrainClone replaces a clone's recording (a failed training, or a voice the user does not like).
func (s *VideoService) RetrainClone(ctx context.Context, key *APIKey, id int64, audio []byte) (*VideoVoiceClone, error) {
	if key == nil {
		return nil, ErrVideoKey
	}
	if !s.clonesEnabled(s.Settings(ctx)) {
		return nil, ErrVideoCloneOff
	}
	c, err := s.ownClone(ctx, key.UserID, id)
	if err != nil {
		return nil, err
	}
	switch {
	case c.Status == VideoCloneDisabled:
		return nil, ErrVideoCloneNotFound
	case c.Status == VideoCloneTraining:
		return nil, ErrVideoCloneBusy
	case c.Trainings >= videoCloneMaxTrainings:
		return nil, ErrVideoCloneRetrain
	}
	format, err := checkCloneAudio(audio)
	if err != nil {
		return nil, err
	}
	ready, err := s.trainer.Train(ctx, c.SpeakerID, audio, format)
	if err != nil {
		slog.Warn("video: voice clone retraining", "clone", c.ID, "user", key.UserID, "err", err)
		return nil, cloneTrainError(err)
	}
	c.Status, c.Error, c.CheckedAt = VideoCloneTraining, "", nil
	c.Trainings++
	if err := s.clones.UpdateClone(ctx, c); err != nil {
		return nil, err
	}
	if ready {
		s.cloneReady(ctx, c)
	}
	decorateClone(c)
	return c, nil
}

func (s *VideoService) RenameClone(ctx context.Context, userID, id int64, name string) (*VideoVoiceClone, error) {
	c, err := s.ownClone(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if c.Name = clipRunes(name, 20); c.Name == "" {
		return nil, ErrVideoCloneName
	}
	if err := s.clones.UpdateClone(ctx, c); err != nil {
		return nil, err
	}
	decorateClone(c)
	return c, nil
}

// DeleteClone frees the slot. Films already narrated keep their audio; new narration with it stops.
func (s *VideoService) DeleteClone(ctx context.Context, userID, id int64) error {
	if s.clones == nil {
		return ErrVideoCloneNotFound
	}
	ok, err := s.clones.DeleteClone(ctx, userID, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrVideoCloneNotFound
	}
	return nil
}

// PreviewClone speaks a short sentence with a ready clone (free, a few per day), so the user can hear it.
func (s *VideoService) PreviewClone(ctx context.Context, userID, id int64, text string) (*VideoSpeech, error) {
	c, err := s.ownClone(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if c.Status != VideoCloneReady {
		return nil, ErrVideoCloneNotReady
	}
	if s.paidTTS == nil {
		return nil, ErrVideoPaidVoiceOff
	}
	if !s.allowClonePreview(userID) {
		return nil, ErrVideoClonePreview
	}
	text = clipRunes(text, 60)
	if text == "" {
		text = "你好，这是我用自己的声音复刻的 AI 音色，听起来像我吗？"
	}
	return s.paidTTS.Synthesize(ctx, text, c.SpeakerID, 0, "")
}

func (s *VideoService) allowClonePreview(userID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.previews == nil {
		s.previews = map[int64][]time.Time{}
	}
	cutoff := s.now().Add(-24 * time.Hour)
	kept := s.previews[userID][:0]
	for _, t := range s.previews[userID] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= videoClonePreviewDaily {
		s.previews[userID] = kept
		return false
	}
	s.previews[userID] = append(kept, s.now())
	return true
}

func (s *VideoService) ownClone(ctx context.Context, userID, id int64) (*VideoVoiceClone, error) {
	if s.clones == nil {
		return nil, ErrVideoCloneNotFound
	}
	c, err := s.clones.GetClone(ctx, id)
	if err != nil || c == nil || c.UserID != userID {
		return nil, ErrVideoCloneNotFound
	}
	return c, nil
}

// AdminClones lists recent clones for an admin (who made them, the slot, the consent record).
func (s *VideoService) AdminClones(ctx context.Context, admin *User) ([]VideoAdminClone, error) {
	if admin == nil || admin.Role != RoleAdmin {
		return nil, ErrVideoForbidden
	}
	if s.clones == nil {
		return []VideoAdminClone{}, nil
	}
	list, err := s.clones.AdminListClones(ctx, 200)
	if err != nil {
		return nil, err
	}
	for i := range list {
		decorateClone(&list[i].VideoVoiceClone)
	}
	return list, nil
}

// AdminDisableClone stops a clone from being used (a complaint, an impersonation); enable restores it.
func (s *VideoService) AdminDisableClone(ctx context.Context, admin *User, id int64, disable bool) error {
	if admin == nil || admin.Role != RoleAdmin {
		return ErrVideoForbidden
	}
	if s.clones == nil {
		return ErrVideoCloneNotFound
	}
	status, message := VideoCloneReady, ""
	if disable {
		status, message = VideoCloneDisabled, "这个音色已被管理员停用，如有疑问请联系客服"
	}
	return s.clones.SetCloneStatus(ctx, id, status, message)
}

// checkCloneAudio accepts WAV (what the page records), MP3 and M4A, within the length limits.
func checkCloneAudio(b []byte) (string, error) {
	if len(b) < 1000 || len(b) > VideoCloneMaxBytes {
		return "", ErrVideoCloneAudio
	}
	var format string
	var seconds float64
	switch {
	case len(b) > 44 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WAVE":
		format, seconds = "wav", wavDuration(b)
	case string(b[:3]) == "ID3" || (b[0] == 0xff && b[1]&0xe0 == 0xe0):
		format, seconds = "mp3", mp3Duration(b)
	case len(b) > 12 && string(b[4:8]) == "ftyp":
		format, seconds = "m4a", -1 // length checked by 豆包
	default:
		return "", ErrVideoCloneAudio
	}
	if seconds >= 0 && (seconds < videoCloneMinSeconds || seconds > videoCloneMaxSeconds) {
		return "", ErrVideoCloneLength
	}
	return format, nil
}

// wavDuration reads the fmt and data chunks of a RIFF WAVE file.
func wavDuration(b []byte) float64 {
	byteRate, dataLen := 0, 0
	for i := 12; i+8 <= len(b); {
		id, size := string(b[i:i+4]), int(uint32(b[i+4])|uint32(b[i+5])<<8|uint32(b[i+6])<<16|uint32(b[i+7])<<24)
		body := i + 8
		switch id {
		case "fmt ":
			if body+12 <= len(b) {
				byteRate = int(uint32(b[body+8]) | uint32(b[body+9])<<8 | uint32(b[body+10])<<16 | uint32(b[body+11])<<24)
			}
		case "data":
			dataLen = min(size, len(b)-body)
		}
		if size < 0 || body+size > len(b) {
			break
		}
		i = body + size + size%2
	}
	if byteRate <= 0 {
		return 0
	}
	return float64(dataLen) / float64(byteRate)
}

func cloneTrainError(err error) error {
	var bad *videoCloneRejected
	if errors.As(err, &bad) {
		return infraerrors.BadRequest("VIDEO_CLONE_REJECTED", fmt.Sprintf("录音没通过：%s。请在安静的地方、离麦克风近一点重新录，这次不扣费（Recording rejected）", bad.reason))
	}
	return ErrVideoCloneTrain
}

// videoCloneRejected is 豆包 refusing the recording itself (noise, silence, too short), which the
// user can fix by recording again, as opposed to the service being down.
type videoCloneRejected struct{ reason string }

func (e *videoCloneRejected) Error() string { return "recording rejected: " + e.reason }
