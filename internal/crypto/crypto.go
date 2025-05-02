// Package crypto provides functionality to encrypt and decrypt .env files
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const (
	// DefaultKeyEnvVar is the environment variable name for the encryption key
	DefaultKeyEnvVar = "ENVCIPHER_KEY"

	// KeySize is the size in bytes for the AES-256 encryption key
	KeySize = 32
)

// EncryptEnv encrypts the contents of a .env file
func EncryptEnv(filePath, outFilePath string) {
	// If no output file is specified, create one with ".enc" extension
	if outFilePath == "" {
		outFilePath = filePath + ".enc"
	}

	// Get encryption key
	key, err := getEncryptionKey()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Read the content of the .env file
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Encrypt the content
	encrypted, err := encrypt(content, key)
	if err != nil {
		fmt.Printf("Error encrypting file: %v\n", err)
		return
	}

	// Write the encrypted content to the output file
	err = ioutil.WriteFile(outFilePath, []byte(encrypted), 0644)
	if err != nil {
		fmt.Printf("Error writing to output file: %v\n", err)
		return
	}

	fmt.Printf("✅ Encrypted file saved to %s\n", outFilePath)
}

// DecryptEnv decrypts the contents of an encrypted .env file
func DecryptEnv(filePath, outFilePath string) {
	// If no output file is specified, remove ".enc" extension if present
	if outFilePath == "" {
		outFilePath = strings.TrimSuffix(filePath, ".enc")
		if outFilePath == filePath {
			// If the file didn't have .enc extension, add .decrypted
			outFilePath = filePath + ".decrypted"
		}
	}

	// Get encryption key
	key, err := getEncryptionKey()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Read the content of the encrypted file
	encryptedContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Decrypt the content
	decrypted, err := decrypt(string(encryptedContent), key)
	if err != nil {
		fmt.Printf("Error decrypting file: %v\n", err)
		return
	}

	// Write the decrypted content to the output file
	err = ioutil.WriteFile(outFilePath, decrypted, 0644)
	if err != nil {
		fmt.Printf("Error writing to output file: %v\n", err)
		return
	}

	fmt.Printf("✅ Decrypted file saved to %s\n", outFilePath)
}

// getEncryptionKey retrieves or generates an encryption key
func getEncryptionKey() ([]byte, error) {
	// Try to get key from environment variable
	keyStr := os.Getenv(DefaultKeyEnvVar)

	// If not found in environment, generate a new key (for development only)
	if keyStr == "" {
		fmt.Println("⚠️ Warning: No encryption key found in environment variable.")
		fmt.Printf("Set the %s environment variable for consistent encryption/decryption.\n", DefaultKeyEnvVar)
		fmt.Println("Generating a temporary key for this session...")

		key := make([]byte, KeySize)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("failed to generate encryption key: %w", err)
		}

		// Print the generated key for user to save
		fmt.Printf("Generated key (save this for decryption): %s\n", base64.StdEncoding.EncodeToString(key))
		return key, nil
	}

	// Decode base64-encoded key from environment
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key format: %w", err)
	}

	// Ensure key is the correct size
	if len(key) != KeySize {
		return nil, fmt.Errorf("encryption key must be %d bytes (got %d)", KeySize, len(key))
	}

	return key, nil
}

// encrypt encrypts data using AES-GCM
func encrypt(data, key []byte) (string, error) {
	// Create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cipher creation failed: %w", err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("GCM creation failed: %w", err)
	}

	// Create a nonce (number used once)
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce creation failed: %w", err)
	}

	// Encrypt the data
	ciphertext := aesGCM.Seal(nonce, nonce, data, nil)

	// Return base64-encoded ciphertext
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts data using AES-GCM
func decrypt(encryptedData string, key []byte) ([]byte, error) {
	// Decode base64-encoded ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("base64 decoding failed: %w", err)
	}

	// Create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cipher creation failed: %w", err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("GCM creation failed: %w", err)
	}

	// Ensure ciphertext is valid
	if len(ciphertext) < aesGCM.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce from ciphertext
	nonce, ciphertext := ciphertext[:aesGCM.NonceSize()], ciphertext[aesGCM.NonceSize():]

	// Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}
