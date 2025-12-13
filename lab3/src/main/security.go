package main

import (
	"crypto/aes"
	cipher2 "crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"
)

// random password -> 32 byte for AES(256)
func GetAES32ByteKey(password string) []byte {
	sum256 := sha256.Sum256([]byte(password))
	return sum256[:]
}
func EncryptFileContent(content []byte, password string) []byte {
	key := GetAES32ByteKey(password)
	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	gcm, err := cipher2.NewGCM(cipher)
	if err != nil {
		return nil
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil
	}
	ciphertext := gcm.Seal(nonce, nonce, content, nil)
	return ciphertext
}

func DecryptFileContent(ciphertext []byte, password string) []byte {
	key := GetAES32ByteKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}

	aes, err := cipher2.NewGCM(block)
	if err != nil {
		return nil
	}
	nonceSize := aes.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aes.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil
	}

	return plaintext

}
