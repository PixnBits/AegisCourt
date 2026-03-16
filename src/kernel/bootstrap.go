package kernel

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aegiscourt/src/crypto"
)

// Bootstrap performs the kernel bootstrap process.
// It generates a key pair, signs the kernel binary, stores the signature,
// loads the constitution, and creates an initial log entry.
func Bootstrap() (kernelHash string, err error) {
	// Get the kernel binary path
	binaryPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Read and hash the binary
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to read binary: %w", err)
	}
	binaryHash := crypto.Hash(binaryData)
	kernelHash = hex.EncodeToString(binaryHash)

	// Generate key pair
	pubKey, privKey, err := crypto.GenerateKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Sign the binary hash
	signature, err := crypto.Sign(binaryHash, privKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign binary: %w", err)
	}

	// Store signature: public key + signature
	sigData := append(pubKey, signature...)
	configDir := getConfigDir()
	sigPath := filepath.Join(configDir, "kernel.sig")
	err = os.MkdirAll(configDir, 0700)
	if err != nil {
		return "", fmt.Errorf("failed to create config dir: %w", err)
	}
	err = os.WriteFile(sigPath, sigData, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write signature: %w", err)
	}

	// Load constitution
	constitution, err := loadConstitution()
	if err != nil {
		return "", fmt.Errorf("failed to load constitution: %w", err)
	}
	_ = constitution // TODO: parse later

	// Create initial log entry
	logEntry := fmt.Sprintf("Kernel bootstrapped at %s with hash %s", time.Now().Format(time.RFC3339), kernelHash)
	logPath := filepath.Join(configDir, "bootstrap.log")
	err = os.WriteFile(logPath, []byte(logEntry), 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write log: %w", err)
	}

	return kernelHash, nil
}

// VerifyKernel checks the integrity of the kernel binary.
// It panics if the binary has been tampered with.
func VerifyKernel() error {
	// Get binary path
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Read binary and hash
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to read binary: %w", err)
	}
	binaryHash := crypto.Hash(binaryData)

	// Read signature
	configDir := getConfigDir()
	sigPath := filepath.Join(configDir, "kernel.sig")
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		return fmt.Errorf("failed to read signature: %w", err)
	}
	if len(sigData) != 32+64 { // pub 32, sig 64
		return fmt.Errorf("invalid signature file")
	}
	pubKey := sigData[:32]
	signature := sigData[32:]

	// Verify
	if !crypto.Verify(binaryHash, signature, pubKey) {
		panic("Kernel integrity violation: binary has been tampered with. Halting per Constitution Rule 1.")
	}

	return nil
}

// loadConstitution loads the constitution from docs/constitution.md
var loadConstitution = func() (string, error) {
	constitutionPath := "docs/constitution.md"
	data, err := os.ReadFile(constitutionPath)
	if err != nil {
		return "", fmt.Errorf("failed to read constitution: %w", err)
	}
	return string(data), nil
}

// getConfigDir returns the config directory ~/.aegiscourt
var getConfigDir = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aegiscourt")
}
