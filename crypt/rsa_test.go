package crypt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

func TestNewRSADecrypter(t *testing.T) {

	tempPEMFile := "test_key.pem"
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("Failed to generate test private key: %v", err)
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	pemData := pem.EncodeToMemory(pemBlock)

	err = os.WriteFile(tempPEMFile, pemData, 0644)
	if err != nil {
		t.Fatalf("Failed to write test PEM file: %v", err)
	}
	defer os.Remove(tempPEMFile)

	rsaObj, err := NewRSADecrypter(tempPEMFile)
	if err != nil {
		t.Errorf("Expected no error when loading PEM file, got: %v", err)
	}
	if rsaObj.privateKey == nil {
		t.Error("Expected private key to be loaded, but it was nil")
	}
}

func TestNewRSADecrypter_FileNotFound(t *testing.T) {
	_, err := NewRSADecrypter("non_existent.pem")
	if err == nil {
		t.Error("Expected error when loading non-existent PEM file, got none")
	}
}

func TestLoadPEM_InvalidPEM(t *testing.T) {
	rsaObj := &RSA{}

	tempPEMFile := "invalid_key.pem"
	err := os.WriteFile(tempPEMFile, []byte("INVALID PEM DATA"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid test PEM file: %v", err)
	}
	defer os.Remove(tempPEMFile)

	err = rsaObj.LoadPEM(tempPEMFile)
	if err == nil {
		t.Error("Expected error when loading invalid PEM file, got none")
	}
}

func TestDecryptNoPadding_InvalidCiphertextLength(t *testing.T) {
	rsaObj := &RSA{}

	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("Failed to generate test private key: %v", err)
	}
	rsaObj.privateKey = privateKey

	invalidCiphertext := make([]byte, 64) // Invalid size, should be 128 bytes
	_, err = rsaObj.DecryptNoPadding(invalidCiphertext)
	if err == nil {
		t.Error("Expected error for invalid ciphertext length, got none")
	}
}

func TestDecryptNoPadding_ValidCiphertext(t *testing.T) {
	rsaObj := &RSA{}

	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("Failed to generate test private key: %v", err)
	}
	rsaObj.privateKey = privateKey

	// Prepare a valid ciphertext (dummy ciphertext for simplicity)
	ciphertext := make([]byte, 128) // Correct size

	// Decrypt the ciphertext
	plaintext, err := rsaObj.DecryptNoPadding(ciphertext)
	if err != nil {
		t.Errorf("Expected no error for valid ciphertext, got: %v", err)
	}

	if len(plaintext) != 128 {
		t.Errorf("Expected plaintext length to be 128, got: %d", len(plaintext))
	}
}
