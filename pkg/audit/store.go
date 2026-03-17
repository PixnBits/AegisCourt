package audit

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cbergoon/merkletree"
)

type AuditEntry struct {
	ID        string
	Timestamp time.Time
	PrevHash  []byte
	Payload   []byte
	Signature []byte
	Proof     merkletree.Hash
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

func (s *Store) CreateSnapshot(enterprise bool) (string, error) {
	// Create tar.gz with config, constitution, audit, sbom, nist
	dir := "snapshots"
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, fmt.Sprintf("aegis-%d.tar.gz", time.Now().Unix()))
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Add files
	files := []string{"go.mod", "constitution/initial_rules_v0.1.md"}
	for _, f := range files {
		if err := addFileToTar(tw, f); err != nil {
			return "", err
		}
	}

	if enterprise {
		// Add SBOM stub, NIST stub
		nistContent := "# NIST Mapping\n- Govern: Constitution\n- etc."
		addContentToTar(tw, "nist-mapping.md", nistContent)
	}

	return path, nil
}

func addFileToTar(tw *tar.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name: filename,
		Size: stat.Size(),
		Mode: int64(stat.Mode()),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

func addContentToTar(tw *tar.Writer, name, content string) error {
	header := &tar.Header{
		Name: name,
		Size: int64(len(content)),
		Mode: 0644,
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err := tw.Write([]byte(content))
	return err
}






















































































}	return nil	// Stub: write to filefunc (s *Store) ExportJSONL(path string) error {}	return nil	}		return fmt.Errorf("audit chain invalid")	if !valid || err != nil {	valid, err := s.tree.VerifyTree()func (s *Store) Verify() error {}	return hash	hash, _ := s.entries[len(s.entries)-1].CalculateHash()	}		return []byte{}	if len(s.entries) == 0 {func (s *Store) getLastHash() []byte {}	return nil	s.tree = tree	}		return err	if err != nil {	tree, err := merkletree.NewTree(contents)	}		contents = append(contents, e)	for _, e := range s.entries {	var contents []merkletree.Content	// Rebuild tree	s.entries = append(s.entries, entry)	entry.PrevHash = s.getLastHash()func (s *Store) Append(entry AuditEntry) error {}	}		entries: []AuditEntry{},	return &Store{func NewStore() *Store {}	tree    *merkletree.MerkleTree	entries []AuditEntrytype Store struct {}	return string(thisHash) == string(otherHash), nil	}		return false, err	if err != nil {	thisHash, err := ae.CalculateHash()	}		return false, err	if err != nil {	otherHash, err := other.CalculateHash()func (ae AuditEntry) Equals(other merkletree.Content) (bool, error) {}	return hash[:], nil	hash := sha256.Sum256(data)	}		return nil, err	if err != nil {	data, err := json.Marshal(ae)func (ae AuditEntry) CalculateHash() ([]byte, error) {}	Proof     merkletree.Hash	Signature []byte	Payload   []byte	PrevHash  []byte	Timestamp time.Time	ID        stringtype AuditEntry struct {)	"github.com/cbergoon/merkletree"	"time"	"fmt"	"encoding/json"	"crypto/sha256"import (