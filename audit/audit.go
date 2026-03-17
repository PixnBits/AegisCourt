package audit

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/uuid/v5"
)

var rootPrivateKey ed25519.PrivateKey
var rootPublicKey ed25519.PublicKey
var auditFilePath string

func init() {
	// Hard-coded root key pair (for demo; in real, generate and secure)
	pubHex := "cefbf1f37dd2ccb8fd2cec9230a2fa3f213a44323f0a712cc68380240173c1bd"
	privHex := "27277f311dfa688756fb5312aad7136dc496f35c16f50d69f7308c2c336447afcefbf1f37dd2ccb8fd2cec9230a2fa3f213a44323f0a712cc68380240173c1bd"

	var err error
	rootPublicKey, err = hex.DecodeString(pubHex)
	if err != nil {
		panic(fmt.Sprintf("failed to decode public key: %v", err))
	}
	rootPrivateKey, err = hex.DecodeString(privHex)
	if err != nil {
		panic(fmt.Sprintf("failed to decode private key: %v", err))
	}

	auditFilePath, err = getAuditFilePath()
	if err != nil {
		panic(err)
	}
}

type Entry struct {
	UUID        string `json:"uuid"`
	Timestamp   string `json:"timestamp"`
	PrevHash    string `json:"prev_hash"`
	PayloadHash string `json:"payload_hash"`
	Signature   string `json:"signature"`
}

func (e *Entry) ComputeHash() string {
	data := e.UUID + e.Timestamp + e.PrevHash + e.PayloadHash
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (e *Entry) Sign() error {
	hashStr := e.ComputeHash()
	hash := sha256.Sum256([]byte(hashStr))
	sig := ed25519.Sign(rootPrivateKey, hash[:])
	e.Signature = hex.EncodeToString(sig)
	return nil
}

func (e *Entry) Verify() error {
	hashStr := e.ComputeHash()
	hash := sha256.Sum256([]byte(hashStr))
	sig, err := hex.DecodeString(e.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature hex: %v", err)
	}
	if !ed25519.Verify(rootPublicKey, hash[:], sig) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func getAuditFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".aegiscourt")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "audit.log"), nil
}

func Append(payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	payloadHash := sha256.Sum256(payloadJSON)
	payloadHashStr := hex.EncodeToString(payloadHash[:])

	// Get prev hash
	prevHash, err := getLastHash()
	if err != nil {
		return err
	}

	entry := Entry{
		UUID:        uuid.Must(uuid.NewV7()).String(),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		PrevHash:    prevHash,
		PayloadHash: payloadHashStr,
	}

	if err := entry.Sign(); err != nil {
		return err
	}

	file, err := os.OpenFile(auditFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	if _, err := file.WriteString(string(entryJSON) + "\n"); err != nil {
		return err
	}

	return nil
}

func getLastHash() (string, error) {
	file, err := os.Open(auditFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // first entry
		}
		return "", err
	}
	defer file.Close()

	var lastEntry Entry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &lastEntry); err != nil {
			return "", err
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	if lastEntry.UUID == "" {
		return "", nil
	}

	return lastEntry.ComputeHash(), nil
}

func Verify() (bool, []error) {
	file, err := os.Open(auditFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // no file, intact
		}
		return false, []error{err}
	}
	defer file.Close()

	var errors []error
	var prevHash string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			errors = append(errors, fmt.Errorf("unmarshal error: %v", err))
			continue
		}

		if entry.PrevHash != prevHash {
			errors = append(errors, fmt.Errorf("prev_hash mismatch for entry %s", entry.UUID))
		}

		if err := entry.Verify(); err != nil {
			errors = append(errors, fmt.Errorf("verification error for entry %s: %v", entry.UUID, err))
		}

		prevHash = entry.ComputeHash()
	}

	if err := scanner.Err(); err != nil {
		errors = append(errors, err)
	}

	return len(errors) == 0, errors
}