package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"time"
)

func hmacHex(payload, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(payload))
	return hex.EncodeToString(m.Sum(nil))
}

// GenerateDownloadURL 生成带 HMAC 签名的下载链接。
func GenerateDownloadURL(baseURL, secret, orderID, sku string, ttl time.Duration) string {
	expires := time.Now().Add(ttl).Unix()
	payload := fmt.Sprintf("%s|%s|%d", orderID, sku, expires)
	sig := hmacHex(payload, secret)
	return fmt.Sprintf("%s/api/download?order_id=%s&sku=%s&expires=%d&sig=%s",
		baseURL, url.QueryEscape(orderID), url.QueryEscape(sku), expires, sig)
}

// VerifyDownload 校验下载链接是否有效。
func VerifyDownload(secret, orderID, sku string, expires int64, sig string) error {
	if time.Now().Unix() > expires {
		return errors.New("download link expired")
	}
	payload := fmt.Sprintf("%s|%s|%d", orderID, sku, expires)
	expected := hmacHex(payload, secret)
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return errors.New("invalid download signature")
	}
	return nil
}
