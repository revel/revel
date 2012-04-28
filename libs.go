package rev

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io"
)

// Sign a given string with the app-configured secret key.
// Return the signature in base64 (URLEncoding).
func Sign(message string) string {
	mac := hmac.New(sha1.New, secretKey)
	io.WriteString(mac, message)
	return hex.EncodeToString(mac.Sum(nil))
}
