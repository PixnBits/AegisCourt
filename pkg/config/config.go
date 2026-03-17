package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Profile struct {
	CourtMode       string             `toml:"court_mode"`
	RiskTolerance   float64            `toml:"risk_tolerance"`
	DeferralTimeout string             `toml:"deferral_timeout"`
	PreferredLLM    string             `toml:"preferred_llm"`
	LLMEndpoint     string             `toml:"llm_endpoint"`
	APIKeyEncrypted string             `toml:"api_key_encrypted"`
	ReviewerWeights map[string]float64 `toml:"reviewer_weights"`
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aegiscourt", "config.toml")
}

func Load(path string) (*Profile, error) {
	var p Profile
	_, err := toml.DecodeFile(path, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func Save(path string, p *Profile) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(p)
}

// Placeholder for encryption
func EncryptAPIKey(key string) string {
	return key // TODO: implement encryption
}

func DecryptAPIKey(encrypted string) string {
	return encrypted // TODO: implement decryption
}
