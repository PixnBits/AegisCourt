package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PixnBits/AegisCourt/pkg/config"
	"github.com/PixnBits/AegisCourt/pkg/court"
	"github.com/PixnBits/AegisCourt/pkg/kernel"
	"github.com/spf13/cobra"
)

var (
	verbose bool
	jsonOut bool
	dryRun  bool
	profile string
	confirm bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "aegiscourt",
		Short: "AegisCourt – Constitutional Self-Evolving Agent Framework",
		Long:  `AegisCourt provides a paranoid, constitutional framework for self-evolving agents.`,
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "dry run mode")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "user profile")
	rootCmd.PersistentFlags().BoolVarP(&confirm, "confirm", "y", false, "auto-confirm")

	// Subcommands
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(stopCmd())
	rootCmd.AddCommand(agentCmd())
	rootCmd.AddCommand(haltCmd())
	rootCmd.AddCommand(proposeCmd())
	rootCmd.AddCommand(courtCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(logCmd())
	rootCmd.AddCommand(snapshotCmd())
	rootCmd.AddCommand(rollbackCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize AegisCourt",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit()
		},
	}
}

func configCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Configuration commands",
	}
}

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the AegisCourt kernel",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart()
		},
	}
}

func runStart() error {
	path := config.DefaultConfigPath()
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	kernel, err := kernel.NewKernel(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kernel: %w", err)
	}

	if err := kernel.Start(); err != nil {
		return fmt.Errorf("failed to start kernel: %w", err)
	}

	// For now, just return after starting
	return nil
}

func runInit() error {
	fmt.Println("Welcome to AegisCourt – paranoid mode always on.")

	// LLM selector
	var llm string
	fmt.Print("Select LLM provider (ollama/openai): ")
	fmt.Scanln(&llm)
	if llm == "" {
		llm = "ollama"
	}

	var endpoint, model, apiKey string
	if llm == "ollama" {
		endpoint = "http://127.0.0.1:11434"
		model = "nemotron-3-nano"
		fmt.Printf("Suggested model: %s\n", model)
	} else if llm == "openai" {
		fmt.Print("Enter OpenAI API key: ")
		fmt.Scanln(&apiKey)
		endpoint = "https://api.openai.com/v1"
		model = "gpt-4"
	}

	// About Me
	var persona string
	fmt.Print("Select persona (Alex/Jordan/Sam/Lena): ")
	fmt.Scanln(&persona)
	if persona == "" {
		persona = "Alex Rivera"
	}

	var riskLevel string
	fmt.Print("Risk tolerance (low/medium/high): ")
	fmt.Scanln(&riskLevel)
	if riskLevel == "" {
		riskLevel = "medium"
	}

	var courtMode string
	fmt.Print("Court mode (auto/assisted/hybrid/manual): ")
	fmt.Scanln(&courtMode)
	if courtMode == "" {
		courtMode = "assisted"
	}

	// Save config
	profile := &config.Profile{
		CourtMode:       courtMode,
		RiskTolerance:   0.5,   // placeholder
		DeferralTimeout: "30s", // placeholder
		PreferredLLM:    llm,
		LLMEndpoint:     endpoint,
		APIKeyEncrypted: config.EncryptAPIKey(apiKey),
		ReviewerWeights: map[string]float64{
			"ciso":           1.0,
			"compliance":     1.0,
			"helpfulness":    1.0,
			"mrm":            1.0,
			"responsible-ai": 1.0,
			"sre":            1.0,
		},
	}
	path := config.DefaultConfigPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := config.Save(path, profile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Println("Configuration saved.")
	return nil
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the AegisCourt kernel",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Stopping AegisCourt...")
		},
	}
}

func agentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent commands",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "run <task>",
		Short: "Run a one-shot agent task",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgent(args)
		},
	})
	return cmd
}

func haltCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "halt",
		Short: "Emergency halt",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Emergency halt")
		},
	}
}

func proposeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "propose <type> <name>",
		Short: "Propose a change",
		Example: `aegiscourt propose add-tool web_search
aegiscourt propose amend-rule 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPropose(args)
		},
	}
}

func runPropose(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("type and name required")
	}
	propType := args[0]
	name := args[1]

	// Load config
	path := config.DefaultConfigPath()
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create and start kernel
	k, err := kernel.NewKernel(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kernel: %w", err)
	}
	if err := k.Start(); err != nil {
		return fmt.Errorf("failed to start kernel: %w", err)
	}

	// Create proposal
	// Stub: create a simple proposal
	proposal := court.Proposal{
		ID:          "prop-" + name,
		Type:        propType,
		Name:        name,
		Description: "Proposed " + propType + " " + name,
		Diff:        nil, // no diff for now
		CreatedAt:   time.Now(),
		Status:      "pending",
	}

	// Handle proposal
	if err := k.HandleProposal(proposal); err != nil {
		return fmt.Errorf("failed to handle proposal: %w", err)
	}

	fmt.Printf("Proposal %s submitted\n", proposal.ID)
	return nil
}

func runAgent(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task required")
	}
	task := args[0]

	// Load config
	path := config.DefaultConfigPath()
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create and start kernel
	k, err := kernel.NewKernel(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kernel: %w", err)
	}
	if err := k.Start(); err != nil {
		return fmt.Errorf("failed to start kernel: %w", err)
	}

	// Run agent task
	result := k.AgentRuntime.RunLoop(task)
	fmt.Println("Agent result:", result)
	return nil
}

func courtCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "court",
		Short: "Court commands",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List proposals",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Court list")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "view <id>",
		Short: "View proposal",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Court view:", args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "vote <id> <decision>",
		Short: "Vote on proposal",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Court vote:", args[0], args[1])
		},
	})
	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show system status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}
}

func runStatus() error {
	path := config.DefaultConfigPath()
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	fmt.Printf("Configuration loaded from %s\n", path)
	fmt.Printf("Court Mode: %s\n", cfg.CourtMode)
	fmt.Printf("Preferred LLM: %s\n", cfg.PreferredLLM)
	fmt.Printf("LLM Endpoint: %s\n", cfg.LLMEndpoint)
	return nil
}

func logCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Log commands",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List logs",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Log list")
		},
	})
	return cmd
}

func snapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Snapshot commands",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			enterprise, _ := cmd.Flags().GetBool("enterprise")
			fmt.Printf("Snapshot created (enterprise: %t)\n", enterprise)
			return nil
		},
	})
	cmd.Flags().Bool("enterprise", false, "Create enterprise snapshot")
	return cmd
}

func rollbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback <id>",
		Short: "Rollback to snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("id required")
			}
			fmt.Println("Rolling back to:", args[0])
			return nil
		},
	}
}

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update commands",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Update")
		},
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("AegisCourt v0.1")
			fmt.Println("Build: dev")
		},
	}
}
