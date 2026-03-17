package audit

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pixnbits/aegiscourt/pkg/keys"
)

type Entry struct {
	ID          string `json:"id"`
	Timestamp   string `json:"timestamp"`
	PrevHash    string `json:"prev_hash"`
	Payload     string `json:"payload"`
	PayloadHash string `json:"payload_hash"`
	Signature   string `json:"signature"`
	EntryHash   string `json:"entry_hash"`
}

type Log struct {
	mu       sync.Mutex
	filePath string
	lastHash string
	privKey  ed25519.PrivateKey
	pubKey   ed25519.PublicKey
}

func NewLog(privKey ed25519.PrivateKey, pubKey ed25519.PublicKey) (*Log, error) {
	dir, err := keys.AegisCourtDir()
	if err != nil {
		return nil, err
	}
	if err := keys.EnsureDir(dir); err != nil {
		return nil, err
	}
	logPath := filepath.Join(dir, "audit.jsonl")
	l := &Log{
		filePath: logPath,
		lastHash: "0000000000000000000000000000000000000000000000000000000000000000",
		privKey:  privKey,
		pubKey:   pubKey,
	}
	if err := l.loadLastHash(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *Log) loadLastHash() error {
	data, err := os.ReadFile(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := splitLines(data)
	if len(lines) == 0 {
		return nil
	}
	var lastEntry Entry
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &lastEntry); err != nil {
		return fmt.Errorf("corrupt last audit entry: %w", err)
	}
	l.lastHash = lastEntry.EntryHash
	return nil
}

func splitLines(data []byte) []string {
	var lines []string
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			line := string(data[start:i])
			if len(line) > 0 {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(data) {
		line := string(data[start:])
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}
	return lines
}

func generateUUIDv7() string {
	now := time.Now()
	ms := now.UnixMilli()
	var uuid [16]byte
	uuid[0] = byte(ms >> 40)
	uuid[1] = byte(ms >> 32)
	uuid[2] = byte(ms >> 24)
	uuid[3] = byte(ms >> 16)
	uuid[4] = byte(ms >> 8)
	uuid[5] = byte(ms)

	randBytes := make([]byte, 10)
	rand.Read(randBytes)
	copy(uuid[6:], randBytes)

	uuid[6] = 0x70 | (uuid[6] & 0x0f)
	uuid[8] = 0x80 | (uuid[8] & 0x3f)

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(uuid[0])<<24|uint32(uuid[1])<<16|uint32(uuid[2])<<8|uint32(uuid[3]),
		uint16(uuid[4])<<8|uint16(uuid[5]),
		uint16(uuid[6])<<8|uint16(uuid[7]),
		uint16(uuid[8])<<8|uint16(uuid[9]),
		uint64(uuid[10])<<40|uint64(uuid[11])<<32|uint64(uuid[12])<<24|uint64(uuid[13])<<16|uint64(uuid[14])<<8|uint64(uuid[15]),
	)
}

func (l *Log) Append(payload string) (*Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	id := generateUUIDv7()
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	payloadHash := hashString(payload)

	preimage := id + ts + l.lastHash + payloadHash
	sig := ed25519.Sign(l.privKey, []byte(preimage))
	sigHex := hex.EncodeToString(sig)

	entryHash := hashString(preimage + sigHex)

	entry := Entry{
		ID:          id,
		Timestamp:   ts,
		PrevHash:    l.lastHash,
		Payload:     payload,
		PayloadHash: payloadHash,
		Signature:   sigHex,
		EntryHash:   entryHash,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write audit entry: %w", err)
	}

	l.lastHash = entryHash
	return &entry, nil
}

func (l *Log) Verify() (int, []string, error) {
	data, err := os.ReadFile(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil, nil
		}
		return 0, nil, fmt.Errorf("failed to read audit log: %w", err)
	}

	lines := splitLines(data)
	if len(lines) == 0 {
		return 0, nil, nil
	}

	var errors []string
	prevHash := "0000000000000000000000000000000000000000000000000000000000000000"

	for i, line := range lines {
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			errors = append(errors, fmt.Sprintf("entry %d: corrupt JSON: %v", i, err))
			continue
		}
		if entry.PrevHash != prevHash {
			errors = append(errors, fmt.Sprintf("entry %d (%s): chain broken — expected prev_hash %s, got %s", i, entry.ID, prevHash, entry.PrevHash))
		}
		expectedPayloadHash := hashString(entry.Payload)
		if entry.PayloadHash != expectedPayloadHash {
			errors = append(errors, fmt.Sprintf("entry %d (%s): payload tampered", i, entry.ID))
		}
		preimage := entry.ID + entry.Timestamp + entry.PrevHash + entry.PayloadHash
		sig, err := hex.DecodeString(entry.Signature)
		if err != nil {
			errors = append(errors, fmt.Sprintf("entry %d (%s): corrupt signature hex", i, entry.ID))
		} else if !ed25519.Verify(l.pubKey, []byte(preimage), sig) {
			errors = append(errors, fmt.Sprintf("entry %d (%s): invalid signature", i, entry.ID))
		}
		expectedEntryHash := hashString(preimage + entry.Signature)
		if entry.EntryHash != expectedEntryHash {
			errors = append(errors, fmt.Sprintf("entry %d (%s): entry hash mismatch", i, entry.ID))
		}
		prevHash = entry.EntryHash
	}

	return len(lines), errors, nil
}

func (l *Log) List(filter string) ([]Entry, error) {
	data, err := os.ReadFile(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read audit log: %w", err)
	}
	lines := splitLines(data)
	var entries []Entry
	for _, line := range lines {
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if filter == "" || containsStr(entry.Payload, filter) || containsStr(entry.ID, filter) {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (l *Log) Export(path string) error {
	data, err := os.ReadFile(l.filePath)
	if err != nil {
		return fmt.Errorf("failed to read audit log: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}
