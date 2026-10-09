package service

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"strings"
)

// Uploaded works are checked by their content, not their name: MP4 and WebM headers are parsed here
// for the duration, the picture size and the codec (the image has no ffprobe), SVG is sanitized in
// video_upload_svg.go and posters are matched by their magic bytes.

// videoMediaInfo is what the checks learned about an uploaded video.
type videoMediaInfo struct {
	Mime     string
	Ext      string
	Duration float64 // seconds
	Width    int
	Height   int
	Codec    string
}

var (
	errVideoMediaUnknown   = errors.New("not an mp4 or webm file")
	errVideoMediaMalformed = errors.New("malformed media file")
	errVideoMediaNoTrack   = errors.New("no video track")
	errVideoMediaDuration  = errors.New("duration unknown")
	errVideoMediaCodec     = errors.New("unsupported codec")
	errVideoMediaQuickTime = errors.New("quicktime file")
)

// probeVideoMedia identifies an MP4 or WebM file and reads its duration, size and codec.
func probeVideoMedia(r io.ReaderAt, size int64) (*videoMediaInfo, error) {
	head := make([]byte, 12)
	if size < 12 {
		return nil, errVideoMediaUnknown
	}
	if _, err := r.ReadAt(head, 0); err != nil {
		return nil, errVideoMediaUnknown
	}
	switch {
	case string(head[4:8]) == "ftyp":
		return probeMP4(r, size)
	case bytes.Equal(head[:4], []byte{0x1A, 0x45, 0xDF, 0xA3}):
		return probeWebM(r, size)
	}
	return nil, errVideoMediaUnknown
}

// --- MP4 (ISO BMFF) ----------------------------------------------------------------------------

type mp4Box struct {
	typ        string
	start, end int64 // the whole box
	data       int64 // after the header
}

const mp4MaxBoxes = 20000

// mp4Children calls fn for each box in [start, end).
func mp4Children(r io.ReaderAt, start, end int64, fn func(b mp4Box) error) error {
	var h [16]byte
	for off, n := start, 0; off+8 <= end; n++ {
		if n > mp4MaxBoxes {
			return errVideoMediaMalformed
		}
		if _, err := r.ReadAt(h[:8], off); err != nil {
			return errVideoMediaMalformed
		}
		size, hdr := int64(binary.BigEndian.Uint32(h[:4])), int64(8)
		switch size {
		case 1:
			if _, err := r.ReadAt(h[8:16], off+8); err != nil {
				return errVideoMediaMalformed
			}
			big := binary.BigEndian.Uint64(h[8:16])
			if big > math.MaxInt64 {
				return errVideoMediaMalformed
			}
			size, hdr = int64(big), 16
		case 0:
			size = end - off
		}
		if size < hdr || size > end-off {
			return errVideoMediaMalformed
		}
		if err := fn(mp4Box{typ: string(h[4:8]), start: off, end: off + size, data: off + hdr}); err != nil {
			return err
		}
		off += size
	}
	return nil
}

func mp4Read(r io.ReaderAt, b mp4Box, limit int) ([]byte, error) {
	n := min(int64(limit), b.end-b.data)
	buf := make([]byte, n)
	if _, err := r.ReadAt(buf, b.data); err != nil && !errors.Is(err, io.EOF) {
		return nil, errVideoMediaMalformed
	}
	return buf, nil
}

type mp4Track struct {
	id            uint32
	handler       string
	codec         string
	width, height int
	timescale     uint32 // mdhd
}

func probeMP4(r io.ReaderAt, size int64) (*videoMediaInfo, error) {
	var (
		ftyp      []byte
		timescale uint32
		duration  uint64
		fragDur   uint64
		tracks    []mp4Track
		sawMoov   bool
		trex      = map[uint32]uint32{} // track → default sample duration
		fragEnd   = map[uint32]uint64{} // track → end of its last fragment, in its timescale
	)
	err := mp4Children(r, 0, size, func(b mp4Box) error {
		switch b.typ {
		case "ftyp":
			var err error
			ftyp, err = mp4Read(r, b, 256)
			return err
		case "moov":
			sawMoov = true
			return mp4Children(r, b.data, b.end, func(c mp4Box) error {
				switch c.typ {
				case "mvhd":
					data, err := mp4Read(r, c, 32)
					if err != nil || len(data) < 20 {
						return errVideoMediaMalformed
					}
					if data[0] == 1 {
						if len(data) < 32 {
							return errVideoMediaMalformed
						}
						timescale, duration = binary.BigEndian.Uint32(data[20:24]), binary.BigEndian.Uint64(data[24:32])
					} else {
						timescale, duration = binary.BigEndian.Uint32(data[12:16]), uint64(binary.BigEndian.Uint32(data[16:20]))
					}
				case "mvex":
					return mp4Children(r, c.data, c.end, func(m mp4Box) error {
						if m.typ == "trex" {
							data, err := mp4Read(r, m, 24)
							if err != nil || len(data) < 16 {
								return errVideoMediaMalformed
							}
							trex[binary.BigEndian.Uint32(data[4:8])] = binary.BigEndian.Uint32(data[12:16])
							return nil
						}
						if m.typ != "mehd" {
							return nil
						}
						data, err := mp4Read(r, m, 12)
						if err != nil || len(data) < 8 {
							return errVideoMediaMalformed
						}
						if data[0] == 1 && len(data) >= 12 {
							fragDur = binary.BigEndian.Uint64(data[4:12])
						} else {
							fragDur = uint64(binary.BigEndian.Uint32(data[4:8]))
						}
						return nil
					})
				case "trak":
					t, err := probeMP4Track(r, c)
					if err != nil {
						return err
					}
					tracks = append(tracks, t)
				}
				return nil
			})
		case "moof":
			return mp4Children(r, b.data, b.end, func(c mp4Box) error {
				if c.typ != "traf" {
					return nil
				}
				id, end, err := probeMP4Fragment(r, c, trex)
				if err == nil && end > fragEnd[id] {
					fragEnd[id] = end
				}
				return err
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(ftyp) < 4 {
		return nil, errVideoMediaMalformed
	}
	switch string(ftyp[:4]) {
	case "qt  ":
		return nil, errVideoMediaQuickTime
	case "avif", "avis", "heic", "heix", "mif1", "msf1", "M4A ", "M4B ":
		return nil, errVideoMediaNoTrack
	}
	if !sawMoov {
		return nil, errVideoMediaMalformed
	}
	var video *mp4Track
	for i := range tracks {
		if tracks[i].handler == "vide" && tracks[i].width > 0 && tracks[i].height > 0 {
			video = &tracks[i]
			break
		}
	}
	if video == nil {
		return nil, errVideoMediaNoTrack
	}
	switch video.codec {
	case "avc1", "avc3", "hvc1", "hev1", "av01", "vp09":
	default:
		return nil, errVideoMediaCodec
	}
	if duration == 0 || duration == math.MaxUint32 || duration == math.MaxUint64 {
		duration = fragDur
	}
	seconds := 0.0
	if timescale > 0 && duration > 0 {
		seconds = float64(duration) / float64(timescale)
	} else if end := fragEnd[video.id]; end > 0 && video.timescale > 0 {
		// Fragmented files (MediaRecorder, live encoders) often leave the duration out: it is where
		// the video track's last fragment ends.
		seconds = float64(end) / float64(video.timescale)
	}
	if seconds <= 0 {
		return nil, errVideoMediaDuration
	}
	return &videoMediaInfo{
		Mime: "video/mp4", Ext: ".mp4", Codec: video.codec,
		Duration: seconds, Width: video.width, Height: video.height,
	}, nil
}

func probeMP4Track(r io.ReaderAt, trak mp4Box) (mp4Track, error) {
	var t mp4Track
	err := mp4Children(r, trak.data, trak.end, func(b mp4Box) error {
		switch b.typ {
		case "tkhd":
			data, err := mp4Read(r, b, 96)
			if err != nil || len(data) < 1 {
				return errVideoMediaMalformed
			}
			// version 0: 4 flags + 20 times/ids; version 1: 4 + 32. Then 16 reserved/layer/volume bytes,
			// the 36-byte matrix, and the width and height as 16.16 fixed point.
			at := 4 + 20
			if data[0] == 1 {
				at = 4 + 32
			}
			at += 16
			if len(data) < at+44 {
				return errVideoMediaMalformed
			}
			if data[0] == 1 {
				t.id = binary.BigEndian.Uint32(data[20:24])
			} else {
				t.id = binary.BigEndian.Uint32(data[12:16])
			}
			m := data[at : at+36]
			w := int(binary.BigEndian.Uint32(data[at+36:at+40]) >> 16)
			h := int(binary.BigEndian.Uint32(data[at+40:at+44]) >> 16)
			// A 90° / 270° rotation (phones filming upright) shows the picture the other way round.
			a, bb := int32(binary.BigEndian.Uint32(m[0:4])), int32(binary.BigEndian.Uint32(m[4:8]))
			if a == 0 && (bb == 1<<16 || bb == -(1<<16)) {
				w, h = h, w
			}
			t.width, t.height = w, h
		case "mdia":
			return mp4Children(r, b.data, b.end, func(c mp4Box) error {
				switch c.typ {
				case "mdhd":
					data, err := mp4Read(r, c, 24)
					if err != nil || len(data) < 16 {
						return errVideoMediaMalformed
					}
					if data[0] == 1 {
						if len(data) < 24 {
							return errVideoMediaMalformed
						}
						t.timescale = binary.BigEndian.Uint32(data[20:24])
					} else {
						t.timescale = binary.BigEndian.Uint32(data[12:16])
					}
				case "hdlr":
					data, err := mp4Read(r, c, 12)
					if err != nil || len(data) < 12 {
						return errVideoMediaMalformed
					}
					t.handler = string(data[8:12])
				case "minf":
					return mp4Children(r, c.data, c.end, func(d mp4Box) error {
						if d.typ != "stbl" {
							return nil
						}
						return mp4Children(r, d.data, d.end, func(e mp4Box) error {
							if e.typ != "stsd" {
								return nil
							}
							data, err := mp4Read(r, e, 16)
							if err != nil || len(data) < 16 {
								return errVideoMediaMalformed
							}
							t.codec = string(data[12:16])
							return nil
						})
					})
				}
				return nil
			})
		}
		return nil
	})
	return t, err
}

// probeMP4Fragment returns a track fragment's track and where it ends: its decode time (tfdt) plus
// the durations of its samples (trun, else the tfhd / trex default).
func probeMP4Fragment(r io.ReaderAt, traf mp4Box, trex map[uint32]uint32) (uint32, uint64, error) {
	var (
		id         uint32
		base       uint64
		defaultDur uint32
		hasDefault bool
		total      uint64
	)
	var runs []mp4Box
	err := mp4Children(r, traf.data, traf.end, func(b mp4Box) error {
		switch b.typ {
		case "tfhd":
			data, err := mp4Read(r, b, 40)
			if err != nil || len(data) < 8 {
				return errVideoMediaMalformed
			}
			flags := uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
			id = binary.BigEndian.Uint32(data[4:8])
			at := 8
			if flags&0x1 != 0 {
				at += 8
			}
			if flags&0x2 != 0 {
				at += 4
			}
			if flags&0x8 != 0 {
				if len(data) < at+4 {
					return errVideoMediaMalformed
				}
				defaultDur, hasDefault = binary.BigEndian.Uint32(data[at:at+4]), true
			}
		case "tfdt":
			data, err := mp4Read(r, b, 12)
			if err != nil || len(data) < 8 {
				return errVideoMediaMalformed
			}
			if data[0] == 1 && len(data) >= 12 {
				base = binary.BigEndian.Uint64(data[4:12])
			} else {
				base = uint64(binary.BigEndian.Uint32(data[4:8]))
			}
		case "trun":
			runs = append(runs, b)
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	if !hasDefault {
		defaultDur = trex[id]
	}
	for _, b := range runs {
		var head [12]byte
		if b.end-b.data < 8 {
			return 0, 0, errVideoMediaMalformed
		}
		if _, err := r.ReadAt(head[:8], b.data); err != nil {
			return 0, 0, errVideoMediaMalformed
		}
		flags := uint32(head[1])<<16 | uint32(head[2])<<8 | uint32(head[3])
		count := int64(binary.BigEndian.Uint32(head[4:8]))
		at := b.data + 8
		if flags&0x1 != 0 {
			at += 4
		}
		if flags&0x4 != 0 {
			at += 4
		}
		if flags&0x100 == 0 {
			total += uint64(count) * uint64(defaultDur)
			continue
		}
		var stride int64
		for _, f := range []uint32{0x100, 0x200, 0x400, 0x800} {
			if flags&f != 0 {
				stride += 4
			}
		}
		if count*stride > b.end-at {
			return 0, 0, errVideoMediaMalformed
		}
		buf := make([]byte, count*stride)
		if _, err := r.ReadAt(buf, at); err != nil {
			return 0, 0, errVideoMediaMalformed
		}
		for i := int64(0); i < count; i++ {
			total += uint64(binary.BigEndian.Uint32(buf[i*stride : i*stride+4]))
		}
	}
	return id, base + total, nil
}

// --- WebM (Matroska subset) ---------------------------------------------------------------------

const (
	ebmlIDHeader      = 0x1A45DFA3
	ebmlIDDocType     = 0x4282
	ebmlIDSegment     = 0x18538067
	ebmlIDInfo        = 0x1549A966
	ebmlIDTimecode    = 0x2AD7B1
	ebmlIDDuration    = 0x4489
	ebmlIDTracks      = 0x1654AE6B
	ebmlIDTrackEntry  = 0xAE
	ebmlIDTrackType   = 0x83
	ebmlIDCodecID     = 0x86
	ebmlIDVideo       = 0xE0
	ebmlIDPixelWidth  = 0xB0
	ebmlIDPixelHeight = 0xBA
	ebmlIDCluster     = 0x1F43B675
	ebmlIDClusterTime = 0xE7
	ebmlIDSimpleBlock = 0xA3
	ebmlIDBlockGroup  = 0xA0
	ebmlIDBlock       = 0xA1

	ebmlMaxElements = 2_000_000
)

// Level-1 elements end a Cluster of unknown size (MediaRecorder writes those).
var ebmlLevel1 = map[uint64]bool{
	ebmlIDCluster: true, ebmlIDInfo: true, ebmlIDTracks: true, 0x1C53BB6B: true, 0x1254C367: true,
	0x1043A770: true, 0x1941A469: true, 0x114D9B74: true,
}

type ebmlElem struct {
	id         uint64
	start      int64
	data       int64
	end        int64 // -1 when the size is unknown
	unknownLen bool
}

type ebmlReader struct {
	r     io.ReaderAt
	size  int64
	count int
}

// vint reads a variable-length integer; keepMarker keeps the length bit (element IDs).
func (e *ebmlReader) vint(off int64, keepMarker bool) (val uint64, n int, unknown bool, err error) {
	var b [8]byte
	if off >= e.size {
		return 0, 0, false, errVideoMediaMalformed
	}
	if _, err := e.r.ReadAt(b[:1], off); err != nil {
		return 0, 0, false, errVideoMediaMalformed
	}
	first := b[0]
	if first == 0 {
		return 0, 0, false, errVideoMediaMalformed
	}
	n = 1
	for mask := byte(0x80); first&mask == 0; mask >>= 1 {
		n++
	}
	if n > 1 {
		if _, err := e.r.ReadAt(b[1:n], off+1); err != nil {
			return 0, 0, false, errVideoMediaMalformed
		}
	}
	val = uint64(first)
	if !keepMarker {
		val &= uint64(0xFF >> n)
	}
	allOnes := val == uint64(0xFF>>n)
	for i := 1; i < n; i++ {
		val = val<<8 | uint64(b[i])
		allOnes = allOnes && b[i] == 0xFF
	}
	return val, n, !keepMarker && allOnes, nil
}

func (e *ebmlReader) elem(off, parentEnd int64) (ebmlElem, error) {
	e.count++
	if e.count > ebmlMaxElements {
		return ebmlElem{}, errVideoMediaMalformed
	}
	id, n1, _, err := e.vint(off, true)
	if err != nil {
		return ebmlElem{}, err
	}
	size, n2, unknown, err := e.vint(off+int64(n1), false)
	if err != nil {
		return ebmlElem{}, err
	}
	el := ebmlElem{id: id, start: off, data: off + int64(n1+n2)}
	if unknown {
		el.end, el.unknownLen = parentEnd, true
		return el, nil
	}
	if size > uint64(parentEnd-el.data) {
		return ebmlElem{}, errVideoMediaMalformed
	}
	el.end = el.data + int64(size)
	return el, nil
}

func (e *ebmlReader) bytes(el ebmlElem, limit int) ([]byte, error) {
	n := el.end - el.data
	if n < 0 || n > int64(limit) {
		return nil, errVideoMediaMalformed
	}
	buf := make([]byte, n)
	if _, err := e.r.ReadAt(buf, el.data); err != nil {
		return nil, errVideoMediaMalformed
	}
	return buf, nil
}

func (e *ebmlReader) uint(el ebmlElem) (uint64, error) {
	b, err := e.bytes(el, 8)
	if err != nil {
		return 0, err
	}
	var v uint64
	for _, x := range b {
		v = v<<8 | uint64(x)
	}
	return v, nil
}

func (e *ebmlReader) float(el ebmlElem) (float64, error) {
	b, err := e.bytes(el, 8)
	if err != nil {
		return 0, err
	}
	switch len(b) {
	case 4:
		return float64(math.Float32frombits(binary.BigEndian.Uint32(b))), nil
	case 8:
		return math.Float64frombits(binary.BigEndian.Uint64(b)), nil
	}
	return 0, errVideoMediaMalformed
}

// children calls fn for each child of [start, end); fn returns how far to skip (the child's end,
// or its data start to descend into an unknown-size child).
func (e *ebmlReader) children(start, end int64, fn func(el ebmlElem) (int64, error)) error {
	for off := start; off < end; {
		el, err := e.elem(off, end)
		if err != nil {
			return err
		}
		next, err := fn(el)
		if err != nil {
			return err
		}
		if next <= off {
			return errVideoMediaMalformed
		}
		off = next
	}
	return nil
}

func probeWebM(r io.ReaderAt, size int64) (*videoMediaInfo, error) {
	e := &ebmlReader{r: r, size: size}
	head, err := e.elem(0, size)
	if err != nil || head.id != ebmlIDHeader || head.unknownLen {
		return nil, errVideoMediaMalformed
	}
	docType := ""
	if err := e.children(head.data, head.end, func(el ebmlElem) (int64, error) {
		if el.id == ebmlIDDocType {
			b, err := e.bytes(el, 32)
			if err != nil {
				return 0, err
			}
			docType = strings.TrimRight(string(b), "\x00")
		}
		return el.end, nil
	}); err != nil {
		return nil, err
	}
	if docType != "webm" {
		return nil, errVideoMediaUnknown
	}
	seg, err := e.elem(head.end, size)
	if err != nil || seg.id != ebmlIDSegment {
		return nil, errVideoMediaMalformed
	}
	var (
		scale         uint64 = 1_000_000 // ns per tick
		durTicks      float64
		lastTicks     float64
		codec         string
		width, height int
	)
	err = e.children(seg.data, seg.end, func(el ebmlElem) (int64, error) {
		switch el.id {
		case ebmlIDInfo:
			return el.end, e.children(el.data, el.end, func(c ebmlElem) (int64, error) {
				var err error
				switch c.id {
				case ebmlIDTimecode:
					scale, err = e.uint(c)
				case ebmlIDDuration:
					durTicks, err = e.float(c)
				}
				return c.end, err
			})
		case ebmlIDTracks:
			return el.end, e.children(el.data, el.end, func(t ebmlElem) (int64, error) {
				if t.id != ebmlIDTrackEntry || codec != "" {
					return t.end, nil
				}
				var typ uint64
				var id string
				var w, h uint64
				err := e.children(t.data, t.end, func(c ebmlElem) (int64, error) {
					var err error
					switch c.id {
					case ebmlIDTrackType:
						typ, err = e.uint(c)
					case ebmlIDCodecID:
						var b []byte
						b, err = e.bytes(c, 64)
						id = strings.TrimRight(string(b), "\x00")
					case ebmlIDVideo:
						err = e.children(c.data, c.end, func(v ebmlElem) (int64, error) {
							var err error
							switch v.id {
							case ebmlIDPixelWidth:
								w, err = e.uint(v)
							case ebmlIDPixelHeight:
								h, err = e.uint(v)
							}
							return v.end, err
						})
					}
					return c.end, err
				})
				if err == nil && typ == 1 {
					codec, width, height = id, int(min(w, 1<<15)), int(min(h, 1<<15))
				}
				return t.end, err
			})
		case ebmlIDCluster:
			if durTicks > 0 {
				return el.end, nil // the header already has the duration
			}
			end, ticks, err := e.clusterEnd(el)
			lastTicks = math.Max(lastTicks, ticks)
			return end, err
		}
		if el.unknownLen {
			return 0, errVideoMediaMalformed
		}
		return el.end, nil
	})
	if err != nil {
		return nil, err
	}
	if codec == "" || width <= 0 || height <= 0 {
		return nil, errVideoMediaNoTrack
	}
	switch codec {
	case "V_VP8", "V_VP9", "V_AV1":
	default:
		return nil, errVideoMediaCodec
	}
	ticks := durTicks
	if ticks <= 0 {
		ticks = lastTicks
	}
	if ticks <= 0 || scale == 0 || math.IsNaN(ticks) || math.IsInf(ticks, 0) {
		return nil, errVideoMediaDuration
	}
	return &videoMediaInfo{
		Mime: "video/webm", Ext: ".webm", Codec: codec,
		Duration: ticks * float64(scale) / 1e9, Width: width, Height: height,
	}, nil
}

// clusterEnd walks a cluster's blocks (files without a duration in the header, e.g. from
// MediaRecorder) and returns where it ends and its last block time in ticks.
func (e *ebmlReader) clusterEnd(cl ebmlElem) (int64, float64, error) {
	var base, last float64
	off := cl.data
	for off < cl.end {
		c, err := e.elem(off, cl.end)
		if err != nil {
			return 0, 0, err
		}
		if cl.unknownLen && ebmlLevel1[c.id] {
			return off, last, nil
		}
		if c.unknownLen {
			return 0, 0, errVideoMediaMalformed
		}
		switch c.id {
		case ebmlIDClusterTime:
			v, err := e.uint(c)
			if err != nil {
				return 0, 0, err
			}
			base = float64(v)
			last = math.Max(last, base)
		case ebmlIDSimpleBlock:
			last = math.Max(last, base+e.blockTime(c))
		case ebmlIDBlockGroup:
			_ = e.children(c.data, c.end, func(b ebmlElem) (int64, error) {
				if b.id == ebmlIDBlock {
					last = math.Max(last, base+e.blockTime(b))
				}
				return b.end, nil
			})
		}
		off = c.end
	}
	return cl.end, last, nil
}

// blockTime is a block's signed 16-bit time relative to its cluster.
func (e *ebmlReader) blockTime(b ebmlElem) float64 {
	_, n, _, err := e.vint(b.data, false) // track number
	if err != nil {
		return 0
	}
	var t [2]byte
	if _, err := e.r.ReadAt(t[:], b.data+int64(n)); err != nil {
		return 0
	}
	return float64(int16(binary.BigEndian.Uint16(t[:])))
}

// --- posters -------------------------------------------------------------------------------------

// sniffVideoPoster accepts JPEG, WebP and PNG cover images.
func sniffVideoPoster(head []byte) (mime, ext string, ok bool) {
	switch {
	case len(head) >= 3 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF:
		return "image/jpeg", ".jpg", true
	case len(head) >= 12 && string(head[:4]) == "RIFF" && string(head[8:12]) == "WEBP":
		return "image/webp", ".webp", true
	case len(head) >= 8 && bytes.Equal(head[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "image/png", ".png", true
	}
	return "", "", false
}
