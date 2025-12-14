package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"
)

// GetAES32ByteKey converts any password string to a 32-byte AES key using SHA-256
// AES-256 requires exactly 32 bytes, SHA-256 always outputs 32 bytes
func GetAES32ByteKey(password string) []byte {
	sum256 := sha256.Sum256([]byte(password))
	return sum256[:]
}

// encrypts file content using AES-256-GCM
// returns encrypted data with nonce prepended: [nonce][ciphertext][auth_tag]
func EncryptFileContent(content []byte, password string) []byte {
	key := GetAES32ByteKey(password)

	// create AES cipher
	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}

	// create GCM mode (authenticated encryption)
	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return nil
	}

	// generate random nonce (number used once)
	// CRITICAL: Must be unique for each encryption with same key
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil
	}

	// encrypt and authenticate
	// seal prepends nonce to output: [nonce][encrypted_data][auth_tag]
	ciphertext := gcm.Seal(nonce, nonce, content, nil)
	return ciphertext
}

// decrypts AES-256-GCM encrypted data
// Expects format: [nonce][ciphertext][auth_tag]
func DecryptFileContent(ciphertext []byte, password string) []byte {
	key := GetAES32ByteKey(password)

	// create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}

	// create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil
	}

	// extract nonce from beginning
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// decrypt and verify authentication tag
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Wrong password or tampered data
		return nil
	}

	return plaintext
}
