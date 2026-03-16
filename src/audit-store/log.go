package auditstore

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aegiscourt/src/crypto"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	ID          string             `json:"id"`
	Timestamp   time.Time          `json:"timestamp"`
	PrevHash    []byte             `json:"prev_hash"`
	Payload     []byte             `json:"payload"` // JSON of event
	Signature   []byte             `json:"signature"`
	MerkleProof []crypto.ProofItem `json:"merkle_proof"`
}

// AuditLog manages an append-only audit log using Merkle tree.
type AuditLog struct {
	entries    []*AuditEntry
	merkleTree *crypto.MerkleTree
	logFile    string
	privKey    ed25519.PrivateKey
	pubKey     ed25519.PublicKey
}

// NewAuditLog creates a new audit log.
func NewAuditLog(logPath string, privKey ed25519.PrivateKey, pubKey ed25519.PublicKey) *AuditLog {
	return &AuditLog{
		entries:    make([]*AuditEntry, 0),
		merkleTree: crypto.NewMerkleTree(),
		logFile:    logPath,
		privKey:    privKey,
		pubKey:     pubKey,
	}
}

// Append adds a new entry to the audit log.
func (al *AuditLog) Append(eventType string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	entryData := map[string]interface{}{
		"type":    eventType,
		"payload": payload,
	}
	entryBytes, err := json.Marshal(entryData)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %v", err)
	}

	// Sign the entry
	signature := ed25519.Sign(al.privKey, entryBytes)

	// Create entry
	entry := &AuditEntry{
		ID:        fmt.Sprintf("%d", len(al.entries)+1), // Simple ID
		Timestamp: time.Now(),
		PrevHash:  al.merkleTree.GetRoot(),
		Payload:   payloadBytes,
		Signature: signature,
	}

	// Append to Merkle tree
	al.merkleTree.Append(entryBytes)
	entry.MerkleProof, _ = al.merkleTree.GetProof(len(al.entries))

	al.entries = append(al.entries, entry)

	// Append to file
	return al.appendToFile(entry)
}

// GetRootHash returns the current Merkle root.
func (al *AuditLog) GetRootHash() []byte {
	return al.merkleTree.GetRoot()
}

// VerifyEntry verifies the integrity of an entry.
func (al *AuditLog) VerifyEntry(id string) (bool, error) {
	// Find entry
	var entry *AuditEntry
	for _, e := range al.entries {
		if e.ID == id {
			entry = e
			break
		}
	}
	if entry == nil {
		return false, fmt.Errorf("entry not found")
	}

	// Verify signature
	entryData := map[string]interface{}{
		"type":    "unknown", // TODO: store type in entry
		"payload": entry.Payload,
	}
	entryBytes, _ := json.Marshal(entryData)
	if !ed25519.Verify(al.pubKey, entryBytes, entry.Signature) {
		return false, nil
	}

	// Verify Merkle proof
	return crypto.VerifyProof(crypto.Hash(entryBytes), entry.MerkleProof, al.GetRootHash()), nil
}

// ExportJSONL exports the log to a JSONL file.
func (al *AuditLog) ExportJSONL(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, entry := range al.entries {
		data, _ := json.Marshal(entry)
		file.WriteString(string(data) + "\n")
	}
	return nil
}

// appendToFile appends the entry to the log file.
func (al *AuditLog) appendToFile(entry *AuditEntry) error {
	// Ensure directory exists
	os.MkdirAll(filepath.Dir(al.logFile), 0700)

	file, err := os.OpenFile(al.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	data, _ := json.Marshal(entry)
	_, err = file.WriteString(string(data) + "\n")
	return err
}
