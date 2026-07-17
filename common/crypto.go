package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Channel key encryption prefix (AES-GCM, key derived from CryptoSecret).
const channelKeyCipherPrefix = "enc:v1:"

func GenerateHMACWithKey(key []byte, data string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func GenerateHMAC(data string) string {
	h := hmac.New(sha256.New, []byte(CryptoSecret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func Password2Hash(password string) (string, error) {
	passwordBytes := []byte(password)
	hashedPassword, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)
	return string(hashedPassword), err
}

func ValidatePasswordAndHash(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func cryptoSecretKey32() []byte {
	sum := sha256.Sum256([]byte(CryptoSecret))
	return sum[:]
}

// EncryptChannelKey encrypts a channel credential for at-rest storage.
// Empty input returns empty. Already-encrypted values are returned unchanged.
func EncryptChannelKey(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if strings.HasPrefix(plain, channelKeyCipherPrefix) {
		return plain, nil
	}
	block, err := aes.NewCipher(cryptoSecretKey32())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return channelKeyCipherPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptChannelKey decrypts enc:v1:… payloads; plaintext legacy keys pass through.
func DecryptChannelKey(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, channelKeyCipherPrefix) {
		return stored, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, channelKeyCipherPrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(cryptoSecretKey32())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// IsEncryptedChannelKey reports whether value uses enc:v1: packaging.
func IsEncryptedChannelKey(stored string) bool {
	return strings.HasPrefix(stored, channelKeyCipherPrefix)
}
