package keys

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// RootPublicKeyHex is the embedded root public key.
// On first `init`, a keypair is generated and this gets replaced by the build.
// For development, we generate and store keys at runtime.
const RootPublicKeyHex = ""

func AegisCourtDir() (string, error) {
	if dir := os.Getenv("AEGISCOURT_HOME"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".aegiscourt"), nil
}

func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0700)
}

func KeyDir() (string, error) {
	base, err := AegisCourtDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "keys"), nil
}

func GenerateAndStoreKeypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("key generation failed: %w", err)
	}
	dir, err := KeyDir()
	if err != nil {
		return nil, nil, err
	}
	if err := EnsureDir(dir); err != nil {
		return nil, nil, fmt.Errorf("cannot create key directory: %w", err)
	}
	pubPath := filepath.Join(dir, "root.pub")
	privPath := filepath.Join(dir, "root.key")
	if err := os.WriteFile(pubPath, []byte(hex.EncodeToString(pub)), 0600); err != nil {
		return nil, nil, fmt.Errorf("failed to write public key: %w", err)
	}
	if err := os.WriteFile(privPath, []byte(hex.EncodeToString(priv)), 0600); err != nil {
		return nil, nil, fmt.Errorf("failed to write private key: %w", err)
	}
	return pub, priv, nil
}

func LoadKeypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	dir, err := KeyDir()
	if err != nil {
		return nil, nil, err
	}
	pubBytes, err := os.ReadFile(filepath.Join(dir, "root.pub"))
	if err != nil {
		return nil, nil, fmt.Errorf("no root public key found (run 'aegiscourt init' first): %w", err)
	}
	pub, err := hex.DecodeString(string(pubBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("corrupt public key: %w", err)
	}
	privBytes, err := os.ReadFile(filepath.Join(dir, "root.key"))
	if err != nil {
		return nil, nil, fmt.Errorf("no root private key found: %w", err)
	}
	priv, err := hex.DecodeString(string(privBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("corrupt private key: %w", err)
	}
	return ed25519.PublicKey(pub), ed25519.PrivateKey(priv), nil
}

func Sign(priv ed25519.PrivateKey, message []byte) []byte {
	return ed25519.Sign(priv, message)
}

func Verify(pub ed25519.PublicKey, message, sig []byte) bool {
	return ed25519.Verify(pub, message, sig)
}

func PublicKeyHex(pub ed25519.PublicKey) string {
	return hex.EncodeToString(pub)
}
