package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var downloadURLSecret = []byte(getDownloadURLSecret())

func getDownloadURLSecret() string {
	if v := os.Getenv("DOWNLOAD_URL_SECRET"); v != "" {
		return v
	}
	// NOTE: 반드시 운영환경에서는 환경변수로 재정의해야 합니다.
	return "change-this-download-url-secret"
}

// GenerateSignedDownloadQuery builds a presigned query string (exp, nonce, sig) for the given file ID.
// expiresIn이 0 이하이면 기본값으로 5분을 사용합니다.
func GenerateSignedDownloadQuery(fileID string, expiresIn time.Duration) (string, error) {
	if fileID == "" {
		return "", errors.New("fileID is required")
	}

	if expiresIn <= 0 {
		expiresIn = 5 * time.Minute
	}

	expiration := time.Now().Add(expiresIn).Unix()
	nonceBytes := make([]byte, 12)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce := hex.EncodeToString(nonceBytes)

	payload := buildDownloadSignaturePayload(fileID, expiration, nonce)
	signature := signDownloadPayload(payload)

	return fmt.Sprintf("exp=%d&nonce=%s&sig=%s", expiration, nonce, signature), nil
}

// ValidateSignedDownloadRequest validates query params for the presigned download URL.
func ValidateSignedDownloadRequest(fileID, expStr, nonce, sig string) error {
	if fileID == "" || expStr == "" || nonce == "" || sig == "" {
		return errors.New("missing download signature parameters")
	}

	expiration, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid expiration: %w", err)
	}

	if time.Now().Unix() > expiration {
		return errors.New("download link has expired")
	}

	expectedPayload := buildDownloadSignaturePayload(fileID, expiration, nonce)
	expectedSig := signDownloadPayload(expectedPayload)

	// Compare in constant time
	if !hmac.Equal([]byte(expectedSig), []byte(sig)) {
		return errors.New("invalid download signature")
	}

	return nil
}

func buildDownloadSignaturePayload(fileID string, expiration int64, nonce string) string {
	return strings.Join([]string{fileID, strconv.FormatInt(expiration, 10), nonce}, "|")
}

func signDownloadPayload(payload string) string {
	mac := hmac.New(sha256.New, downloadURLSecret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
