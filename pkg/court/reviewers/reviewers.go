package reviewers

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"AegisCourt/llm"
)

//go:embed *.md
var reviewersFS embed.FS

var prompts map[string]string

func init() {
	prompts = make(map[string]string)
	entries, err := reviewersFS.ReadDir(".")
	if err != nil {
		panic(fmt.Sprintf("failed to read reviewers dir: %v", err))
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".md") {
			name := strings.TrimSuffix(entry.Name(), ".md")
			data, err := reviewersFS.ReadFile(entry.Name())
			if err != nil {
				panic(fmt.Sprintf("failed to read %s: %v", entry.Name(), err))
			}
			prompts[name] = string(data)
		}
	}
}

func LoadPrompt(persona string) (string, error) {
	prompt, ok := prompts[persona]
	if !ok {
		return "", fmt.Errorf("persona %s not found", persona)
	}
	return prompt, nil
}

func CallReviewer(persona string, proposalJSON string) (*ReviewerOutput, error) {
	prompt, err := LoadPrompt(persona)
	if err != nil {
		return nil, err
	}

	fullPrompt := prompt + "\n\nEvaluate the following proposal:\n" + proposalJSON + "\n\nOutput your review as JSON matching this schema:\n" + schemaJSON

	response, err := llm.CallLLM(fullPrompt, "")
	if err != nil {
		return nil, err
	}

	var output ReviewerOutput
	if err := json.Unmarshal([]byte(response), &output); err != nil {
		// Retry once
		retryPrompt := fullPrompt + "\n\nYour previous output was not valid JSON. Please output valid JSON matching the schema."
		response, err = llm.CallLLM(retryPrompt, "")
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(response), &output); err != nil {
			return nil, fmt.Errorf("failed to parse reviewer output after retry: %v", err)
		}
	}

	if err := output.Validate(); err != nil {
		// Retry once
		retryPrompt := fullPrompt + "\n\nYour previous output did not match the schema. Please output JSON that validates against the schema."
		response, err = llm.CallLLM(retryPrompt, "")
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(response), &output); err != nil {
			return nil, fmt.Errorf("failed to parse reviewer output after schema retry: %v", err)
		}
		if err := output.Validate(); err != nil {
			return nil, fmt.Errorf("schema validation failed after retry: %v", err)
		}
	}

	return &output, nil
}