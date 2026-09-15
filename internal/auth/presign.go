package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// GeneratePresignedURL generates a MinIO/S3-style presigned URL using HMAC-SHA256
func GeneratePresignedURL(method, baseURL, path, accessKey, secretKey string, expiry time.Duration) (string, error) {
	parsedURL, err := url.Parse(baseURL + path)
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(expiry).Unix()
	
	// Set query params
	query := parsedURL.Query()
	query.Set("X-Amz-Algorithm", "HMAC-SHA256")
	query.Set("X-Amz-Credential", accessKey)
	query.Set("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))
	query.Set("X-Amz-Expires", strconv.FormatInt(int64(expiry.Seconds()), 10))
	query.Set("Expires", strconv.FormatInt(expiresAt, 10))

	parsedURL.RawQuery = query.Encode()

	// Create string to sign
	stringToSign := fmt.Sprintf("%s\n%s\n%s", method, parsedURL.Path, parsedURL.RawQuery)

	// Generate HMAC signature
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(stringToSign))
	signature := hex.EncodeToString(h.Sum(nil))

	// Append signature to query
	query.Set("X-Amz-Signature", signature)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

// ValidatePresignedURL validates an incoming request with a presigned URL
func ValidatePresignedURL(r *http.Request, secretKey string) bool {
	query := r.URL.Query()
	
	signature := query.Get("X-Amz-Signature")
	if signature == "" {
		return false
	}

	expiresStr := query.Get("Expires")
	expiresAt, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		return false
	}

	if time.Now().Unix() > expiresAt {
		return false // Expired
	}

	// Recreate string to sign
	originalQuery := r.URL.Query()
	originalQuery.Del("X-Amz-Signature")
	
	stringToSign := fmt.Sprintf("%s\n%s\n%s", r.Method, r.URL.Path, originalQuery.Encode())

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(stringToSign))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
