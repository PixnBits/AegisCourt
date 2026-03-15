package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	PrevHash    string          `json:"prev_hash"`
	PayloadHash string          `json:"payload_hash"`
	Data        json.RawMessage `json:"data"`
	Sig         string          `json:"sig"`
}

// FlatFileAuditStore implements AuditStore using flat files with signatures.
type FlatFileAuditStore struct {
	filePath string
	prevHash string
}

func NewFlatFileAuditStore(filePath string) *FlatFileAuditStore {
	return &FlatFileAuditStore{
		filePath: filePath,
		prevHash: "",
	}
}

func (a *FlatFileAuditStore) Append(entry json.RawMessage) error {
	payloadHash := sha256.Sum256(entry)
	payloadHashStr := hex.EncodeToString(payloadHash[:])

	auditEntry := AuditEntry{
		PrevHash:    a.prevHash,
		PayloadHash: payloadHashStr,
		Data:        entry,
		Sig:         "", // TODO: sign with kernel key
	}

	auditBytes, err := json.Marshal(auditEntry)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(a.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(string(auditBytes) + "\n"); err != nil {
		return err
	}

	// Update prevHash
	entryHash := sha256.Sum256(auditBytes)
	a.prevHash = hex.EncodeToString(entryHash[:])

	return nil
}

func (a *FlatFileAuditStore) GetHistory(since time.Time) ([]json.RawMessage, error) {
	data, err := os.ReadFile(a.filePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var history []json.RawMessage
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry AuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}
		history = append(history, entry.Data)
	}
	return history, nil
}

func (a *FlatFileAuditStore) VerifyIntegrity() error {
	data, err := os.ReadFile(a.filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	expectedPrev := ""
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry AuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return err
		}
		if entry.PrevHash != expectedPrev {
			return fmt.Errorf("chain broken at entry")
		}
		// Verify payload hash
		payloadHash := sha256.Sum256(entry.Data)
		if hex.EncodeToString(payloadHash[:]) != entry.PayloadHash {
			return fmt.Errorf("payload hash mismatch")
		}
		// Update expectedPrev
		entryBytes, _ := json.Marshal(entry)
		entryHash := sha256.Sum256(entryBytes)
		expectedPrev = hex.EncodeToString(entryHash[:])
	}
	return nil
}
