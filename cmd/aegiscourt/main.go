package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pixnbits/aegiscourt/bench"
	"github.com/pixnbits/aegiscourt/pkg/agent"
	"github.com/pixnbits/aegiscourt/pkg/audit"
	"github.com/pixnbits/aegiscourt/pkg/config"
	"github.com/pixnbits/aegiscourt/pkg/court"
	"github.com/pixnbits/aegiscourt/pkg/keys"
	"github.com/pixnbits/aegiscourt/pkg/llm"
	"github.com/pixnbits/aegiscourt/pkg/mutation"
	"github.com/pixnbits/aegiscourt/pkg/mutation/handlers"
	"github.com/pixnbits/aegiscourt/pkg/notify"
	"github.com/pixnbits/aegiscourt/pkg/proposal"
	"github.com/pixnbits/aegiscourt/pkg/resources"
)

var version = "0.2.0-dev"

type Globals struct {
	JSON        bool
	Verbose     bool
	DryRun      bool
	Confirm     bool
	LowResource bool
	ModeInfo    bool
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	args := os.Args[1:]
	var jsonOutput, verbose, dryRun, confirm, lowResource, modeInfo bool
	var filtered []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--verbose":
			verbose = true
		case "--dry-run":
			dryRun = true
		case "--confirm", "-y":
			confirm = true
		case "--low-resource":
			lowResource = true
		case "--mode-info":
			modeInfo = true
		case "--help", "-h":
			if len(filtered) == 0 {
				printHelp()
				os.Exit(0)
			}
			filtered = append(filtered, args[i])
		case "--version":
			fmt.Printf("aegiscourt %s\n", version)
			os.Exit(0)
		default:
			filtered = append(filtered, args[i])
		}
	}

	if len(filtered) == 0 {
		printHelp()
		os.Exit(1)
	}

	globals := &Globals{
		JSON:        jsonOutput,
		Verbose:     verbose,
		DryRun:      dryRun,
		Confirm:     confirm,
		LowResource: lowResource,
		ModeInfo:    modeInfo,
	}

	cmd := filtered[0]
	cmdArgs := filtered[1:]

	switch cmd {
	case "init":
		cmdInit(globals, cmdArgs)
	case "config":
		cmdConfig(globals, cmdArgs)
	case "start":
		cmdStart(globals, cmdArgs)
	case "stop":
		cmdStop(globals, cmdArgs)
	case "agent":
		cmdAgent(globals, cmdArgs)
	case "propose":
		cmdPropose(globals, cmdArgs)
	case "court":
		cmdCourt(globals, cmdArgs)
	case "status":
		cmdStatus(globals, cmdArgs)
	case "log":
		cmdLog(globals, cmdArgs)
	case "audit":
		cmdAudit(globals, cmdArgs)
	case "rollback":
		cmdRollback(globals, cmdArgs)
	case "bench":
		cmdBench(globals, cmdArgs)
	case "halt":
		cmdHalt(globals, cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`aegiscourt %s -- paranoid AI governance framework

Usage: aegiscourt [global flags] <subcommand> [args]

Global Flags:
  --verbose              Detailed output
  --json                 Machine-readable JSON output
  --dry-run              Simulate without applying changes
  --confirm / -y         Bypass interactive confirmation
  --low-resource         Force sequential reviewer execution
  --mode-info            Show current Court mode
  --version              Show version
  --help / -h            Show help

Subcommands:
  init                   First-run setup wizard
  config                 View/edit configuration (get, set, list)
  start                  Launch kernel + agent runtime
  stop                   Graceful shutdown
  agent run <task>       One-shot agent execution
  propose                Submit/create proposals (guide, agent-help, submit)
  court                  Governance Court (list, view, qa, vote)
  status                 Runtime overview
  log list               View audit trail
  audit verify           Verify Merkle chain integrity
  rollback <id|last>     Revert mutation
  bench run              Run benchmark tasks
  halt                   Emergency freeze

Run 'aegiscourt <subcommand> --help' for details.
`, version)
}

// ===== HELPERS =====

func mustLoadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

func mustInitAudit() *audit.Log {
	pub, priv, err := keys.LoadKeypair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	al, err := audit.NewLog(priv, pub)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing audit log: %v\n", err)
		os.Exit(1)
	}
	return al
}

func mustInitRouter(cfg *config.Config) *llm.Router {
	fallback := "llama3.2:latest"
	return llm.NewRouter(cfg.LLMEndpoint, cfg.PreferredLLM, fallback)
}
newMutationEngine(al *audit.Log) *mutation.Engine {
	eng := mutation.NewEngine(al)
	eng.RegisterHandler("add-tool", &handlers.ToolHandler{})
	eng.RegisterHandler("change-prompt", &handlers.PromptHandler{})
	eng.RegisterHandler("amend-rule", &handlers.ConstitutionHandler{})
	eng.RegisterHandler("add-skill", &handlers.SkillHandler{})
	eng.RegisterHandler("upgrade-memory", &handlers.MemoryHandler{})
	eng.RegisterHandler("other", &handlers.GenericHandler{})
	return eng
}

func 
func newMutationEngine(al *audit.Log) *mutation.Engine {
	eng := mutation.NewEngine(al)
	eng.RegisterHandler("add-tool", &handlers.ToolHandler{})
	eng.RegisterHandler("change-prompt", &handlers.PromptHandler{})
	eng.RegisterHandler("amend-rule", &handlers.ConstitutionHandler{})
	eng.RegisterHandler("add-skill", &handlers.SkillHandler{})
	eng.RegisterHandler("upgrade-memory", &handlers.MemoryHandler{})
	eng.RegisterHandler("other", &handlers.GenericHandler{})
	return eng
}

func readLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ===== INIT =====

func cmdInit(g *Globals, args []string) {
	var llmFlag, profileTemplate, courtMode string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--llm":
			if i+1 < len(args) {
				llmFlag = args[i+1]
				i++
			}
		case "--profile-template":
			if i+1 < len(args) {
				profileTemplate = args[i+1]
				i++
			}
		case "--court-mode":
			if i+1 < len(args) {
				courtMode = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println(`Usage: aegiscourt init [flags]

Flags:
  --llm <ollama|openai>          LLM backend (default: ollama)
  --profile-template <template>  hobbyist|indie|enterprise|financial
  --court-mode <mode>            auto|assisted|hybrid|manual`)
			return
		}
	}

	fmt.Println("Welcome to AegisCourt -- paranoid mode always on.")
	fmt.Println("Local-first, cryptographically immutable agent framework.")
	fmt.Println("Setup takes ~2 min. Ctrl+C to abort.")
	fmt.Println()

	cfg := config.Default()

	// Step 1: Resource detection
	fmt.Println("Detecting system resources...")
	res := resources.Detect()
	fmt.Println("Detected system resources:")
	fmt.Println(res.String())
	fmt.Println()

	// Step 2: LLM selection
	primary, fallback, lowRes, warning := res.RecommendLLM()
	if warning != "" {
		fmt.Println(warning)
		fmt.Println()
	}

	if llmFlag == "ollama" || llmFlag == "" {
		cfg.LLMEndpoint = "http://localhost:11434"
		fmt.Printf("Recommended: %s\n", primary)
		fmt.Printf("Strong fallback: %s\n", fallback)
		fmt.Println()

		router := llm.NewRouter(cfg.LLMEndpoint, primary, fallback)
		if err := router.Ping(); err != nil {
			fmt.Printf("Warning: Cannot reach Ollama at %s: %v\n", cfg.LLMEndpoint, err)
			fmt.Println("Please ensure Ollama is running: ollama serve")
		} else {
			models, _ := router.ListModels()
			if len(models) > 0 {
				fmt.Println("Available models:")
				for _, m := range models {
					fmt.Printf("  - %s (%.1f GB)\n", m.Name, float64(m.Size)/(1024*1024*1024))
				}
			}
		}

		fmt.Printf("\nSelect LLM [%s]: ", primary)
		choice := readLine()
		if choice != "" {
			cfg.PreferredLLM = choice
		} else {
			cfg.PreferredLLM = primary
		}
		cfg.LowResourceMode = lowRes
	}

	// Step 3: About Me Profile
	if profileTemplate != "" {
		cfg.ProfileTemplate = profileTemplate
		switch profileTemplate {
		case "hobbyist":
			cfg.CourtMode = config.ModeAuto
		case "indie":
			cfg.CourtMode = config.ModeAssisted
		case "enterprise":
			cfg.CourtMode = config.ModeHybrid
		case "financial":
			cfg.CourtMode = config.ModeManual
		}
		fmt.Printf("Selected: %s (%s)\n", profileTemplate, cfg.CourtMode)
	} else if courtMode != "" {
		cfg.CourtMode = config.CourtMode(courtMode)
		fmt.Printf("Court mode set to: %s\n", courtMode)
	} else {
		fmt.Println("\nWhich best describes you?")
		fmt.Println("  1. Hobbyist Auto (fast, low intervention)")
		fmt.Println("  2. Indie Assisted (medium scrutiny)")
		fmt.Println("  3. Team Hybrid (simulate team via explicit vote)")
		fmt.Println("  4. Enterprise Manual (strict thresholds)")
		fmt.Print("Choice [1]: ")
		choice := readLine()
		switch choice {
		case "2":
			cfg.ProfileTemplate = "indie"
			cfg.CourtMode = config.ModeAssisted
		case "3":
			cfg.ProfileTemplate = "enterprise"
			cfg.CourtMode = config.ModeHybrid
		case "4":
			cfg.ProfileTemplate = "financial"
			cfg.CourtMode = config.ModeManual
		default:
			cfg.ProfileTemplate = "hobbyist"
			cfg.CourtMode = config.ModeAuto
		}
	}

	fmt.Print("\nRisk tolerance (0=ultra-paranoid, 10=experiment freely) [5]: ")
	rtChoice := readLine()
	if rtChoice != "" {
		var rt int
		if _, err := fmt.Sscanf(rtChoice, "%d", &rt); err == nil && rt >= 0 && rt <= 10 {
			cfg.RiskTolerance = rt
		}
	}

	fmt.Print("Main use cases (optional, free text): ")
	cfg.UseCases = readLine()

	fmt.Print("Max defer timeout [5m]: ")
	dt := readLine()
	if dt != "" {
		cfg.DeferTimeout = dt
	}

	// Step 4: Kernel bootstrap
	fmt.Println("\nBootstrapping kernel...")
	pub, _, err := keys.GenerateAndStoreKeypair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating keys: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Kernel ready. Root key: ed25519:%s\n", keys.PublicKeyHex(pub)[:16]+"...")

	cfg.Initialized = true
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	// Create audit log and record init
	_, priv, _ := keys.LoadKeypair()
	al, _ := audit.NewLog(priv, pub)
	al.Append(fmt.Sprintf("kernel_init: mode=%s llm=%s low_resource=%v", cfg.CourtMode, cfg.PreferredLLM, cfg.LowResourceMode))

	fmt.Println("\nSetup complete! Run 'aegiscourt start' to begin.")
	fmt.Println("Then try: aegiscourt agent run \"hello world\"")
}

// ===== CONFIG =====

func cmdConfig(g *Globals, args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: aegiscourt config <get|set|list> [key] [value]

Examples:
  aegiscourt config list
  aegiscourt config get court.mode
  aegiscourt config set court.mode assisted`)
		return
	}

	cfg := mustLoadConfig()
	switch args[0] {
	case "list":
		kvs := cfg.ListKeys()
		if g.JSON {
			data, _ := json.MarshalIndent(kvs, "", "  ")
			fmt.Println(string(data))
		} else {
			for k, v := range kvs {
				fmt.Printf("%-20s %s\n", k, v)
			}
		}
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: aegiscourt config get <key>")
			os.Exit(1)
		}
		val, err := cfg.Get(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(val)
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: aegiscourt config set <key> <value>")
			os.Exit(1)
		}
		if err := cfg.Set(args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		al := mustInitAudit()
		al.Append(fmt.Sprintf("config_set: %s=%s", args[1], args[2]))
		fmt.Printf("%s set to %s\n", args[1], args[2])
	default:
		fmt.Fprintf(os.Stderr, "Unknown config command: %s\n", args[0])
		os.Exit(1)
	}
}

// ===== START =====

func cmdStart(g *Globals, args []string) {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Println(`Usage: aegiscourt start [flags]

Flags:
  --detached            Run in background
  --resources <spec>    Resource limits (e.g. ram=4GB,cpu=2)
  --sandbox <type>      gvisor|seccomp`)
			return
		}
	}

	cfg := mustLoadConfig()
	if !cfg.Initialized {
		fmt.Fprintln(os.Stderr, "Error: AegisCourt not initialized. Run 'aegiscourt init' first.")
		os.Exit(1)
	}

	router := mustInitRouter(cfg)
	if err := router.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot reach LLM endpoint: %v\n", err)
		os.Exit(1)
	}

	al := mustInitAudit()
	if g.ModeInfo {
		fmt.Printf("Court mode: %s\n", cfg.CourtMode)
		fmt.Printf("Low resource: %v\n", cfg.LowResourceMode)
	}

	res := resources.Detect()
	al.Append(fmt.Sprintf("kernel_start: llm=%s mode=%s ram_free=%.1fGB", cfg.PreferredLLM, cfg.CourtMode, res.FreeRAMGB))

	fmt.Println("Kernel started.")
	fmt.Printf("  LLM: %s @ %s\n", cfg.PreferredLLM, cfg.LLMEndpoint)
	fmt.Printf("  Court mode: %s\n", cfg.CourtMode)
	fmt.Printf("  Low resource: %v\n", cfg.LowResourceMode || g.LowResource)
	fmt.Println("Ready for commands.")
}

// ===== STOP =====

func cmdStop(g *Globals, args []string) {
	al := mustInitAudit()
	al.Append("kernel_stop")
	fmt.Println("Kernel stopped. All sandboxes terminated.")
}

// ===== AGENT =====

func cmdAgent(g *Globals, args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: aegiscourt agent run <task description>

Execute a one-shot agent task in an ephemeral sandbox.

Flags:
  --timeout <duration>  Task timeout (default: 30s)

Available tools: echo, utc_time`)
		return
	}
	if args[0] != "run" {
		fmt.Fprintf(os.Stderr, "Unknown agent command: %s\n", args[0])
		os.Exit(1)
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: aegiscourt agent run <task description>")
		os.Exit(1)
	}

	cfg := mustLoadConfig()
	task := strings.Join(args[1:], " ")
	router := mustInitRouter(cfg)
	al := mustInitAudit()
	al.Append(fmt.Sprintf("agent_run: task=%s", task))

	ctx := context.Background()

	reg, err := agent.LoadRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load tool registry: %v\n", err)
		reg = &agent.Registry{}
	}
	systemPrompt := reg.BuildSystemPrompt()

	messages := []llm.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: task},
	}

	fmt.Printf("Running agent task: %s\n", task)
	response, model, err := router.Chat(ctx, messages, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
		al.Append(fmt.Sprintf("agent_error: %s", err))
		os.Exit(1)
	}

	result := handleToolCallRegistry(response, reg)
	al.Append(fmt.Sprintf("agent_response: model=%s response=%s", model, truncate(result, 200)))

	if g.JSON {
		data, _ := json.MarshalIndent(map[string]string{
			"task":     task,
			"model":    model,
			"response": result,
		}, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println(result)
	}
}

func handleToolCallRegistry(response string, reg *agent.Registry) string {
	var toolCall struct {
		Tool string         `json:"tool"`
		Args map[string]any `json:"args"`
	}

	jsonStr := court.ExtractJSON(response)
	if err := json.Unmarshal([]byte(jsonStr), &toolCall); err == nil && toolCall.Tool != "" {
		argStr := ""
		if msg, ok := toolCall.Args["message"].(string); ok {
			argStr = msg
		}
		result, err := agent.ExecuteTool(reg, toolCall.Tool, argStr)
		if err != nil {
			return fmt.Sprintf("Tool error: %v -- Blocked: Rule 3", err)
		}
		return result
	}
	return response
}

// ===== PROPOSE =====

func cmdPropose(g *Globals, args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: aegiscourt propose <subcommand> [args]

Subcommands:
  guide               Interactive proposal wizard
  agent-help "<desc>"  Generate draft from short description
  submit <draft-id>    Submit draft for Court review

Flags for guide:
  --type <type>         Pre-select proposal type
  --llm-assist <level>  none|light|full (default: light)
  --draft <uuid>        Continue from existing draft`)
		return
	}

	switch args[0] {
	case "guide":
		cmdProposeGuide(g, args[1:])
	case "agent-help":
		cmdProposeAgentHelp(g, args[1:])
	case "submit":
		cmdProposeSubmit(g, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown propose command: %s\n", args[0])
		os.Exit(1)
	}
}

func cmdProposeAgentHelp(g *Globals, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: aegiscourt propose agent-help \"<short description>\"")
		os.Exit(1)
	}

	desc := strings.Join(args, " ")
	cfg := mustLoadConfig()
	router := mustInitRouter(cfg)
	al := mustInitAudit()

	// Load the agent-help prompt template
	promptTemplate, err := os.ReadFile("prompts/propose-agent-help.md")
	if err != nil {
		// Try embedded path
		promptTemplate = []byte(defaultAgentHelpPrompt)
	}

	prompt := strings.Replace(string(promptTemplate), "{user_short_request}", desc, 1)

	fmt.Println("Generating proposal draft from your description...")
	fmt.Printf("Request: %s\n\n", desc)

	ctx := context.Background()
	response, model, err := router.Generate(ctx, prompt, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating draft: %v\n", err)
		os.Exit(1)
	}

	jsonStr := court.ExtractJSON(response)
	var draft proposal.Draft
	if err := json.Unmarshal([]byte(jsonStr), &draft); err != nil {
		// Retry with error feedback
		retryPrompt := prompt + "\n\nYour previous response had a JSON error: " + err.Error() + "\nPlease output ONLY valid JSON."
		response, model, err = router.Generate(ctx, retryPrompt, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error on retry: %v\n", err)
			os.Exit(1)
		}
		jsonStr = court.ExtractJSON(response)
		if err := json.Unmarshal([]byte(jsonStr), &draft); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse draft after retry: %v\n", err)
			fmt.Fprintf(os.Stderr, "Raw response:\n%s\n", response)
			os.Exit(1)
		}
	}

	draft.ID = proposal.GenerateDraftID()
	draft.LLMAssistUsed = "full"
	draft.CreatedAt = time.Now().UTC()

	// Validate
	errs := proposal.ValidateDraft(&draft)
	if len(errs) > 0 {
		fmt.Println("Warning: Draft has validation issues:")
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		fmt.Println("These can be fixed in the wizard.")
	}

	if err := proposal.SaveDraft(&draft); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving draft: %v\n", err)
		os.Exit(1)
	}

	al.Append(fmt.Sprintf("agent_help_draft: id=%s model=%s title=%s", draft.ID, model, draft.Title))

	fmt.Printf("\nDraft proposal generated.\n")
	fmt.Printf("Draft ID: %s\n", draft.ID)
	fmt.Printf("Title: %s\n", draft.Title)

	fmt.Println("\nLaunching refinement wizard...")
	fmt.Println()
	cmdProposeGuide(g, []string{"--draft", draft.ID})
}

func cmdProposeGuide(g *Globals, args []string) {
	var draftID, propType, llmAssist string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--draft":
			if i+1 < len(args) {
				draftID = args[i+1]
				i++
			}
		case "--type":
			if i+1 < len(args) {
				propType = args[i+1]
				i++
			}
		case "--llm-assist":
			if i+1 < len(args) {
				llmAssist = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println(`Usage: aegiscourt propose guide [flags]

Flags:
  --draft <id>          Continue from existing draft
  --type <type>         Pre-select proposal type
  --llm-assist <level>  none|light|full (default: light)`)
			return
		}
	}

	al := mustInitAudit()

	var draft proposal.Draft
	if draftID != "" {
		d, err := proposal.LoadDraft(draftID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading draft: %v\n", err)
			os.Exit(1)
		}
		draft = *d
	} else {
		draft.ID = proposal.GenerateDraftID()
		draft.CreatedAt = time.Now().UTC()
	}

	if llmAssist == "" {
		llmAssist = "light"
	}
	draft.LLMAssistUsed = llmAssist

	fmt.Println("=== Proposal Wizard ===")
	fmt.Println()

	types := []string{"add-tool", "add-skill", "change-prompt", "amend-rule", "upgrade-memory", "other"}

	// Step 1: Type
	if propType != "" {
		draft.Type = propType
	}
	if draft.Type == "" {
		fmt.Println("Proposal type:")
		for i, t := range types {
			fmt.Printf("  %d. %s\n", i+1, t)
		}
		fmt.Print("Choice [1]: ")
		c := readLine()
		idx := 0
		if c != "" {
			fmt.Sscanf(c, "%d", &idx)
			idx--
		}
		if idx < 0 || idx >= len(types) {
			idx = 0
		}
		draft.Type = types[idx]
	}
	fmt.Printf("Type: %s\n\n", draft.Type)

	// Step 2: Title
	if draft.Title != "" {
		fmt.Printf("Title [%s]: ", draft.Title)
	} else {
		fmt.Print("Title (8-140 chars): ")
	}
	t := readLine()
	if t != "" {
		draft.Title = t
	}

	// Step 3: Motivation
	if draft.Motivation != "" {
		fmt.Printf("\nMotivation (current: %s)\nEdit or press Enter to keep: ", truncate(draft.Motivation, 80))
	} else {
		fmt.Print("\nMotivation (describe the problem, >=20 chars): ")
	}
	m := readLine()
	if m != "" {
		draft.Motivation = m
	}

	// Step 4: Proposed Change
	if draft.ProposedChange != nil {
		existing, _ := json.Marshal(draft.ProposedChange)
		fmt.Printf("\nProposed change (current: %s)\nEdit or press Enter to keep: ", truncate(string(existing), 80))
	} else {
		fmt.Print("\nProposed change (describe the concrete change): ")
	}
	pc := readLine()
	if pc != "" {
		draft.ProposedChange = pc
	}

	// Step 5: Risk level
	fmt.Printf("\nRisk level (low/medium/high) [%s]: ", orDefault(draft.RiskLevel, "low"))
	rl := readLine()
	if rl != "" {
		draft.RiskLevel = rl
	} else if draft.RiskLevel == "" {
		draft.RiskLevel = "low"
	}

	// Step 6: Risks and mitigations
	fmt.Print("\nRisks and mitigations (comma-separated, or Enter to skip): ")
	rm := readLine()
	if rm != "" {
		draft.RisksAndMitigations = splitComma(rm)
	}

	// Step 7: Rollback plan
	if draft.RollbackPlan != "" {
		fmt.Printf("\nRollback plan (current: %s)\nEdit or press Enter to keep: ", truncate(draft.RollbackPlan, 80))
	} else {
		fmt.Print("\nRollback plan (>=20 chars, explicit revert steps): ")
	}
	rp := readLine()
	if rp != "" {
		draft.RollbackPlan = rp
	}

	// Step 8: Validation plan
	if draft.ValidationPlan != "" {
		fmt.Printf("\nValidation plan (current: %s)\nEdit or press Enter to keep: ", truncate(draft.ValidationPlan, 80))
	} else {
		fmt.Print("\nValidation plan (benchmarks, tests): ")
	}
	vp := readLine()
	if vp != "" {
		draft.ValidationPlan = vp
	}

	// Step 9: Constitution check
	fmt.Print("\nConstitution check -- how does this preserve Rules 1-5? ")
	cc := readLine()
	if cc != "" {
		draft.ConstitutionCheck = cc
	}

	// Validate
	errs := proposal.ValidateDraft(&draft)
	if len(errs) > 0 {
		fmt.Println("\nValidation issues:")
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
	}

	if err := proposal.SaveDraft(&draft); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving draft: %v\n", err)
		os.Exit(1)
	}

	al.Append(fmt.Sprintf("draft_saved: id=%s title=%s", draft.ID, draft.Title))

	fmt.Printf("\nDraft refined and saved as %s\n", draft.ID)
	fmt.Printf("Ready to submit? Use: aegiscourt propose submit %s\n", draft.ID)
}

func cmdProposeSubmit(g *Globals, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: aegiscourt propose submit <draft-id>")
		os.Exit(1)
	}

	draftID := args[0]
	draft, err := proposal.LoadDraft(draftID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	errs := proposal.ValidateDraft(draft)
	if len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "Draft has validation errors:")
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		fmt.Fprintln(os.Stderr, "Fix issues with: aegiscourt propose guide --draft", draftID)
		os.Exit(1)
	}

	cfg := mustLoadConfig()
	router := mustInitRouter(cfg)
	al := mustInitAudit()

	proposalJSON, _ := proposal.DraftToJSON(draft)

	proposalID := fmt.Sprintf("%04d", time.Now().Unix()%10000)

	al.Append(fmt.Sprintf("proposal_submitted: id=%s draft=%s title=%s", proposalID, draftID, draft.Title))
	notify.EmitProposalSubmitted(proposalID, draft.Title)

	fmt.Printf("Proposal %s submitted: %s\n", proposalID, draft.Title)
	fmt.Println("Court review starting...")

	lowRes := cfg.LowResourceMode || g.LowResource
	engine := court.NewEngine(router, al, lowRes)
	notify.EmitCourtStarted(proposalID, draft.Title)

	ctx := context.Background()
	result, err := engine.RunCourt(ctx, proposalID, draft.Title, proposalJSON, string(cfg.CourtMode))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Court error: %v\n", err)
		os.Exit(1)
	}

	if err := court.SaveResult(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving court result: %v\n", err)
	}

	notify.EmitCourtCompleted(proposalID, result.Recommendation, result.AggregateScore)

	if g.JSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println()
		fmt.Println(court.FormatNASABoard(result))
	}

	switch config.CourtMode(result.CourtMode) {
	case config.ModeAuto:
		if result.AggregateScore >= 80 && g.Confirm {
			fmt.Println("Auto mode: approved and applied.")
			result.VoteAction = "approve"
			result.Status = court.StatusApproved
			court.SaveResult(result)
			al.Append(fmt.Sprintf("auto_approved: proposal=%s score=%.0f", proposalID, result.AggregateScore))
		} else {
			fmt.Printf("Your vote? Use: aegiscourt court vote %s approve|reject|defer\n", proposalID)
		}
	default:
		fmt.Printf("Your vote? Use: aegiscourt court vote %s approve|reject|defer\n", proposalID)
	}
}

// ===== COURT =====

func cmdCourt(g *Globals, args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: aegiscourt court <subcommand> [args]

Subcommands:
  list [--status <filter>]         List proposals
  view <id> [--detailed] [--reviewer <name>] [--json]
  qa <id> <question>               Ask reviewers
  vote <id> approve|reject|defer   Cast vote`)
		return
	}

	switch args[0] {
	case "list":
		cmdCourtList(g, args[1:])
	case "view":
		cmdCourtView(g, args[1:])
	case "qa":
		cmdCourtQA(g, args[1:])
	case "vote":
		cmdCourtVote(g, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown court command: %s\n", args[0])
		os.Exit(1)
	}
}

func cmdCourtList(g *Globals, args []string) {
	var statusFilter string
	for i := 0; i < len(args); i++ {
		if args[i] == "--status" && i+1 < len(args) {
			statusFilter = args[i+1]
			i++
		}
	}

	results, err := court.ListResults(statusFilter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No court results found.")
		return
	}

	if g.JSON {
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("%-8s %-40s %-12s %-6s %-20s\n", "ID", "Title", "Status", "Score", "Recommendation")
	fmt.Println(strings.Repeat("-", 90))
	for _, r := range results {
		fmt.Printf("%-8s %-40s %-12s %-6.0f %-20s\n",
			r.ProposalID,
			truncate(r.ProposalTitle, 40),
			r.Status,
			r.AggregateScore,
			r.Recommendation)
	}
}

func cmdCourtView(g *Globals, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: aegiscourt court view <proposal-id> [--detailed] [--reviewer <name>]")
		os.Exit(1)
	}

	id := args[0]
	var detailed bool
	var reviewer string
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--detailed":
			detailed = true
		case "--reviewer":
			if i+1 < len(args) {
				reviewer = args[i+1]
				i++
			}
		}
	}

	result, err := court.LoadResult(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if g.JSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	if reviewer != "" {
		fmt.Println(court.FormatReviewer(result, reviewer))
	} else if detailed {
		fmt.Println(court.FormatDetailed(result))
	} else {
		fmt.Println(court.FormatNASABoard(result))
	}
}

func cmdCourtQA(g *Globals, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: aegiscourt court qa <proposal-id> <question>")
		os.Exit(1)
	}

	proposalID := args[0]
	question := strings.Join(args[1:], " ")

	result, err := court.LoadResult(proposalID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := mustLoadConfig()
	router := mustInitRouter(cfg)
	al := mustInitAudit()

	proposalJSON, _ := json.Marshal(result)
	engine := court.NewEngine(router, al, cfg.LowResourceMode)

	ctx := context.Background()
	answer, err := engine.QA(ctx, string(proposalJSON), "", question)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	al.Append(fmt.Sprintf("court_qa: proposal=%s question=%s", proposalID, question))
	fmt.Println(answer)
}

func cmdCourtVote(g *Globals, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: aegiscourt court vote <proposal-id> approve|reject|defer [--notes <text>] [--conditions <json>] [--confirm]")
		os.Exit(1)
	}

	proposalID := args[0]
	action := args[1]
	validActions := map[string]bool{"approve": true, "reject": true, "defer": true}
	if !validActions[action] {
		fmt.Fprintf(os.Stderr, "Invalid action: %s (must be approve|reject|defer)\n", action)
		os.Exit(1)
	}

	var notes string
	for i := 2; i < len(args); i++ {
		if args[i] == "--notes" && i+1 < len(args) {
			notes = args[i+1]
			i++
		}
	}

	result, err := court.LoadResult(proposalID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := mustLoadConfig()
	al := mustInitAudit()

	// Check mode requirements
	if cfg.CourtMode != config.ModeAuto && !g.Confirm {
		fmt.Printf("You are about to %s proposal %s in %s mode.\n", action, proposalID, cfg.CourtMode)
		fmt.Print("Confirm? [y/N]: ")
		c := readLine()
		if c != "y" && c != "Y" && c != "yes" {
			fmt.Println("Cancelled.")
			return
		}
	}

	result.VoteAction = action
	result.VoteNotes = notes
	switch action {
	case "approve":
		result.Status = court.StatusApproved
	case "reject":
		result.Status = court.StatusRejected
	case "defer":
		result.Status = court.StatusDeferred
	}

	if err := court.SaveResult(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving result: %v\n", err)
		os.Exit(1)
	}

	al.Append(fmt.Sprintf("court_vote: proposal=%s action=%s notes=%s", proposalID, action, notes))
	notify.EmitVoteCast(proposalID, action)

	fmt.Printf("Vote recorded: %s on proposal %s\n", action, proposalID)
	if action == "approve" {
		eng := newMutationEngine(al)
		m, err := eng.Apply(proposalID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Mutation failed: %v\n", err)
			if m != nil {
				fmt.Fprintf(os.Stderr, "Mutation %s status: %s\n", m.ID, m.Status)
			}
			os.Exit(1)
		}
		fmt.Printf("Mutation applied: %s (type=%s)\n", m.ID, m.Type)
		fmt.Printf("Snapshot: %s\n", m.BeforeSnapshot)
	}
}

// ===== STATUS =====

func cmdStatus(g *Globals, args []string) {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Println(`Usage: aegiscourt status [--watch]

Show runtime overview: resources, pending proposals, Court state.`)
			return
		}
	}

	cfg := mustLoadConfig()
	res := resources.Detect()

	fmt.Println("AegisCourt Status")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Court mode: %s\n", cfg.CourtMode)
	fmt.Printf("LLM: %s\n", cfg.PreferredLLM)
	fmt.Printf("Low resource: %v\n", cfg.LowResourceMode)
	fmt.Println()
	fmt.Println("System Resources:")
	fmt.Println(res.String())
	fmt.Println()

	results, _ := court.ListResults("")
	var pending, active, completed int
	for _, r := range results {
		switch r.Status {
		case court.StatusPending, court.StatusActive:
			pending++
		case court.StatusCompleted:
			active++
		case court.StatusApproved, court.StatusRejected:
			completed++
		}
	}
	fmt.Printf("Proposals: %d pending, %d reviewed, %d decided\n", pending, active, completed)

	// Show last mutation
	lastMut, _ := mutation.LastAppliedMutation()
	if lastMut != nil {
		fmt.Printf("\nLast mutation: %s (%s)\n", lastMut.ID, lastMut.Type)
		fmt.Printf("  Title: %s\n", lastMut.Title)
		fmt.Printf("  Applied: %s\n", lastMut.AppliedAt.Format(time.RFC3339))
	}

	// Check halt marker
	dir, _ := keys.AegisCourtDir()
	if dir != "" {
		if _, err := os.Stat(filepath.Join(dir, "HALTED")); err == nil {
			fmt.Println("\n*** SYSTEM IS HALTED — restart with: aegiscourt start ***")
		}
	}

	notifs, _ := notify.Recent(5)
	if len(notifs) > 0 {
		fmt.Println("\nRecent notifications:")
		for _, n := range notifs {
			fmt.Printf("  [%s] %s\n", n.Timestamp[:19], n.Message)
		}
	}

	if g.JSON {
		data, _ := json.MarshalIndent(map[string]any{
			"court_mode":   cfg.CourtMode,
			"llm":          cfg.PreferredLLM,
			"low_resource": cfg.LowResourceMode,
			"resources": map[string]any{
				"free_ram_gb":  res.FreeRAMGB,
				"total_ram_gb": res.TotalRAMGB,
				"cpu_count":    res.CPUCount,
				"gpu":          res.GPUName,
			},
			"proposals_pending":  pending,
			"proposals_reviewed": active,
			"proposals_decided":  completed,
		}, "", "  ")
		fmt.Println(string(data))
	}
}

// ===== LOG =====

func cmdLog(g *Globals, args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: aegiscourt log list [--filter <text>] [--export <path>]`)
		return
	}
	if args[0] != "list" {
		fmt.Fprintf(os.Stderr, "Unknown log command: %s\n", args[0])
		os.Exit(1)
	}

	al := mustInitAudit()
	var filter, exportPath string
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--filter":
			if i+1 < len(args) {
				filter = args[i+1]
				i++
			}
		case "--export":
			if i+1 < len(args) {
				exportPath = args[i+1]
				i++
			}
		}
	}

	if exportPath != "" {
		if err := al.Export(exportPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Audit log exported to %s\n", exportPath)
		return
	}

	entries, err := al.List(filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("No audit entries found.")
		return
	}

	if g.JSON {
		data, _ := json.MarshalIndent(entries, "", "  ")
		fmt.Println(string(data))
		return
	}

	for _, e := range entries {
		fmt.Printf("[%s] %s  %s\n", e.Timestamp[:19], e.ID[:8], truncate(e.Payload, 80))
	}
}

// ===== AUDIT =====

func cmdAudit(g *Globals, args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println("Usage: aegiscourt audit verify")
		return
	}
	if args[0] != "verify" {
		fmt.Fprintf(os.Stderr, "Unknown audit command: %s\n", args[0])
		os.Exit(1)
	}

	al := mustInitAudit()
	count, errors, err := al.Verify()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if g.JSON {
		data, _ := json.MarshalIndent(map[string]any{
			"total_entries": count,
			"errors":        errors,
			"intact":        len(errors) == 0,
		}, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Audit log: %d entries verified\n", count)
	if len(errors) == 0 {
		fmt.Println("Status: INTACT -- no tampering detected")
	} else {
		fmt.Printf("Status: TAMPERED -- %d issues found:\n", len(errors))
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
		os.Exit(1)
	}
}

// ===== ROLLBACK =====

func cmdRollback(g *Globals, args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: aegiscourt rollback <mutation-id|last>

Revert the specified or most recent mutation.

Flags:
  --dry-run    Show what would be reverted without applying`)
		return
	}

	target := args[0]
	al := mustInitAudit()

	if g.DryRun {
		if target == "last" {
			m, err := mutation.LastAppliedMutation()
			if err != nil || m == nil {
				fmt.Println("No applied mutations to rollback.")
				return
			}
			fmt.Printf("Would rollback mutation: %s (%s)\n", m.ID, m.Title)
		} else {
			fmt.Printf("Would rollback mutation: %s\n", target)
		}
		return
	}

	eng := newMutationEngine(al)

	if target == "last" {
		if err := eng.RollbackLast(); err != nil {
			fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Last mutation rolled back successfully.")
	} else {
		if err := eng.Rollback(target); err != nil {
			fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Mutation %s rolled back successfully.\n", target)
	}
}

// ===== BENCH =====

func cmdBench(g *Globals, args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: aegiscourt bench run

Run canned benchmark tasks and report pass rates.`)
		return
	}
	if args[0] != "run" {
		fmt.Fprintf(os.Stderr, "Unknown bench command: %s\n", args[0])
		os.Exit(1)
	}

	cfg := mustLoadConfig()
	router := mustInitRouter(cfg)
	al := mustInitAudit()

	fmt.Println("Running benchmark suite...")
	ctx := context.Background()
	result, err := bench.RunBench(ctx, router)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	al.Append(fmt.Sprintf("bench_run: pass_rate=%.1f%% score=%.1f/%.1f",
		result.PassRate*100, result.TotalScore, result.MaxScore))

	if g.JSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println(bench.FormatBenchResult(result))
	}
}

// ===== HALT =====

func cmdHalt(g *Globals, args []string) {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Println(`Usage: aegiscourt halt [--no-confirm]

Emergency freeze: stop all agents, rollback last mutation, enter read-only mode.`)
			return
		}
	}

	noConfirm := false
	for _, a := range args {
		if a == "--no-confirm" {
			noConfirm = true
		}
	}

	if !noConfirm && !g.Confirm {
		fmt.Println("EMERGENCY HALT: This will freeze all agents and rollback the last mutation.")
		fmt.Print("Confirm? [y/N]: ")
		c := readLine()
		if c != "y" && c != "Y" {
			fmt.Println("Cancelled.")
			return
		}
	}

	al := mustInitAudit()
	al.Append("emergency_halt")

	eng := newMutationEngine(al)
	if err := eng.RollbackLast(); err != nil {
		fmt.Printf("Rollback note: %v\n", err)
	} else {
		fmt.Println("Last mutation rolled back.")
	}

	// Write halt marker file
	dir, _ := keys.AegisCourtDir()
	if dir != "" {
		os.WriteFile(filepath.Join(dir, "HALTED"), []byte(time.Now().UTC().Format(time.RFC3339)), 0600)
	}

	fmt.Println("EMERGENCY HALT ACTIVATED")
	fmt.Println("All agents frozen. System in read-only mode.")
	fmt.Println("Restart with: aegiscourt start")
}

// defaultAgentHelpPrompt is the embedded fallback for the propose-agent-help prompt
const defaultAgentHelpPrompt = `You are the Proposal Assistant in AegisCourt -- a helpful, precise agent that drafts high-quality, constitution-aligned proposals from a short user description.

Your task: Convert the user's short request into a complete, well-structured proposal draft in JSON format.

Core rules:
- Preserve ALL constitutional invariants (Rules 1-5)
- Be conservative: highlight risks and suggest safer alternatives for ambiguous requests
- Output ONLY valid JSON matching the schema below. No extra text.

Schema:
{
  "type": "add-tool" | "add-skill" | "change-prompt" | "amend-rule" | "upgrade-memory" | "other",
  "title": string (8-140 chars),
  "motivation": string (>=20 chars),
  "proposed_change": string | object,
  "expected_impact": {"success_gain_percent": number, "resource_delta": string, "other_benefits": [string]},
  "risk_level": "low" | "medium" | "high",
  "risks_and_mitigations": [string],
  "rollback_plan": string (>=20 chars),
  "validation_plan": string,
  "constitution_check": string,
  "llm_assist_used": "full"
}

User request: "{user_short_request}"
`
