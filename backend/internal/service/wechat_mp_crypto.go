package service

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1" //nolint:gosec // WeChat's message signature is defined as SHA-1
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// 公众号消息加解密（安全模式）：
//   key  = base64(EncodingAESKey + "=")，32 字节；IV = key[:16]
//   明文 = random(16) + len(msg)(4 字节大端) + msg + appid，PKCS#7 补位，块大小 32
//   AES-256-CBC 加密后 base64。

const weChatMPPKCS7BlockSize = 32

// WeChatMPSignature is sha1 of the sorted, concatenated parts (token, timestamp, nonce[, encrypt]).
func WeChatMPSignature(parts ...string) string {
	sorted := append([]string(nil), parts...)
	sort.Strings(sorted)
	sum := sha1.Sum([]byte(strings.Join(sorted, ""))) //nolint:gosec // defined by WeChat
	return hex.EncodeToString(sum[:])
}

// WeChatMPSignatureValid compares in constant time.
func WeChatMPSignatureValid(signature string, parts ...string) bool {
	signature = strings.ToLower(strings.TrimSpace(signature))
	if signature == "" {
		return false
	}
	expected := WeChatMPSignature(parts...)
	return subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) == 1
}

// WeChatMPCrypter encrypts and decrypts 安全模式 messages for one 公众号.
type WeChatMPCrypter struct {
	key   []byte
	appID string
	rand  io.Reader
}

// NewWeChatMPCrypter builds a crypter from the 43-character EncodingAESKey.
func NewWeChatMPCrypter(encodingAESKey, appID string, random io.Reader) (*WeChatMPCrypter, error) {
	key, err := decodeWeChatMPAESKey(strings.TrimSpace(encodingAESKey))
	if err != nil {
		return nil, err
	}
	if random == nil {
		return nil, errors.New("wechat mp crypter needs a random source")
	}
	return &WeChatMPCrypter{key: key, appID: strings.TrimSpace(appID), rand: random}, nil
}

// Encrypt returns base64(AES-256-CBC(random16 + len + msg + appid)).
func (c *WeChatMPCrypter) Encrypt(msg []byte) (string, error) {
	random := make([]byte, 16)
	if _, err := io.ReadFull(c.rand, random); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	var buf bytes.Buffer
	_, _ = buf.Write(random)
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(msg))) //nolint:gosec // message size is bounded by the request body limit
	_, _ = buf.Write(length[:])
	_, _ = buf.Write(msg)
	_, _ = buf.WriteString(c.appID)
	plain := weChatMPPKCS7Pad(buf.Bytes())

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("new aes cipher: %w", err)
	}
	out := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, c.key[:16]).CryptBlocks(out, plain)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt reverses Encrypt and checks that the trailing app id is this 公众号's.
func (c *WeChatMPCrypter) Decrypt(encrypted string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encrypted))
	if err != nil {
		return nil, fmt.Errorf("decode encrypted message: %w", err)
	}
	if len(raw) == 0 || len(raw)%aes.BlockSize != 0 {
		return nil, errors.New("encrypted message has a bad length")
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("new aes cipher: %w", err)
	}
	plain := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, c.key[:16]).CryptBlocks(plain, raw)
	plain, err = weChatMPPKCS7Unpad(plain)
	if err != nil {
		return nil, err
	}
	if len(plain) < 20 {
		return nil, errors.New("decrypted message is too short")
	}
	msgLen := int(binary.BigEndian.Uint32(plain[16:20]))
	if msgLen < 0 || 20+msgLen > len(plain) {
		return nil, errors.New("decrypted message length is out of range")
	}
	msg := plain[20 : 20+msgLen]
	appID := string(plain[20+msgLen:])
	if c.appID != "" && subtle.ConstantTimeCompare([]byte(appID), []byte(c.appID)) != 1 {
		return nil, errors.New("decrypted message belongs to another app id")
	}
	return msg, nil
}

func weChatMPPKCS7Pad(data []byte) []byte {
	pad := weChatMPPKCS7BlockSize - len(data)%weChatMPPKCS7BlockSize
	return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

func weChatMPPKCS7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty padded message")
	}
	pad := int(data[len(data)-1])
	if pad < 1 || pad > weChatMPPKCS7BlockSize || pad > len(data) {
		return nil, errors.New("bad message padding")
	}
	return data[:len(data)-pad], nil
}
