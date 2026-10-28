package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
)

// The 课本诗画 queue: one loop hands queued drawings to at most Concurrency goroutines (and stops
// for the day at DailyLimit), and runs one super-resolution at a time on the imagetools service.

type shihuaDrawn struct {
	Data          []byte
	Width, Height int
}

type shihuaDrawFunc func(ctx context.Context, key, prompt, size, quality string) (*shihuaDrawn, error)
type shihuaChatFunc func(ctx context.Context, key, model string, messages []LearnMessage) (string, error)

// shihuaUpscaleFunc doubles an image whose long edge is at most 1024 px.
type shihuaUpscaleFunc func(ctx context.Context, image []byte) ([]byte, error)

// shihuaDrawError is a failed image request; Status 0 is a network error or a timeout.
type shihuaDrawError struct {
	Status int
	Msg    string
}

func (e *shihuaDrawError) Error() string { return e.Msg }

func (e *shihuaDrawError) retryable() bool {
	return e.Status == 0 || e.Status == http.StatusTooManyRequests || e.Status >= 500
}

const (
	shihuaDrawTimeout    = 8 * time.Minute
	shihuaUpscaleTimeout = 6 * time.Minute
	shihuaTick           = 5 * time.Second
	shihuaUpscaleEdge    = 1024 // the imagetools service shrinks anything longer before upscaling
	shihuaUpscaleOverlap = 64
)

// NewShihuaService: gatewayURL is this server's own base URL; dir keeps the pictures (on the data
// volume). tools may be nil or unconfigured, then pictures are not upscaled.
func NewShihuaService(repo ShihuaRepository, settings SettingRepository, keys shihuaKeyReader, billing *BillingService, tools *ImageToolsService, gatewayURL, dir string) *ShihuaService {
	gatewayURL = strings.TrimRight(gatewayURL, "/")
	client := &http.Client{Timeout: shihuaDrawTimeout}
	s := &ShihuaService{repo: repo, settings: settings, keys: keys, billing: billing, tools: tools, files: newShihuaFiles(dir), wikisource: newShihuaWikisource(shihuaWikisourceBase), now: time.Now,
		wake: make(chan struct{}, 1), stop: make(chan struct{})}
	s.draw = func(ctx context.Context, key, prompt, size, quality string) (*shihuaDrawn, error) {
		return shihuaGatewayDraw(ctx, client, gatewayURL, key, prompt, size, quality)
	}
	s.chat = func(ctx context.Context, key, model string, messages []LearnMessage) (string, error) {
		var out strings.Builder
		_, err := streamChatCompletion(ctx, &http.Client{}, gatewayURL, "hivegpt-shihua/1", key, model, messages, 6000, func(d string) error {
			_, _ = out.WriteString(d)
			return nil
		})
		return out.String(), err
	}
	if tools.Available() {
		s.upscale = func(ctx context.Context, img []byte) ([]byte, error) { return tools.UpscaleInternal(ctx, img, 2) }
	}
	return s
}

// Start puts drawings a previous process left running back in the queue and starts the loop.
func (s *ShihuaService) Start() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if n, err := s.repo.RequeueRunning(ctx); err != nil {
		slog.Warn("shihua: requeue interrupted drawings", "err", err)
	} else if n > 0 {
		slog.Info("shihua: requeued interrupted drawings", "count", n)
	}
	if _, err := s.repo.RequeueUpscaleRunning(ctx); err != nil {
		slog.Warn("shihua: requeue interrupted upscales", "err", err)
	}
	s.wg.Add(1)
	go s.loop()
}

// Stop ends the loop and waits for running drawings (tests).
func (s *ShihuaService) Stop() {
	close(s.stop)
	s.wg.Wait()
}

func (s *ShihuaService) kick() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *ShihuaService) runningCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *ShihuaService) loop() {
	defer s.wg.Done()
	ticker := time.NewTicker(shihuaTick)
	defer ticker.Stop()
	upscaling := make(chan struct{}, 1)
	for {
		select {
		case <-s.stop:
			return
		case <-s.wake:
		case <-ticker.C:
		}
		s.dispatch()
		if s.upscale != nil {
			select {
			case upscaling <- struct{}{}:
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					defer func() { <-upscaling }()
					s.upscaleNext()
				}()
			default:
			}
		}
	}
}

func (s *ShihuaService) dispatch() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	st := s.Settings(ctx)
	for s.runningCount() < st.Concurrency {
		stats, err := s.repo.QueueStats(ctx, shihuaDayStart(s.now()))
		if err != nil || stats.Queued == 0 || stats.StartedToday >= st.DailyLimit {
			return
		}
		key, err := s.adminKey(ctx, st.KeyID, 0)
		if err != nil {
			return
		}
		img, err := s.repo.ClaimNext(ctx)
		if err != nil {
			slog.Warn("shihua: claim", "err", err)
			return
		}
		if img == nil {
			return
		}
		s.mu.Lock()
		s.running++
		s.mu.Unlock()
		s.wg.Add(1)
		go s.runDraw(img, key.Key)
	}
}

func (s *ShihuaService) runDraw(img *ShihuaImage, key string) {
	defer s.wg.Done()
	defer func() {
		s.mu.Lock()
		s.running--
		s.mu.Unlock()
		s.kick()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), shihuaDrawTimeout)
	defer cancel()
	started := s.now()
	out, err := s.draw(ctx, key, img.Prompt, img.Size, img.Quality)
	img.DurationMs = int(s.now().Sub(started) / time.Millisecond)
	now := s.now()
	if err == nil {
		file, w, h, saveErr := s.files.saveDrawing(img.WorkID, img.ID, out.Data)
		if saveErr != nil {
			err = &shihuaDrawError{Status: -1, Msg: "保存图片失败：" + saveErr.Error()}
		} else {
			img.Status, img.File, img.Width, img.Height, img.Error = ShihuaImageSucceeded, file, w, h, ""
			img.CostUSD = s.imageCost(w, h)
			img.FinishedAt, img.NotBefore = &now, nil
		}
	}
	if err != nil {
		var de *shihuaDrawError
		if !errors.As(err, &de) {
			de = &shihuaDrawError{Msg: err.Error()}
		}
		img.Error = de.Msg
		if de.retryable() && img.Attempts < shihuaMaxAttempts {
			next := now.Add(shihuaBackoff(img.Attempts))
			img.Status, img.NotBefore = ShihuaImageQueued, &next
		} else {
			img.Status, img.FinishedAt = ShihuaImageFailed, &now
		}
	}
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelWrite()
	if err := s.repo.FinishImage(writeCtx, img); err != nil {
		slog.Warn("shihua: finish drawing", "id", img.ID, "err", err)
	}
}

// shihuaBackoff is the wait before the next attempt: 30 s, then 90 s.
func shihuaBackoff(attempts int) time.Duration {
	d := 30 * time.Second
	for i := 1; i < attempts; i++ {
		d *= 3
	}
	return d
}

// imageCost is the drawing's price at the site's standard image price (group prices aside).
func (s *ShihuaService) imageCost(w, h int) float64 {
	if s.billing == nil {
		return 0
	}
	tier, ok := ClassifyImageBillingTier(fmt.Sprintf("%dx%d", w, h))
	if !ok {
		tier = ImageBillingSize2K
	}
	return s.billing.CalculateImageCost(shihuaImageModel, tier, 1, nil, 1).TotalCost
}

// shihuaGatewayDraw draws one picture through this site's gateway on the admin's key.
func shihuaGatewayDraw(ctx context.Context, client *http.Client, gatewayURL, key, prompt, size, quality string) (*shihuaDrawn, error) {
	body, _ := json.Marshal(map[string]any{"model": shihuaImageModel, "prompt": prompt, "size": size, "quality": quality, "n": 1})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayURL+"/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "hivegpt-shihua/1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, &shihuaDrawError{Msg: "连接画图服务失败或超时（" + err.Error() + "）"}
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 60<<20))
	if err != nil {
		return nil, &shihuaDrawError{Msg: "读取图片中断（Read interrupted）"}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &shihuaDrawError{Status: resp.StatusCode, Msg: fmt.Sprintf("HTTP %d：%s", resp.StatusCode, upstreamErrorText(raw, resp.StatusCode))}
	}
	var r struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &r) != nil || len(r.Data) == 0 {
		return nil, &shihuaDrawError{Status: http.StatusBadGateway, Msg: "画图服务没有返回图片（No image returned）"}
	}
	var data []byte
	if r.Data[0].B64JSON != "" {
		data, err = base64.StdEncoding.DecodeString(r.Data[0].B64JSON)
	} else if r.Data[0].URL != "" {
		data, err = gatewayImageURL(ctx, client, r.Data[0].URL)
	}
	if err != nil || len(data) == 0 {
		return nil, &shihuaDrawError{Status: http.StatusBadGateway, Msg: "画图服务返回的图片无法读取（Unreadable image）"}
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, &shihuaDrawError{Status: http.StatusBadGateway, Msg: "画图服务返回的图片无法识别（Unknown image format）"}
	}
	return &shihuaDrawn{Data: data, Width: cfg.Width, Height: cfg.Height}, nil
}

func (s *ShihuaService) upscaleNext() {
	ctx, cancel := context.WithTimeout(context.Background(), shihuaUpscaleTimeout)
	defer cancel()
	img, err := s.repo.ClaimUpscale(ctx)
	if err != nil || img == nil {
		if err != nil {
			slog.Warn("shihua: claim upscale", "err", err)
		}
		return
	}
	finish := func(status, errText, file string) {
		writeCtx, cancelWrite := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelWrite()
		if err := s.repo.SetUpscale(writeCtx, img.ID, status, errText, file); err != nil {
			slog.Warn("shihua: finish upscale", "id", img.ID, "err", err)
		}
	}
	src, err := s.files.read(img.File)
	if err != nil {
		finish(ShihuaUpscaleFailed, "读取原图失败（Original missing）", "")
		return
	}
	out, err := shihuaUpscaleTiled(ctx, src, s.upscale)
	if err != nil {
		finish(ShihuaUpscaleFailed, "超分失败："+err.Error(), "")
		return
	}
	file, err := s.files.saveUpscaled(img.WorkID, img.ID, out)
	if err != nil {
		finish(ShihuaUpscaleFailed, "保存超分图失败（Save failed）", "")
		return
	}
	finish(ShihuaUpscaleDone, "", file)
}

// shihuaUpscaleTiled doubles a picture. The imagetools service shrinks inputs to 1024 px on the long
// edge first, so a 1024×1536 drawing is sent as overlapping bands no longer than that, and the
// doubled bands are blended back together across the overlap.
func shihuaUpscaleTiled(ctx context.Context, data []byte, up shihuaUpscaleFunc) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	W, H := b.Dx(), b.Dy()
	vertical := H >= W
	L := max(W, H)
	if L <= shihuaUpscaleEdge {
		out, err := up(ctx, data)
		if err != nil {
			return nil, err
		}
		img, _, err := image.Decode(bytes.NewReader(out))
		if err != nil {
			return nil, err
		}
		return encodeShihuaJPEG(fitShihua(img, 2*W, 2*H))
	}
	if min(W, H) > shihuaUpscaleEdge {
		return nil, errors.New("图片两边都超过 1024px，不需要超分")
	}
	ov := shihuaUpscaleOverlap
	n := (L - ov + shihuaUpscaleEdge - ov - 1) / (shihuaUpscaleEdge - ov)
	pl := (L + (n-1)*ov + n - 1) / n
	dst := image.NewRGBA(image.Rect(0, 0, 2*W, 2*H))
	prevEnd := 0
	for i := 0; i < n; i++ {
		start := 0
		if n > 1 {
			start = i * (L - pl) / (n - 1)
		}
		var r image.Rectangle
		if vertical {
			r = image.Rect(0, start, W, start+pl)
		} else {
			r = image.Rect(start, 0, start+pl, H)
		}
		band := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		xdraw.Copy(band, image.Point{}, src, r.Add(b.Min), xdraw.Src, nil)
		var buf bytes.Buffer
		if err := png.Encode(&buf, band); err != nil {
			return nil, err
		}
		out, err := up(ctx, buf.Bytes())
		if err != nil {
			return nil, err
		}
		piece, _, err := image.Decode(bytes.NewReader(out))
		if err != nil {
			return nil, err
		}
		big := fitShihua(piece, 2*r.Dx(), 2*r.Dy())
		blend := 0
		if i > 0 {
			blend = 2 * (prevEnd - start)
		}
		pasteShihuaBand(dst, big, 2*start, blend, vertical)
		prevEnd = start + pl
	}
	return encodeShihuaJPEG(dst)
}

// fitShihua resizes img to exactly w×h when the service returned another size.
func fitShihua(img image.Image, w, h int) image.Image {
	if img.Bounds().Dx() == w && img.Bounds().Dy() == h {
		return img
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(out, out.Bounds(), img, img.Bounds(), xdraw.Src, nil)
	return out
}

// pasteShihuaBand draws band into dst at offset along the long axis; its first `blend` pixels fade in
// over what the previous band left there.
func pasteShihuaBand(dst *image.RGBA, band image.Image, offset, blend int, vertical bool) {
	bb := band.Bounds()
	for y := 0; y < bb.Dy(); y++ {
		for x := 0; x < bb.Dx(); x++ {
			along := y
			dx, dy := x, offset+y
			if !vertical {
				along, dx, dy = x, offset+x, y
			}
			if !(image.Point{X: dx, Y: dy}.In(dst.Rect)) {
				continue
			}
			c, _ := color.RGBAModel.Convert(band.At(bb.Min.X+x, bb.Min.Y+y)).(color.RGBA)
			if along < blend {
				t := float64(along+1) / float64(blend+1)
				o := dst.RGBAAt(dx, dy)
				c = color.RGBA{R: mixShihua(o.R, c.R, t), G: mixShihua(o.G, c.G, t), B: mixShihua(o.B, c.B, t), A: 255}
			}
			dst.SetRGBA(dx, dy, c)
		}
	}
}

func mixShihua(a, b uint8, t float64) uint8 {
	return uint8(float64(a)*(1-t) + float64(b)*t + 0.5)
}

func encodeShihuaJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ---- shared with the 素材库 ----

// AdminDraw draws one picture on the key chosen in the 课本诗画 settings (the 素材库 shares it).
// Failures are *shihuaDrawError; a missing key is not retryable.
func (s *ShihuaService) AdminDraw(ctx context.Context, prompt, size, quality string) (*shihuaDrawn, error) {
	st := s.Settings(ctx)
	key, err := s.adminKey(ctx, st.KeyID, 0)
	if err != nil {
		return nil, &shihuaDrawError{Status: -1, Msg: "没有可用的 API Key，请在「课本诗画 → 设置」里选一把（No API key）"}
	}
	return s.draw(ctx, key.Key, prompt, size, quality)
}

// DrawLimits returns the drawing concurrency and daily limit of the settings, and whether a usable
// key is configured.
func (s *ShihuaService) DrawLimits(ctx context.Context) (int, int, bool) {
	st := s.Settings(ctx)
	_, err := s.adminKey(ctx, st.KeyID, 0)
	return st.Concurrency, st.DailyLimit, err == nil
}

// ImageCost is a picture's price at the site's standard image price.
func (s *ShihuaService) ImageCost(w, h int) float64 { return s.imageCost(w, h) }

// DataDir is where the 课本诗画 pictures live; the 素材库 keeps its own next to it.
func (s *ShihuaService) DataDir() string { return s.files.dir }
