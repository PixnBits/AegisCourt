package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadConstitution(t *testing.T) {
	consti, err := loadConstitution("../../docs/constitution.md")
	if err != nil {
		t.Fatalf("load constitution: %v", err)
	}
	if len(consti) == 0 {
		t.Error("constitution is empty")
	}
	if !strings.Contains(consti, "AegisCourt Constitution") {
		t.Error("constitution does not contain expected content")
	}
}

func TestKernelSignVerify(t *testing.T) {
	k, err := NewKernel("../../config.json")
	if err != nil {
		t.Fatalf("new kernel: %v", err)
	}
	data := []byte("test data")
	sig, err := k.Sign(data)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !k.VerifySignature(data, sig) {
		t.Error("signature verification failed")
	}
}

func TestAuditStore(t *testing.T) {
	// Generate test keys
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate keys: %v", err)
	}
	
	// Create temp file
	tmpFile, err := os.CreateTemp("", "audit_test_*.log")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	
	store := NewFlatFileAuditStore(tmpFile.Name(), priv, pub)
	entry := json.RawMessage(`{"test": "data"}`)
	err = store.Append(entry)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	err = store.VerifyIntegrity()
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	history, err := store.GetHistory(time.Time{})
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 entry, got %d", len(history))
	}
}

func TestGvisorSandbox(t *testing.T) {
	sb := &GvisorSandbox{}
	ctx := context.Background()
	// Note: This test assumes Docker and gvisor runtime are available
	// For CI, this might be skipped or mocked
	err := sb.Start(ctx, []string{"echo", "test"})
	if err != nil {
		t.Skipf("gvisor not available: %v", err)
	}
	defer sb.Stop()
	output, err := sb.Exec("input")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	// Since Exec is stub, check it's not empty
	if output == "" {
		t.Error("expected output")
	}
}
