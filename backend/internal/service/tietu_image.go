package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"golang.org/x/image/draw"
)

// Pictures for AI 做贴图 (tietu.go): gpt-image-2 through this site's gateway on the user's key.

// gatewayImageQuality draws one picture at the given quality (gatewayImage always asks for medium).
func gatewayImageQuality(ctx context.Context, client *http.Client, gatewayURL, userAgent, key, prompt, size, quality string) ([]byte, error) {
	body, _ := json.Marshal(map[string]any{"model": editorImageModel, "prompt": prompt, "size": size, "quality": quality, "n": 1})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayURL+"/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return gatewayImageDo(ctx, client, req, key, userAgent)
}

// gatewayImageEdit draws a new picture guided by a reference picture (/v1/images/edits, no mask):
// the prompt says what to keep from the reference (style, characters) and what to draw.
func gatewayImageEdit(ctx context.Context, client *http.Client, gatewayURL, userAgent, key, prompt, size, quality string, ref []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range map[string]string{"model": editorImageModel, "prompt": prompt, "size": size, "quality": quality, "n": "1"} {
		_ = w.WriteField(k, v)
	}
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="image"; filename="reference.jpg"`)
	h.Set("Content-Type", "image/jpeg")
	part, err := w.CreatePart(h)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(ref); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayURL+"/v1/images/edits", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return gatewayImageDo(ctx, client, req, key, userAgent)
}

func gatewayImageDo(ctx context.Context, client *http.Client, req *http.Request, key, userAgent string) ([]byte, error) {
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errLearnRunFailed("连接画图服务超时")
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 40<<20))
	if err != nil {
		return nil, errLearnRunFailed("读取图片中断")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errLearnRunFailed(upstreamErrorText(raw, resp.StatusCode))
	}
	var r struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &r) != nil || len(r.Data) == 0 {
		return nil, errLearnRunFailed("画图服务没有返回图片")
	}
	if r.Data[0].B64JSON == "" && r.Data[0].URL != "" {
		return gatewayImageURL(ctx, client, r.Data[0].URL)
	}
	data, err := base64.StdEncoding.DecodeString(r.Data[0].B64JSON)
	if err != nil || len(data) == 0 {
		return nil, errLearnRunFailed("画图服务返回的图片无法识别")
	}
	return data, nil
}

// tietuJPEG keeps the picture at full size as a high-quality JPEG (公众号 permanent material takes
// up to 10 MB; the editor re-encodes after framing anyway).
func tietuJPEG(raw []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, errLearnRunFailed("画图服务返回的图片无法识别")
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// tietuVisionURL shrinks a picture to width px for the vision check and returns it as a data URL.
func tietuVisionURL(data []byte, width int) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	b := img.Bounds()
	if b.Dx() > width {
		h := b.Dy() * width / b.Dx()
		dst := image.NewRGBA(image.Rect(0, 0, width, h))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
		img = dst
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 82}); err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
