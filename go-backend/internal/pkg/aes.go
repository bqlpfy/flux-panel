package pkg

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

// AESCrypto AES-256-GCM 加解密工具（與 Java 端 AESCrypto 格式相容）
// 密鑰 = SHA-256(secret) → 32 bytes
// 加密格式 = Base64( 12-byte nonce + ciphertext + 16-byte GCM tag )
type AESCrypto struct {
	gcm cipher.AEAD
}

// NewAESCrypto 以 secret 字串建立 AES-GCM 加解密器
func NewAESCrypto(secret string) (*AESCrypto, error) {
	if secret == "" {
		return nil, fmt.Errorf("secret 不能為空")
	}

	// SHA-256 → 32-byte key
	hash := sha256.Sum256([]byte(secret))

	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, fmt.Errorf("AES cipher 初始化失敗: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("GCM 初始化失敗: %w", err)
	}

	return &AESCrypto{gcm: gcm}, nil
}

// Encrypt 加密位元組 → Base64(nonce + ciphertext)
func (a *AESCrypto) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, a.gcm.NonceSize()) // 12 bytes
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失敗: %w", err)
	}

	// Seal: nonce + ciphertext(含 GCM tag)
	ciphertext := a.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// EncryptString 加密字串
func (a *AESCrypto) EncryptString(plaintext string) (string, error) {
	return a.Encrypt([]byte(plaintext))
}

// Decrypt Base64 → 分離 nonce + ciphertext → 解密
func (a *AESCrypto) Decrypt(encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("Base64 解碼失敗: %w", err)
	}

	nonceSize := a.gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("密文長度不足")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := a.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失敗: %w", err)
	}

	return plaintext, nil
}

// DecryptString 解密為字串
func (a *AESCrypto) DecryptString(encoded string) (string, error) {
	b, err := a.Decrypt(encoded)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ──────────────────── 加密訊息包裝（與 Java EncryptedMessage 相容）────────────────────

// EncryptedMessage 加密訊息包裝器 JSON 結構
type EncryptedMessage struct {
	Encrypted bool   `json:"encrypted"`
	Data      string `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

// WrapEncrypt 將明文包裝為加密訊息 JSON
func WrapEncrypt(crypto *AESCrypto, plaintext string) (string, error) {
	if crypto == nil {
		return plaintext, nil
	}

	encrypted, err := crypto.EncryptString(plaintext)
	if err != nil {
		slog.Warn("加密失敗，發送原始訊息", "error", err)
		return plaintext, nil
	}

	msg := EncryptedMessage{
		Encrypted: true,
		Data:      encrypted,
		Timestamp: time.Now().UnixMilli(),
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return plaintext, nil
	}
	return string(b), nil
}

// UnwrapDecrypt 嘗試解密包裝訊息，非加密格式直接返回原文
func UnwrapDecrypt(crypto *AESCrypto, payload string) string {
	if crypto == nil || payload == "" {
		return payload
	}

	var msg EncryptedMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return payload // 非 JSON 格式
	}

	if !msg.Encrypted || msg.Data == "" {
		return payload
	}

	decrypted, err := crypto.DecryptString(msg.Data)
	if err != nil {
		slog.Warn("解密失敗，使用原始資料", "error", err)
		return payload
	}

	return decrypted
}

// ──────────────────── 加密器快取 ────────────────────

var (
	cryptoCache   = make(map[string]*AESCrypto)
	cryptoCacheMu sync.RWMutex
)

// GetOrCreateCrypto 取得或建立指定 secret 的加密器（帶快取）
func GetOrCreateCrypto(secret string) *AESCrypto {
	if secret == "" {
		return nil
	}

	cryptoCacheMu.RLock()
	if c, ok := cryptoCache[secret]; ok {
		cryptoCacheMu.RUnlock()
		return c
	}
	cryptoCacheMu.RUnlock()

	cryptoCacheMu.Lock()
	defer cryptoCacheMu.Unlock()

	// double check
	if c, ok := cryptoCache[secret]; ok {
		return c
	}

	c, err := NewAESCrypto(secret)
	if err != nil {
		slog.Error("建立 AES 加密器失敗", "error", err)
		return nil
	}
	cryptoCache[secret] = c
	return c
}
