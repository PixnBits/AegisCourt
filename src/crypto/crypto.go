package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"

	"golang.org/x/crypto/ed25519"
)

// GenerateKeyPair generates a new Ed25519 key pair.
func GenerateKeyPair() (publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey, err error) {
	publicKey, privateKey, err = ed25519.GenerateKey(rand.Reader)
	return
}

// Sign signs the data with the private key and returns the signature.
func Sign(data []byte, privateKey ed25519.PrivateKey) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("private key is nil")
	}
	return ed25519.Sign(privateKey, data), nil
}

// Verify verifies the signature of the data with the public key.
func Verify(data []byte, signature []byte, publicKey ed25519.PublicKey) bool {
	if publicKey == nil || signature == nil {
		return false
	}
	return ed25519.Verify(publicKey, data, signature)
}

// Hash computes the SHA-256 hash of the data.
func Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}
