package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendAndVerify(t *testing.T) {
	// Use a temp file for testing
	tempDir := t.TempDir()
	originalPath := auditFilePath
	auditFilePath = filepath.Join(tempDir, "audit.log")
	defer func() { auditFilePath = originalPath }()

	// Append 3 entries
	payloads := []interface{}{"entry1", map[string]string{"key": "value"}, 42}

	for _, p := range payloads {
		if err := Append(p); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	// Verify intact
	intact, errs := Verify()
	if !intact {
		t.Fatalf("Verify failed before tamper: %v", errs)
	}

	// Tamper: modify the file to change a payload_hash
	file, err := os.OpenFile(auditFilePath, os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	// Read lines, modify second line's payload_hash
	scanner := bufio.NewScanner(file)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) < 2 {
		t.Fatal("not enough lines")
	}

	// Modify second line
	var entry Entry
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatal(err)
	}
	entry.PayloadHash = "tampered"
	modified, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	lines[1] = string(modified)

	// Write back
	file.Seek(0, 0)
	file.Truncate(0)
	for _, line := range lines {
		file.WriteString(line + "\n")
	}
	file.Close()

	// Verify should fail
	intact, errs = Verify()
	if intact {
		t.Fatal("Verify should have failed after tamper")
	}
	if len(errs) == 0 {
		t.Fatal("Expected errors")
	}
	found := false
	for _, err := range errs {
		if err.Error() == "verification error for entry "+entry.UUID+": signature verification failed" || err.Error() == "prev_hash mismatch for entry "+entry.UUID {
			found = true
		}
	}
	if !found {
		t.Fatalf("Expected specific error, got %v", errs)
	}
}