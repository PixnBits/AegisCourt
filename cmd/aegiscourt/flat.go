package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cbergoon/merkletree"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	PrevHash    string          `json:"prev_hash"`
	PayloadHash string          `json:"payload_hash"`
	Data        json.RawMessage `json:"data"`
	Sig         string          `json:"sig"`
}

// MerkleContent implements merkletree.Content.
type MerkleContent struct {
	Data []byte
}

func (m MerkleContent) CalculateHash() ([]byte, error) {
	return m.Data, nil
}

func (m MerkleContent) Equals(other merkletree.Content) (bool, error) {
	return string(m.Data) == string(other.(MerkleContent).Data), nil
}

// FlatFileAuditStore implements AuditStore using flat files with signatures and Merkle tree.
type FlatFileAuditStore struct {
	filePath   string
	prevHash   string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	tree       *merkletree.MerkleTree
	entries    []merkletree.Content
}

func NewFlatFileAuditStore(filePath string, priv ed25519.PrivateKey, pub ed25519.PublicKey) *FlatFileAuditStore {
	return &FlatFileAuditStore{
		filePath:   filePath,
		prevHash:   "",
		privateKey: priv,
		publicKey:  pub,
		entries:    []merkletree.Content{},
	}
}

func (a *FlatFileAuditStore) Append(entry json.RawMessage) error {
	// Canonicalize the JSON for consistent hashing
	var temp interface{}
	if err := json.Unmarshal(entry, &temp); err != nil {
		return fmt.Errorf("unmarshal entry for hash: %w", err)
	}
	dataBytes, err := json.Marshal(temp)
	if err != nil {
		return fmt.Errorf("marshal entry for hash: %w", err)
	}
	payloadHash := sha256.Sum256(dataBytes)
	payloadHashStr := hex.EncodeToString(payloadHash[:])

	auditEntry := AuditEntry{
		PrevHash:    a.prevHash,
		PayloadHash: payloadHashStr,
		Data:        entry,
		Sig:         "",
	}

	auditBytes, err := json.Marshal(auditEntry)
	if err != nil {
		return err
	}

	// Sign the audit entry
	sig := ed25519.Sign(a.privateKey, auditBytes)
	auditEntry.Sig = hex.EncodeToString(sig)

	// Re-marshal with sig
	auditBytes, err = json.Marshal(auditEntry)
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

	// Add to Merkle tree
	a.entries = append(a.entries, MerkleContent{Data: auditBytes})
	tree, err := merkletree.NewTree(a.entries)
	if err != nil {
		return err
	}
	a.tree = tree

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

func (a *FlatFileAuditStore) GetMerkleRoot() []byte {
	if a.tree == nil {
		return nil
	}
	return a.tree.MerkleRoot()
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
		// Verify signature
		sigBytes, err := hex.DecodeString(entry.Sig)
		if err != nil {
			return fmt.Errorf("invalid sig hex")
		}
		entryForSig := entry
		entryForSig.Sig = ""
		entryBytes, _ := json.Marshal(entryForSig)
		if !ed25519.Verify(a.publicKey, entryBytes, sigBytes) {
			return fmt.Errorf("signature verification failed")
		}
		// Verify payload hash
		var temp interface{}
		if err := json.Unmarshal(entry.Data, &temp); err != nil {
			return fmt.Errorf("unmarshal data for hash: %w", err)
		}
		dataBytes, _ := json.Marshal(temp)
		payloadHash := sha256.Sum256(dataBytes)
		if hex.EncodeToString(payloadHash[:]) != entry.PayloadHash {
			return fmt.Errorf("payload hash mismatch")
		}
		// Update expectedPrev
		entryBytes, _ = json.Marshal(entry)
		entryHash := sha256.Sum256(entryBytes)
		expectedPrev = hex.EncodeToString(entryHash[:])
	}
	// Verify Merkle tree
	if a.tree != nil {
		valid, err := a.tree.VerifyTree()
		if err != nil {
			return fmt.Errorf("merkle tree verification error: %w", err)
		}
		if !valid {
			return fmt.Errorf("merkle tree integrity check failed")
		}
	}
	return nil
}
