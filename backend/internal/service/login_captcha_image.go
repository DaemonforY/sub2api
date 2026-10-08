package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand/v2"
	"sync"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/math/fixed"
)

const (
	loginCaptchaWidth  = 132
	loginCaptchaHeight = 44
	loginCaptchaGlyph  = 44 // square cell each character is drawn into before rotating
)

var (
	loginCaptchaFontOnce sync.Once
	loginCaptchaFont     *opentype.Font
	loginCaptchaFontErr  error
)

func loginCaptchaFace(size float64) (font.Face, error) {
	loginCaptchaFontOnce.Do(func() {
		loginCaptchaFont, loginCaptchaFontErr = opentype.Parse(gobold.TTF)
	})
	if loginCaptchaFontErr != nil {
		return nil, fmt.Errorf("captcha font: %w", loginCaptchaFontErr)
	}
	return opentype.NewFace(loginCaptchaFont, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
}

// renderLoginCaptcha draws code as a distorted PNG and returns it as a data URL.
// Visual noise only — the code itself comes from crypto/rand.
func renderLoginCaptcha(code string) (string, error) {
	img := image.NewRGBA(image.Rect(0, 0, loginCaptchaWidth, loginCaptchaHeight))
	bg := color.RGBA{uint8(230 + rand.IntN(20)), uint8(232 + rand.IntN(20)), uint8(236 + rand.IntN(20)), 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	// Faint background specks.
	for i := 0; i < 160; i++ {
		img.Set(rand.IntN(loginCaptchaWidth), rand.IntN(loginCaptchaHeight), randomCaptchaInk(150))
	}

	step := float64(loginCaptchaWidth-12) / float64(len(code))
	for i, ch := range code {
		face, err := loginCaptchaFace(26 + float64(rand.IntN(7)))
		if err != nil {
			return "", err
		}
		glyph := image.NewRGBA(image.Rect(0, 0, loginCaptchaGlyph, loginCaptchaGlyph))
		d := &font.Drawer{Dst: glyph, Src: &image.Uniform{randomCaptchaInk(110)}, Face: face}
		adv := d.MeasureString(string(ch))
		d.Dot = fixed.Point26_6{
			X: fixed.I(loginCaptchaGlyph/2) - adv/2,
			Y: fixed.I(loginCaptchaGlyph/2) + face.Metrics().Ascent/2 - fixed.I(2),
		}
		d.DrawString(string(ch))
		_ = face.Close()

		// Rotate about the cell centre and place it with some jitter.
		angle := (rand.Float64()*50 - 25) * math.Pi / 180
		sin, cos := math.Sin(angle), math.Cos(angle)
		cx := 6 + step*float64(i) + step/2 + float64(rand.IntN(5)-2)
		cy := float64(loginCaptchaHeight)/2 + float64(rand.IntN(7)-3)
		half := float64(loginCaptchaGlyph) / 2
		m := f64.Aff3{
			cos, -sin, cx - cos*half + sin*half,
			sin, cos, cy - sin*half - cos*half,
		}
		draw.BiLinear.Transform(img, m, glyph, glyph.Bounds(), draw.Over, nil)
	}

	// Wavy strike-through lines across the text.
	for i := 0; i < 3; i++ {
		drawCaptchaWave(img, randomCaptchaInk(130))
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("captcha png: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func randomCaptchaInk(maxChannel int) color.RGBA {
	return color.RGBA{uint8(rand.IntN(maxChannel)), uint8(rand.IntN(maxChannel)), uint8(rand.IntN(maxChannel)), 255}
}

func drawCaptchaWave(img *image.RGBA, c color.RGBA) {
	amp := 3 + rand.Float64()*6
	period := 30 + rand.Float64()*50
	phase := rand.Float64() * 2 * math.Pi
	base := 10 + rand.Float64()*float64(loginCaptchaHeight-20)
	for x := 0; x < loginCaptchaWidth; x++ {
		y := int(base + amp*math.Sin(float64(x)/period*2*math.Pi+phase))
		img.SetRGBA(x, y, c)
		img.SetRGBA(x, y+1, c)
	}
}
