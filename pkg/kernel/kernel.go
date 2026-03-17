package kernel

import (
	"crypto/ed25519"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/PixnBits/AegisCourt/pkg/config"
	jsonpatch "github.com/evanphx/json-patch"
)

//go:embed ../../constitution/initial_rules_v0.1.md
var constitution string

type Kernel struct {
	Config *config.Profile
	PubKey ed25519.PublicKey
}

func NewKernel(cfg *config.Profile) (*Kernel, error) {
	return &Kernel{
		Config: cfg,
	}, nil
}

func (k *Kernel) Bootstrap() error {
	// Generate Ed25519 key pair
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return err
	}
	k.PubKey = pub

	// Compute SHA-256 of binary
	binPath := os.Args[0]
	binData, err := os.ReadFile(binPath)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(binData)

	// Sign the hash
	signature := ed25519.Sign(priv, hash[:])

	// Store signature and pubkey hash (placeholder: print for now)
	pubHash := sha256.Sum256(pub)
	fmt.Printf("Kernel bootstrapped. PubKey hash: %x, Signature: %x\n", pubHash, signature)

	return nil
}

func (k *Kernel) VerifySelf() error {
	// On startup, verify
	binPath := os.Args[0]
	binData, err := os.ReadFile(binPath)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(binData)

	// Placeholder: assume signature is stored, but for now, skip verification
	// In real, load stored signature and pubkey, verify
	if !ed25519.Verify(k.PubKey, hash[:], []byte("placeholder")) {
		panic("Self-signature verification failed")
	}
	return nil
}

func (k *Kernel) ApplyMutation(diff jsonpatch.Patch) error {
	// Placeholder: apply to config
	configData, err := toml.Marshal(k.Config)
	if err != nil {
		return err
	}
	patched, err := diff.Apply(configData)
	if err != nil {
		return err
	}
	return toml.Unmarshal(patched, k.Config)
}
