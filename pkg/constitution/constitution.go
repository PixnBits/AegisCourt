package constitution

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed initial_rules_v0.1.md
var rulesMarkdown string

type Rule struct {
	ID       int
	Text     string
	Priority string
	Enforced bool
}

var rules map[int]Rule

func init() {
	rules = parseRules(rulesMarkdown)
}

func parseRules(md string) map[int]Rule {
	r := make(map[int]Rule)
	lines := strings.Split(md, "\n")
	id := 1
	for _, line := range lines {
		// Parse rule
		if strings.HasPrefix(line, "**Rule") {
			r[id] = Rule{
				ID:       id,
				Text:     line,
				Priority: "High", // placeholder
				Enforced: true,
			}
			id++
		}
	}
	return r
}

func GetRules() map[int]Rule {
	return rules
}

func Enforce(ruleID int, action string) error {
	rule, ok := rules[ruleID]
	if !ok {
		return fmt.Errorf("rule %d not found", ruleID)
	}
	if !rule.Enforced {
		return nil
	}
	actionLower := strings.ToLower(action)
	switch ruleID {
	case 1: // Never Cause Irreversible Harm
		if strings.Contains(actionLower, "harm") || strings.Contains(actionLower, "delete") || strings.Contains(actionLower, "transfer") {
			return fmt.Errorf("action violates Rule 1: %s", rule.Text)
		}
	case 2: // Enforce Strict Isolation Boundaries
		// Assume sandbox handles this
	case 3: // No Unauthorized Host or External Access
		if strings.Contains(actionLower, "host") || strings.Contains(actionLower, "file") || strings.Contains(actionLower, "network") {
			return fmt.Errorf("action violates Rule 3: %s", rule.Text)
		}
	case 5: // Prevent Memory Poisoning & Prompt Injection
		if strings.Contains(actionLower, "jailbreak") || strings.Contains(actionLower, "inject") {
			return fmt.Errorf("action violates Rule 5: %s", rule.Text)
		}
	}
	return nil
}
