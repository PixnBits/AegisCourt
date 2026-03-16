package crypto

import (
	"testing"

	"golang.org/x/crypto/ed25519"
)

func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("Public key size mismatch: got %d, want %d", len(pub), ed25519.PublicKeySize)
	}
	if len(priv) != ed25519.PrivateKeySize {
		t.Errorf("Private key size mismatch: got %d, want %d", len(priv), ed25519.PrivateKeySize)
	}
}

func TestSignAndVerify(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	data := []byte("test data")
	sig, err := Sign(data, priv)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if !Verify(data, sig, pub) {
		t.Error("Verify failed for correct signature")
	}
	if Verify([]byte("wrong data"), sig, pub) {
		t.Error("Verify passed for wrong data")
	}
	wrongPub, _, _ := GenerateKeyPair()
	if Verify(data, sig, wrongPub) {
		t.Error("Verify passed for wrong public key")
	}
}

func TestHash(t *testing.T) {
	data := []byte("hello")
	hash := Hash(data)
	if len(hash) != 32 {
		t.Errorf("Hash length mismatch: got %d, want 32", len(hash))
	}
	// SHA-256 of "hello" should be known
	expected := []byte{0x2c, 0xf2, 0x4d, 0xba, 0x5f, 0xb0, 0xa3, 0x0e, 0x26, 0xe8, 0x3b, 0x2a, 0xc5, 0xb9, 0xe2, 0x9e, 0x1b, 0x16, 0x1e, 0x5c, 0x1f, 0xa7, 0x42, 0x5e, 0x73, 0x04, 0x33, 0x62, 0x93, 0x8b, 0x98, 0x24}
	if string(hash) != string(expected) {
		t.Error("Hash mismatch")
	}
}

func TestMerkleTree(t *testing.T) {
	mt := NewMerkleTree()
	// Empty tree
	root := mt.GetRoot()
	if len(root) != 32 {
		t.Errorf("Empty root length: got %d, want 32", len(root))
	}
	// Append one
	mt.Append([]byte("a"))
	root = mt.GetRoot()
	proof, err := mt.GetProof(0)
	if err != nil {
		t.Fatalf("GetProof failed: %v", err)
	}
	if !VerifyProof(mt.leaves[0], proof, root) {
		t.Error("Proof verification failed for single leaf")
	}
	// Append second
	mt.Append([]byte("b"))
	root = mt.GetRoot()
	proof, err = mt.GetProof(1)
	if err != nil {
		t.Fatalf("GetProof failed: %v", err)
	}
	if !VerifyProof(mt.leaves[1], proof, root) {
		t.Error("Proof verification failed for second leaf")
	}
	// Check first still valid
	proof, err = mt.GetProof(0)
	if err != nil {
		t.Fatalf("GetProof failed: %v", err)
	}
	if !VerifyProof(mt.leaves[0], proof, root) {
		t.Error("Proof verification failed for first leaf after append")
	}
}
