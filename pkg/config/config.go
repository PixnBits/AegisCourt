package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pixnbits/aegiscourt/pkg/keys"
)

type CourtMode string

const (
	ModeAuto     CourtMode = "auto"
	ModeAssisted CourtMode = "assisted"
	ModeHybrid   CourtMode = "hybrid"
	ModeManual   CourtMode = "manual"
)

type Config struct {
	mu              sync.RWMutex
	PreferredLLM    string    `json:"preferred_llm"`
	LLMEndpoint     string    `json:"llm_endpoint"`
	CourtMode       CourtMode `json:"court_mode"`
	LowResourceMode bool      `json:"low_resource_mode"`
	RiskTolerance   int       `json:"risk_tolerance"`
	DeferTimeout    string    `json:"defer_timeout"`
	UseCases        string    `json:"use_cases"`
	ProfileTemplate string    `json:"profile_template"`
	Initialized     bool      `json:"initialized"`
}

func Default() *Config {
	return &Config{
		PreferredLLM:    "nemotron-3-nano:latest",
		LLMEndpoint:     "http://localhost:11434",
		CourtMode:       ModeAuto,
		LowResourceMode: false,
		RiskTolerance:   5,
		DeferTimeout:    "5m",
		ProfileTemplate: "hobbyist",
		Initialized:     false,
	}
}

func configPath() (string, error) {
	dir, err := keys.AegisCourtDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("corrupt config file: %w", err)
	}
	return cfg, nil
}

func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := keys.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

func (c *Config) Get(key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	switch key {
	case "preferred_llm":
		return c.PreferredLLM, nil
	case "llm_endpoint":
		return c.LLMEndpoint, nil
	case "court.mode":
		return string(c.CourtMode), nil
	case "low_resource_mode":
		if c.LowResourceMode {
			return "true", nil
		}
		return "false", nil
	case "risk_tolerance":
		return fmt.Sprintf("%d", c.RiskTolerance), nil
	case "defer_timeout":
		return c.DeferTimeout, nil
	case "profile_template":
		return c.ProfileTemplate, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func (c *Config) Set(key, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch key {
	case "preferred_llm":
		c.PreferredLLM = value
	case "llm_endpoint":
		c.LLMEndpoint = value
	case "court.mode":
		mode := CourtMode(value)
		switch mode {
		case ModeAuto, ModeAssisted, ModeHybrid, ModeManual:
			c.CourtMode = mode
		default:
			return fmt.Errorf("invalid court mode: %s (must be auto|assisted|hybrid|manual)", value)
		}
	case "low_resource_mode":
		c.LowResourceMode = value == "true"
	case "risk_tolerance":
		var rt int
		if _, err := fmt.Sscanf(value, "%d", &rt); err != nil || rt < 0 || rt > 10 {
			return fmt.Errorf("risk_tolerance must be 0-10")
		}
		c.RiskTolerance = rt
	case "defer_timeout":
		c.DeferTimeout = value
	default:
		return fmt.Errorf("unknown or read-only config key: %s", key)
	}
	return nil
}

func (c *Config) ListKeys() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return map[string]string{
		"preferred_llm":     c.PreferredLLM,
		"llm_endpoint":      c.LLMEndpoint,
		"court.mode":        string(c.CourtMode),
		"low_resource_mode": fmt.Sprintf("%t", c.LowResourceMode),
		"risk_tolerance":    fmt.Sprintf("%d", c.RiskTolerance),
		"defer_timeout":     c.DeferTimeout,
		"profile_template":  c.ProfileTemplate,
	}
}
