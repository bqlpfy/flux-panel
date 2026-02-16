package pkg

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// JWTClaims JWT 載荷內容
type JWTClaims struct {
	Sub    string `json:"sub"`
	Iat    int64  `json:"iat"`
	Exp    int64  `json:"exp"`
	User   string `json:"user"`
	Name   string `json:"name"`
	RoleID int    `json:"role_id"`
}

// GenerateToken 生成 JWT Token，與 Java 版 JwtUtil 格式完全一致
func GenerateToken(userID int64, username string, roleID int, secret string, expireHours int) (string, error) {
	now := time.Now()
	exp := now.Add(time.Duration(expireHours) * time.Hour)

	// Header: {"alg":"HmacSHA256","typ":"JWT"}
	header := map[string]string{
		"alg": "HmacSHA256",
		"typ": "JWT",
	}
	headerJSON, _ := json.Marshal(header)
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Payload
	claims := JWTClaims{
		Sub:    strconv.FormatInt(userID, 10),
		Iat:    now.Unix(),
		Exp:    exp.Unix(),
		User:   username,
		Name:   username,
		RoleID: roleID,
	}
	payloadJSON, _ := json.Marshal(claims)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Signature: HMAC-SHA256
	signature := calculateSignature(encodedHeader, encodedPayload, secret)

	return fmt.Sprintf("%s.%s.%s", encodedHeader, encodedPayload, signature), nil
}

// ValidateToken 驗證 JWT Token（constant-time 比對）
func ValidateToken(token, secret string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	encodedHeader := parts[0]
	encodedPayload := parts[1]
	signature := parts[2]

	// constant-time 簽名比對
	expectedSig := calculateSignature(encodedHeader, encodedPayload, secret)
	if subtle.ConstantTimeCompare([]byte(expectedSig), []byte(signature)) != 1 {
		return nil, fmt.Errorf("invalid signature")
	}

	// 解碼 payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding")
	}

	var claims JWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("invalid payload JSON")
	}

	// 檢查過期
	if claims.Exp < time.Now().Unix() {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// GetUserIDFromToken 快速從 token 取 user ID（不驗簽）
func GetUserIDFromToken(token string) (int64, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid token")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, err
	}
	var claims JWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return 0, err
	}
	return strconv.ParseInt(claims.Sub, 10, 64)
}

// GetRoleIDFromToken 快速從 token 取 role ID（不驗簽）
func GetRoleIDFromToken(token string) (int, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid token")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, err
	}
	var claims JWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return 0, err
	}
	return claims.RoleID, nil
}

func calculateSignature(encodedHeader, encodedPayload, secret string) string {
	content := encodedHeader + "." + encodedPayload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(content))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
