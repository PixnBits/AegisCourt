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
	// Stub: always allow
	return nil
}
