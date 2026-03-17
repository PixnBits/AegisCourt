package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)
Long:  `AegisCourt provides a paranoid, constitutional framework for self-evolving agents.`,
}

// Global flags
var verbose bool
var jsonOut bool
var dryRun bool
var profile string
var confirm bool
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

func runInit() error {
	fmt.Println("Welcome to AegisCourt – paranoid mode always on.")

	// LLM selector
	var llm string
	llmPrompt := &survey.Select{
		Message: "Select LLM provider:",
		Options: []string{"ollama", "openai"},
		Default: "ollama",
	}
	survey.AskOne(llmPrompt, &llm)

	var endpoint, model, apiKey string
	if llm == "ollama" {
		endpoint = "http://127.0.0.1:11434"
		model = "nemotron-3-nano"
		fmt.Printf("Suggested model: %s\n", model)
	} else if llm == "openai" {
		apiKeyPrompt := &survey.Password{
			Message: "Enter OpenAI API key:",
		}
		survey.AskOne(apiKeyPrompt, &apiKey)
		endpoint = "https://api.openai.com/v1"
		model = "gpt-4"
	}

	// About Me
	var persona string
	personaPrompt := &survey.Select{
		Message: "Select your persona:",
		Options: []string{"Alex Rivera", "Jordan", "Sam", "Lena"},
		Default: "Alex Rivera",
	}
	survey.AskOne(personaPrompt, &persona)

	var riskLevel string
	riskPrompt := &survey.Select{
		Message: "Risk tolerance:",
		Options: []string{"low", "medium", "high"},
		Default: "medium",
	}
	survey.AskOne(riskPrompt, &riskLevel)

	var courtMode string
	modePrompt := &survey.Select{
		Message: "Court mode:",
		Options: []string{"auto", "assisted", "hybrid", "manual"},
		Default: "assisted",
	}
	survey.AskOne(modePrompt, &courtMode)

	// Save config
	config := map[string]string{
		"llm_provider": llm,
		"llm_endpoint": endpoint,
		"llm_model":    model,
		"api_key":      apiKey,
		"persona":      persona,
		"risk_level":   riskLevel,
		"court_mode":   courtMode,
	}
	fmt.Println("Configuration saved.")
	fmt.Printf("%+v\n", config)

	// Kernel bootstrap stub
	fmt.Println("Kernel bootstrapped.")

	// Demo proposal
	fmt.Println("Running demo proposal...")

	return nil
Use:   "config",
Short: "Configuration commands",
}
cmd.AddCommand(&cobra.Command{
Use:   "get [key]",
Short: "Get config value",
Run: func(cmd *cobra.Command, args []string) {
fmt.Println("Config get")
},
})
cmd.AddCommand(&cobra.Command{
Use:   "set <key> <value>",
Short: "Set config value",
Run: func(cmd *cobra.Command, args []string) {
fmt.Println("Config set")
},
})
cmd.AddCommand(&cobra.Command{
Use:   "list",
Short: "List config",
Run: func(cmd *cobra.Command, args []string) {
fmt.Println("Config list")
},
})
return cmd
}

func startCmd() *cobra.Command {
return &cobra.Command{
Use:   "start",
Short: "Start the AegisCourt kernel",
Run: func(cmd *cobra.Command, args []string) {
fmt.Println("Starting AegisCourt...")
},
}
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
Run: func(cmd *cobra.Command, args []string) {
fmt.Println("Running agent task:", args[0])
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
Run: func(cmd *cobra.Command, args []string) {
fmt.Println("Proposing:", args[0], args[1])
},
}
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
Short: "Show status",
Run: func(cmd *cobra.Command, args []string) {
fmt.Println("Status: OK")
},
}
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
Run: func(cmd *cobra.Command, args []string) {
fmt.Println("Snapshot created")
},
})
return cmd
}

func rollbackCmd() *cobra.Command {
return &cobra.Command{
Use:   "rollback <id>",
Short: "Rollback to snapshot",
Run: func(cmd *cobra.Command, args []string) {
fmt.Println("Rolling back to:", args[0])
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
},
}
}
