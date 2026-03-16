package auditstore

import (
	"crypto/ed25519"
	"path/filepath"
)

// AuditStore handles audit logging and rollback
type AuditStore struct {
	log *AuditLog
}

// NewAuditStore creates a new audit store.
func NewAuditStore(homeDir string, privKey ed25519.PrivateKey, pubKey ed25519.PublicKey) *AuditStore {
	logPath := filepath.Join(homeDir, ".aegiscourt", "audit.log.jsonl")
	return &AuditStore{
		log: NewAuditLog(logPath, privKey, pubKey),
	}
}

// LogEvent logs an event to the audit store.
func (as *AuditStore) LogEvent(eventType string, payload interface{}) error {
	return as.log.Append(eventType, payload)
}

// GetRootHash returns the current Merkle root.
func (as *AuditStore) GetRootHash() []byte {
	return as.log.GetRootHash()
}
