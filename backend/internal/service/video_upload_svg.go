package service

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Uploaded SVG is untrusted: it is parsed and written out again keeping only SVG elements (shapes,
// text, gradients, filters, SMIL animation, <style>) and attributes that cannot run code or load
// anything from elsewhere. It is still only ever shown through <img> and served with a CSP that
// blocks scripts and network access, so a gap here is not enough on its own to run code.

const (
	svgNS   = "http://www.w3.org/2000/svg"
	xlinkNS = "http://www.w3.org/1999/xlink"
	xmlNS   = "http://www.w3.org/XML/1998/namespace"

	svgMaxElements = 100_000
	svgMaxDepth    = 256
)

var (
	errSVGNotSVG = errors.New("not an svg document")
	errSVGParse  = errors.New("svg parse error")
	errSVGLarge  = errors.New("svg too complex")
)

var svgAllowedElements = map[string]bool{}

func init() {
	for _, name := range strings.Fields(`svg g defs symbol use path rect circle ellipse line polyline polygon
		text tspan textPath title desc linearGradient radialGradient stop pattern clipPath mask filter marker style
		image switch view a animate animateMotion animateTransform set mpath
		feBlend feColorMatrix feComponentTransfer feComposite feConvolveMatrix feDiffuseLighting feDisplacementMap
		feDistantLight feDropShadow feFlood feFuncA feFuncB feFuncG feFuncR feGaussianBlur feImage feMerge
		feMergeNode feMorphology feOffset fePointLight feSpecularLighting feSpotLight feTile feTurbulence`) {
		svgAllowedElements[name] = true
	}
}

var (
	cssImportRe  = regexp.MustCompile(`(?i)@import[^;]*;?`)
	cssURLRe     = regexp.MustCompile(`(?i)url\(\s*([^)]*?)\s*\)`)
	cssEscapeRe  = regexp.MustCompile(`\\(?:[0-9a-fA-F]{1,6}[ \t\r\n\f]?|[^0-9a-fA-F\r\n\f])`)
	cssImageSet  = regexp.MustCompile(`(?i)(-webkit-)?image-set\(`)
	dataImageRe  = regexp.MustCompile(`(?i)^data:image/(png|jpeg|jpg|gif|webp);base64,[a-z0-9+/=\s]*$`)
	svgNumberRe  = regexp.MustCompile(`^\s*([0-9]*\.?[0-9]+(?:e[-+]?[0-9]+)?)\s*(px)?\s*$`)
	unsafeTextRe = regexp.MustCompile(`(?i)(javascript|vbscript|livescript):|data:text/|data:application/`)
)

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// svgSafeRef allows same-document references and inline raster images.
func svgSafeRef(v string) bool {
	v = strings.TrimSpace(v)
	return strings.HasPrefix(v, "#") || dataImageRe.MatchString(v)
}

// svgUnsafeText reports values that try to run script, after removing the whitespace and control
// characters browsers ignore inside URLs ("java\tscript:").
func svgUnsafeText(v string) bool {
	compact := strings.Map(func(r rune) rune {
		if r <= ' ' || r == 0x7F {
			return -1
		}
		return r
	}, v)
	return unsafeTextRe.MatchString(compact)
}

// sanitizeSVGCSS keeps CSS (animations, keyframes, fills) but removes imports and every url() that
// is not a same-document reference or an inline raster image. CSS escapes are decoded first, since
// they can spell "url(" in ways a pattern does not catch; the result has no backslashes left, so
// the browser reads exactly what was checked.
func sanitizeSVGCSS(css string) string {
	css = cssEscapeRe.ReplaceAllStringFunc(css, func(m string) string {
		if !isHexDigit(m[1]) {
			return m[1:]
		}
		n, _ := strconv.ParseUint(strings.TrimSpace(m[1:]), 16, 32)
		if n == 0 || n > unicode.MaxRune || (n >= 0xD800 && n <= 0xDFFF) {
			return "\uFFFD"
		}
		return string(rune(n))
	})
	css = strings.ReplaceAll(css, `\`, "")
	css = cssImportRe.ReplaceAllString(css, "")
	css = cssImageSet.ReplaceAllString(css, "none(")
	css = cssURLRe.ReplaceAllStringFunc(css, func(m string) string {
		inner := strings.TrimSpace(cssURLRe.FindStringSubmatch(m)[1])
		inner = strings.Trim(inner, `"'`)
		if svgSafeRef(inner) {
			return "url(" + inner + ")"
		}
		return "none"
	})
	if svgUnsafeText(css) {
		return ""
	}
	return css
}

type svgOut struct {
	buf bytes.Buffer
}

// Text keeps its line breaks readable; attributes use xml.EscapeText.
var svgTextEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\r", "&#xD;")

func (o *svgOut) text(s string) {
	_, _ = svgTextEscaper.WriteString(&o.buf, s)
}

func (o *svgOut) attr(name, value string) {
	_ = o.buf.WriteByte(' ')
	_, _ = o.buf.WriteString(name)
	_, _ = o.buf.WriteString(`="`)
	_ = xml.EscapeText(&o.buf, []byte(value))
	_ = o.buf.WriteByte('"')
}

// svgAttrName maps a parsed attribute to how it is written, or "" to drop it.
func svgAttrName(a xml.Attr) string {
	local := a.Name.Local
	switch a.Name.Space {
	case "":
		if local == "xmlns" || strings.Contains(local, ":") {
			return ""
		}
		return local
	case xlinkNS, "xlink":
		if local == "href" || local == "title" {
			return "xlink:" + local
		}
	case xmlNS, "xml":
		if local == "space" || local == "lang" {
			return "xml:" + local
		}
	}
	return "" // xmlns declarations (written once on the root) and editor namespaces
}

type svgInfo struct {
	Width, Height int
}

// sanitizeSVG returns a cleaned copy of an SVG document and its picture size.
func sanitizeSVG(data []byte) ([]byte, svgInfo, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.Strict = true
	var (
		out      svgOut
		info     svgInfo
		depth    int // open elements written
		skip     int // >0 while inside a dropped element
		elements int
		rooted   bool
		closed   bool
		inStyle  bool
		styleBuf strings.Builder
	)
	_, _ = out.buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, info, errSVGParse
		}
		switch t := tok.(type) {
		case xml.StartElement:
			elements++
			if elements > svgMaxElements || depth+skip > svgMaxDepth {
				return nil, info, errSVGLarge
			}
			if !rooted {
				if t.Name.Local != "svg" || (t.Name.Space != svgNS && t.Name.Space != "") {
					return nil, info, errSVGNotSVG
				}
			} else if closed {
				return nil, info, errSVGParse
			}
			if skip > 0 || (t.Name.Space != svgNS && t.Name.Space != "") || !svgAllowedElements[t.Name.Local] || svgDropAnimation(t) {
				if !rooted {
					return nil, info, errSVGNotSVG
				}
				skip++
				continue
			}
			_ = out.buf.WriteByte('<')
			_, _ = out.buf.WriteString(t.Name.Local)
			if !rooted {
				rooted = true
				out.attr("xmlns", svgNS)
				out.attr("xmlns:xlink", xlinkNS)
				info = svgSize(t)
			}
			seen := map[string]bool{}
			for _, a := range t.Attr {
				name := svgAttrName(a)
				lower := strings.ToLower(name)
				if name == "" || seen[name] || strings.HasPrefix(lower, "on") || svgUnsafeText(a.Value) {
					continue
				}
				value := a.Value
				switch {
				case lower == "href" || lower == "xlink:href":
					if !svgSafeRef(value) {
						continue
					}
				case lower == "style" || strings.Contains(strings.ToLower(value), "url(") || strings.Contains(value, `\`):
					value = sanitizeSVGCSS(value)
				}
				seen[name] = true
				out.attr(name, value)
			}
			_ = out.buf.WriteByte('>')
			depth++
			if t.Name.Local == "style" {
				inStyle = true
				styleBuf.Reset()
			}
		case xml.EndElement:
			if skip > 0 {
				skip--
				continue
			}
			if depth == 0 {
				return nil, info, errSVGParse
			}
			if inStyle {
				out.text(sanitizeSVGCSS(styleBuf.String()))
				inStyle = false
			}
			_, _ = out.buf.WriteString("</")
			_, _ = out.buf.WriteString(t.Name.Local)
			_ = out.buf.WriteByte('>')
			depth--
			if depth == 0 {
				closed = true
			}
		case xml.CharData:
			if skip > 0 || depth == 0 {
				continue
			}
			if inStyle {
				_, _ = styleBuf.Write(t)
				continue
			}
			out.text(string(t))
		}
		// Comments, processing instructions and DOCTYPEs are dropped.
	}
	if !rooted || !closed {
		return nil, info, errSVGNotSVG
	}
	return out.buf.Bytes(), info, nil
}

// svgDropAnimation drops SMIL that would animate a link or an event handler into existence.
func svgDropAnimation(t xml.StartElement) bool {
	switch t.Name.Local {
	case "animate", "set", "animateTransform", "animateMotion":
	default:
		return false
	}
	for _, a := range t.Attr {
		if a.Name.Local != "attributeName" {
			continue
		}
		v := strings.ToLower(strings.TrimSpace(a.Value))
		if v == "href" || strings.HasSuffix(v, ":href") || strings.HasPrefix(v, "on") || v == "style" {
			return true
		}
	}
	return false
}

// svgSize reads the picture size from viewBox, else width / height, scaled to a 1920-pixel long
// side (only the ratio matters); unknown sizes are 16:9. Ratios beyond 1:4 are clamped.
func svgSize(root xml.StartElement) svgInfo {
	var w, h float64
	var width, height string
	for _, a := range root.Attr {
		if a.Name.Space != "" {
			continue
		}
		switch a.Name.Local {
		case "viewBox":
			f := strings.FieldsFunc(a.Value, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' || r == '\n' || r == '\r' })
			if len(f) == 4 {
				w, _ = strconv.ParseFloat(f[2], 64)
				h, _ = strconv.ParseFloat(f[3], 64)
			}
		case "width":
			width = a.Value
		case "height":
			height = a.Value
		}
	}
	if !(w > 0 && h > 0) {
		mw, mh := svgNumberRe.FindStringSubmatch(width), svgNumberRe.FindStringSubmatch(height)
		if mw != nil && mh != nil {
			w, _ = strconv.ParseFloat(mw[1], 64)
			h, _ = strconv.ParseFloat(mh[1], 64)
		}
	}
	if !(w > 0 && h > 0) || math.IsInf(w, 0) || math.IsInf(h, 0) {
		return svgInfo{Width: 1920, Height: 1080}
	}
	ratio := math.Min(4, math.Max(0.25, w/h))
	if ratio >= 1 {
		return svgInfo{Width: 1920, Height: int(math.Round(1920 / ratio))}
	}
	return svgInfo{Width: int(math.Round(1920 * ratio)), Height: 1920}
}
