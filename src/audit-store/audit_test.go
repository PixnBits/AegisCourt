package auditstore

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditLog(t *testing.T) {
	// Create temp dir
	tempDir, err := os.MkdirTemp("", "audit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logPath := filepath.Join(tempDir, "audit.log.jsonl")

	// Generate keys
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	log := NewAuditLog(logPath, privKey, pubKey)

	// Append entries
	err = log.Append("test_event", map[string]string{"key": "value1"})
	if err != nil {
		t.Fatal(err)
	}
	err = log.Append("test_event", map[string]string{"key": "value2"})
	if err != nil {
		t.Fatal(err)
	}

	// Check root
	root := log.GetRootHash()
	if len(root) == 0 {
		t.Error("Root hash is empty")
	}

	// Verify entry
	valid, err := log.VerifyEntry("1")
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Error("Entry verification failed")
	}

	// Export
	exportPath := filepath.Join(tempDir, "export.jsonl")
	err = log.ExportJSONL(exportPath)
	if err != nil {
		t.Fatal(err)
	}

	// Check file exists
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Error("Export file not created")
	}
}

func TestAuditStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Generate keys
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	store := NewAuditStore(tempDir, privKey, pubKey)

	err = store.LogEvent("test", map[string]string{"msg": "hello"})
	if err != nil {
		t.Fatal(err)
	}

	root := store.GetRootHash()
	if len(root) == 0 {
		t.Error("Root hash empty")
	}
}
