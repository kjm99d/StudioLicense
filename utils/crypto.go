package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// GenerateLicenseKey 라이선스 키 생성 (형식: XXXX-XXXX-XXXX-XXXX)
func GenerateLicenseKey() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	key := hex.EncodeToString(bytes)
	key = strings.ToUpper(key)

	// 4자리씩 끊어서 대시로 연결
	formatted := fmt.Sprintf("%s-%s-%s-%s",
		key[0:4],
		key[4:8],
		key[8:12],
		key[12:16],
	)

	return formatted, nil
}

// GenerateID UUID 스타일 ID 생성
func GenerateID(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	id := hex.EncodeToString(bytes)
	if prefix != "" {
		return fmt.Sprintf("%s-%s", prefix, id[:16]), nil
	}
	return id[:16], nil
}

// GenerateDeviceFingerprint 디바이스 정보로 핑거프린트 생성
func GenerateDeviceFingerprint(cpuID, motherboardSN, macAddr, diskSerial, machineID string) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s",
		cpuID,
		motherboardSN,
		macAddr,
		diskSerial,
		machineID,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// HashPassword 비밀번호 해싱
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword 비밀번호 검증
func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GenerateTempPassword 임시 비밀번호 생성
func GenerateTempPassword(length int) string {
	if length < 8 {
		length = 8
	}
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%"
	b := make([]byte, length)
	for i := range b {
		randomByte := make([]byte, 1)
		rand.Read(randomByte)
		b[i] = charset[randomByte[0]%byte(len(charset))]
	}
	return string(b)
}
