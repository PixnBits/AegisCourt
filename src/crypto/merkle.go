package crypto

import (
	"errors"
)

// ProofItem represents an item in the Merkle proof.
type ProofItem struct {
	Hash     []byte
	Position string // "left" or "right"
}

// MerkleTree represents an append-only Merkle tree.
type MerkleTree struct {
	leaves [][]byte
	tree   [][][]byte // levels of the tree
}

// NewMerkleTree creates a new empty Merkle tree.
func NewMerkleTree() *MerkleTree {
	return &MerkleTree{
		leaves: make([][]byte, 0),
		tree:   make([][][]byte, 0),
	}
}

// Append adds a new leaf to the tree and updates the root.
func (mt *MerkleTree) Append(data []byte) {
	hash := Hash(data)
	mt.leaves = append(mt.leaves, hash)
	mt.buildTree()
}

// GetRoot returns the current root hash of the tree.
func (mt *MerkleTree) GetRoot() []byte {
	if len(mt.tree) == 0 {
		return Hash([]byte{})
	}
	return mt.tree[len(mt.tree)-1][0]
}

// GetProof returns the Merkle proof for the leaf at the given index.
// The proof is a slice of ProofItems needed to compute the root.
func (mt *MerkleTree) GetProof(index int) ([]ProofItem, error) {
	if index < 0 || index >= len(mt.leaves) {
		return nil, errors.New("invalid index")
	}
	proof := make([]ProofItem, 0)
	currentIndex := index
	for level := 0; level < len(mt.tree)-1; level++ {
		siblingIndex := currentIndex ^ 1
		if siblingIndex < len(mt.tree[level]) {
			position := "left"
			if currentIndex%2 == 1 {
				position = "right"
			}
			proof = append(proof, ProofItem{Hash: mt.tree[level][siblingIndex], Position: position})
		}
		currentIndex /= 2
	}
	return proof, nil
}

// VerifyProof verifies if the given leaf hash, with the proof, leads to the root.
func VerifyProof(leafHash []byte, proof []ProofItem, root []byte) bool {
	current := leafHash
	for _, item := range proof {
		var combined []byte
		if item.Position == "left" {
			combined = append(current, item.Hash...)
		} else {
			combined = append(item.Hash, current...)
		}
		current = Hash(combined)
	}
	return string(current) == string(root)
}

// buildTree rebuilds the entire tree from leaves.
func (mt *MerkleTree) buildTree() {
	if len(mt.leaves) == 0 {
		mt.tree = [][][]byte{}
		return
	}
	mt.tree = [][][]byte{mt.leaves}
	currentLevel := mt.leaves
	for len(currentLevel) > 1 {
		nextLevel := make([][]byte, 0, (len(currentLevel)+1)/2)
		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				combined := append(currentLevel[i], currentLevel[i+1]...)
				nextLevel = append(nextLevel, Hash(combined))
			} else {
				// Odd number, duplicate the last
				combined := append(currentLevel[i], currentLevel[i]...)
				nextLevel = append(nextLevel, Hash(combined))
			}
		}
		mt.tree = append(mt.tree, nextLevel)
		currentLevel = nextLevel
	}
}
