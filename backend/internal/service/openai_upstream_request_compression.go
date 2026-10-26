package service

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// openAIUpstreamZstdMinBytes 小于该大小的请求体不压缩：省下的字节抵不过编码开销。
const openAIUpstreamZstdMinBytes = 32 << 10

// openAIUpstreamZstdEncoder 只用 EncodeAll，可被多个 goroutine 共享。
// 用 SpeedBetterCompression：SpeedDefault 几乎压不动 base64 截图（实测 87% vs 68%），
// 而 5.5MB 请求体也只多花约 4ms。
var openAIUpstreamZstdEncoder, _ = zstd.NewWriter(nil,
	zstd.WithEncoderLevel(zstd.SpeedBetterCompression),
	zstd.WithEncoderConcurrency(1),
)

// compressOpenAIUpstreamRequestBody 把发往上游的请求体换成 zstd 压缩版本，并设置
// Content-Encoding: zstd。用于上游入站带宽很小、请求体又大（Codex 历史里带截图，
// 一次几 MB）的场景。读不到 body、已有 Content-Encoding、太小或压缩后不更小时原样发送。
func compressOpenAIUpstreamRequestBody(req *http.Request) {
	if req == nil || req.Body == nil || req.Body == http.NoBody {
		return
	}
	if enc := strings.TrimSpace(req.Header.Get("Content-Encoding")); enc != "" && !strings.EqualFold(enc, "identity") {
		return
	}
	if req.ContentLength >= 0 && req.ContentLength < openAIUpstreamZstdMinBytes {
		return
	}

	var raw []byte
	if req.GetBody != nil {
		rc, err := req.GetBody()
		if err != nil {
			return
		}
		raw, err = io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return
		}
	} else {
		var err error
		raw, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			// Body 已被读走，只能放回读到的部分；上游会按截断的 body 报错，与原先读失败一致。
			setOpenAIUpstreamRequestBody(req, raw)
			return
		}
		setOpenAIUpstreamRequestBody(req, raw)
	}
	if len(raw) < openAIUpstreamZstdMinBytes {
		return
	}

	compressed := openAIUpstreamZstdEncoder.EncodeAll(raw, make([]byte, 0, len(raw)/2))
	if len(compressed) >= len(raw) {
		return
	}
	setOpenAIUpstreamRequestBody(req, compressed)
	req.Header.Set("Content-Encoding", "zstd")
}

func setOpenAIUpstreamRequestBody(req *http.Request, body []byte) {
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))
	req.Header.Del("Content-Length")
}
