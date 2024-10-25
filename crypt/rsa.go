package crypt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
)

type Decrypter interface {
	DecryptNoPadding(ciphertext []byte) ([]byte, error)
}

type RSA struct {
	privateKey *rsa.PrivateKey
}

func NewRSADecrypter(pemFile string) (*RSA, error) {
	r := &RSA{}
	err := r.LoadPEM(pemFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load RSA private key from %s: %w", pemFile, err)
	}
	return r, nil
}

func (r *RSA) LoadPEM(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filename, err)
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return fmt.Errorf("failed to decode PEM block containing private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse RSA private key: %v", err)
	}

	r.privateKey = privateKey
	return nil
}

func (r *RSA) DecryptNoPadding(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) != 128 { // Ensure it's 128 bytes for a 1024-bit RSA key
		return nil, fmt.Errorf("invalid ciphertext length: %d", len(ciphertext))
	}

	// Convert ciphertext to big.Int
	c := new(big.Int).SetBytes(ciphertext)

	// Perform the raw RSA decryption: m = c^d mod n
	m := new(big.Int).Exp(c, r.privateKey.D, r.privateKey.N)

	// Extract the plaintext bytes
	plaintext := m.Bytes()

	// Since m.Bytes() might return fewer than 128 bytes if the plaintext has leading zeros,
	// pad it manually if needed to get the exact size.
	if len(plaintext) < 128 {
		padding := make([]byte, 128-len(plaintext))
		plaintext = append(padding, plaintext...)
	}

	return plaintext, nil
}
