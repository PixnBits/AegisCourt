package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pixnbits/aegiscourt/pkg/keys"
)

// Tool describes a registered tool available to the agent.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Handler     string         `json:"handler"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	OutputType  string         `json:"output_type,omitempty"`
}

// Registry holds all registered tools and provides the agent runtime.
type Registry struct {
	Tools []Tool `json:"tools"`
	path  string
}

func registryPath() (string, error) {
	dir, err := keys.AegisCourtDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tools.json"), nil
}

// LoadRegistry reads the tool registry from disk or creates a default one.
func LoadRegistry() (*Registry, error) {
	rp, err := registryPath()
	if err != nil {
		return nil, err
	}
	r := &Registry{path: rp}

	data, err := os.ReadFile(rp)
	if err != nil {
		if os.IsNotExist(err) {
			r.Tools = defaultTools()
			return r, r.Save()
		}
		return nil, fmt.Errorf("read tools.json: %w", err)
	}
	if err := json.Unmarshal(data, &r.Tools); err != nil {
		return nil, fmt.Errorf("parse tools.json: %w", err)
	}
	return r, nil
}

// Save persists the registry to disk.
func (r *Registry) Save() error {
	data, err := json.MarshalIndent(r.Tools, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0600)
}

// AddTool registers a new tool. Returns error if name already exists.
func (r *Registry) AddTool(t Tool) error {
	for _, existing := range r.Tools {
		if existing.Name == t.Name {
			return fmt.Errorf("tool %q already registered", t.Name)
		}
	}
	r.Tools = append(r.Tools, t)
	return r.Save()
}

// RemoveTool removes a tool by name.
func (r *Registry) RemoveTool(name string) error {
	for i, t := range r.Tools {
		if t.Name == name {
			r.Tools = append(r.Tools[:i], r.Tools[i+1:]...)
			return r.Save()
		}
	}
	return fmt.Errorf("tool %q not found", name)
}

// FindTool looks up a tool by name.
func (r *Registry) FindTool(name string) *Tool {
	for i := range r.Tools {
		if r.Tools[i].Name == name {
			return &r.Tools[i]
		}
	}
	return nil
}

// BuildSystemPrompt generates the system prompt including available tools.
func (r *Registry) BuildSystemPrompt() string {
	var b strings.Builder
	b.WriteString("You are AegisCourt Agent, a helpful AI assistant that operates under constitutional governance.\n")
	b.WriteString("You have access to the following tools:\n\n")

	for _, t := range r.Tools {
		fmt.Fprintf(&b, "## %s\n", t.Name)
		fmt.Fprintf(&b, "%s\n", t.Description)
		if t.Handler != "" {
			fmt.Fprintf(&b, "Handler: %s\n", t.Handler)
		}
		if len(t.Parameters) > 0 {
			fmt.Fprintf(&b, "Parameters: %v\n", t.Parameters)
		}
		b.WriteString("\n")
	}

	b.WriteString("When you need to use a tool, respond with:\n")
	b.WriteString("TOOL_CALL: <tool_name> <arguments>\n\n")
	b.WriteString("Always explain your reasoning before and after tool use.\n")

	// Load custom agent prompt if exists
	if custom := loadCustomPrompt(); custom != "" {
		b.WriteString("\n---\nCustom Instructions:\n")
		b.WriteString(custom)
		b.WriteString("\n")
	}

	return b.String()
}

// ExecuteTool runs a tool by name with the given arguments.
func ExecuteTool(r *Registry, toolName, args string) (string, error) {
	t := r.FindTool(toolName)
	if t == nil {
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}

	switch t.Handler {
	case "builtin:echo":
		return args, nil
	case "builtin:utc_time":
		return time.Now().UTC().Format(time.RFC3339), nil
	default:
		return "", fmt.Errorf("unsupported handler: %s", t.Handler)
	}
}

func defaultTools() []Tool {
	return []Tool{
		{
			Name:        "echo",
			Description: "Echoes back the input. Useful for testing.",
			Handler:     "builtin:echo",
			OutputType:  "text",
		},
	}
}

func loadCustomPrompt() string {
	dir, err := keys.AegisCourtDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, "agent-prompt.txt"))
	if err != nil {
		return ""
	}
	return string(data)
}
