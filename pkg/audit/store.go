package audit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cbergoon/merkletree"
)

type AuditEntry struct {
	ID        string
	Timestamp time.Time
	PrevHash  []byte
	Payload   []byte
	Signature []byte
	Proof     []byte
}

func (ae AuditEntry) CalculateHash() ([]byte, error) {
	data, err := json.Marshal(ae)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)
	return hash[:], nil
}

func (ae AuditEntry) Equals(other merkletree.Content) (bool, error) {
	otherHash, err := other.CalculateHash()
	if err != nil {
		return false, err
	}
	thisHash, err := ae.CalculateHash()
	if err != nil {
		return false, err
	}
	return string(thisHash) == string(otherHash), nil
}

type Store struct {
	entries []AuditEntry
	tree    *merkletree.MerkleTree
}

func NewStore() *Store {
	return &Store{
		entries: []AuditEntry{},
	}
}

func (s *Store) Append(entry AuditEntry) error {
	entry.PrevHash = s.getLastHash()
	s.entries = append(s.entries, entry)

	// Rebuild tree
	var contents []merkletree.Content
	for _, e := range s.entries {
		contents = append(contents, e)
	}
	tree, err := merkletree.NewTree(contents)
	if err != nil {
		return err
	}
	s.tree = tree
	return nil
}

func (s *Store) getLastHash() []byte {
	if len(s.entries) == 0 {
		return []byte{}
	}
	hash, _ := s.entries[len(s.entries)-1].CalculateHash()
	return hash
}

func (s *Store) Verify() error {
	valid, err := s.tree.VerifyTree()
	if !valid || err != nil {
		return fmt.Errorf("audit chain invalid")
	}
	return nil
}

func (s *Store) ExportJSONL(path string) error {
	// Stub: write to file
	return nil
}
