package service

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
)

// Text out of the materials teachers upload for AI 助教: Word (.docx), PowerPoint (.pptx), PDF and
// plain text / Markdown. Older .doc / .ppt and scanned PDFs (pictures only) give no text; the
// teacher is told to save as .docx / .pptx or paste the text.

const (
	tutorUploadMax     = 20 << 20 // bytes per file
	tutorZipEntryMax   = 30 << 20 // uncompressed bytes read from one entry of a .docx / .pptx
	tutorExtractMaxLen = 200000   // runes kept from one file
)

var errTutorFileType = fmt.Errorf("只支持 Word（.docx）、PPT（.pptx）、PDF、txt 和 Markdown 文件；旧版 .doc / .ppt 请另存为新格式，或直接粘贴文字")

// extractTutorText returns the text of an uploaded file.
func extractTutorText(name string, data []byte) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			text, err = "", fmt.Errorf("文件读取失败，可能已损坏；可以直接粘贴文字")
		}
	}()
	switch strings.ToLower(path.Ext(name)) {
	case ".txt", ".md", ".markdown", ".csv":
		if !utf8.Valid(data) {
			return "", fmt.Errorf("文本文件需要是 UTF-8 编码；可以直接粘贴文字")
		}
		text = string(data)
	case ".docx":
		text, err = tutorDocxText(data)
	case ".pptx":
		text, err = tutorPptxText(data)
	case ".pdf":
		text, err = tutorPDFText(data)
	default:
		return "", errTutorFileType
	}
	if err != nil {
		return "", err
	}
	text = tutorCleanText(text)
	if text == "" {
		return "", fmt.Errorf("没有读到文字（扫描版 PDF 和图片里的字读不出来），可以直接粘贴文字")
	}
	if r := []rune(text); len(r) > tutorExtractMaxLen {
		text = string(r[:tutorExtractMaxLen])
	}
	return text, nil
}

var tutorBlankLines = regexp.MustCompile(`\n{3,}`)

func tutorCleanText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t ")
	}
	return strings.TrimSpace(tutorBlankLines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}

func tutorZipFile(zr *zip.Reader, name string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer func() { _ = rc.Close() }()
			return io.ReadAll(io.LimitReader(rc, tutorZipEntryMax))
		}
	}
	return nil, nil
}

// xmlText walks Office XML and writes the text of `textTag` elements, a newline after each `paraTag`.
func xmlText(data []byte, textTag, paraTag string) string {
	var b strings.Builder
	dec := xml.NewDecoder(bytes.NewReader(data))
	inText := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case textTag:
				inText = true
			case "tab":
				_ = b.WriteByte('\t')
			case "br":
				_ = b.WriteByte('\n')
			}
		case xml.EndElement:
			switch t.Name.Local {
			case textTag:
				inText = false
			case paraTag:
				_ = b.WriteByte('\n')
			}
		case xml.CharData:
			if inText {
				_, _ = b.Write(t)
			}
		}
	}
	return b.String()
}

func tutorDocxText(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("打不开这个 Word 文件，请确认是 .docx 格式")
	}
	doc, err := tutorZipFile(zr, "word/document.xml")
	if err != nil || doc == nil {
		return "", fmt.Errorf("这个 Word 文件里没有正文")
	}
	return xmlText(doc, "t", "p"), nil
}

var tutorSlideName = regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)

func tutorPptxText(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("打不开这个 PPT 文件，请确认是 .pptx 格式")
	}
	type slide struct {
		n    int
		name string
	}
	var slides []slide
	for _, f := range zr.File {
		if m := tutorSlideName.FindStringSubmatch(f.Name); m != nil {
			n, _ := strconv.Atoi(m[1])
			slides = append(slides, slide{n, f.Name})
		}
	}
	sort.Slice(slides, func(i, j int) bool { return slides[i].n < slides[j].n })
	var b strings.Builder
	for _, s := range slides {
		x, err := tutorZipFile(zr, s.name)
		if err != nil || x == nil {
			continue
		}
		fmt.Fprintf(&b, "【第 %d 页】\n%s\n", s.n, xmlText(x, "t", "p"))
	}
	return b.String(), nil
}

func tutorPDFText(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("打不开这个 PDF 文件（加密或已损坏），可以直接粘贴文字")
	}
	var b strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		rows, err := p.GetTextByRow()
		if err != nil {
			continue
		}
		for _, row := range rows {
			for _, w := range row.Content {
				_, _ = b.WriteString(w.S)
			}
			_ = b.WriteByte('\n')
		}
		_ = b.WriteByte('\n')
		if b.Len() > tutorExtractMaxLen*4 {
			break
		}
	}
	return b.String(), nil
}
