package pkg

import (
	"crypto/md5"
	"encoding/hex"
)

const defaultSalt = "admin_salt_2024"

// MD5 基礎 MD5 雜湊
func MD5(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// MD5WithSalt 帶鹽值的 MD5 雜湊（與 Java Md5Util.md5WithSalt 一致）
func MD5WithSalt(input string) string {
	return MD5(input + defaultSalt)
}

// MD5WithCustomSalt 帶自定義鹽值
func MD5WithCustomSalt(input, salt string) string {
	return MD5(input + salt)
}

// VerifyPassword 驗證密碼
func VerifyPassword(password, hashedPassword string) bool {
	return MD5WithSalt(password) == hashedPassword
}
