package kernel

import (
	"fmt"
	"strings"
)

// Rule represents a constitution rule.
type Rule struct {
	Number   int
	Priority string
	Text     string
}

// Checker evaluates actions against constitution rules.
type Checker struct {
	Rules []Rule
}

// NewChecker creates a new rule checker by parsing the constitution.
func NewChecker(constitution string) *Checker {
	rules := parseRules(constitution)
	return &Checker{Rules: rules}
}

// Check evaluates an action against the rules.
// Returns allowed, violating rule numbers, and reason.
func (c *Checker) Check(action string) (bool, []int, string) {
	var violations []int
	var reasons []string
	for _, rule := range c.Rules {
		if violatesRule(action, rule) {
			violations = append(violations, rule.Number)
			reasons = append(reasons, fmt.Sprintf("Rule %d violation", rule.Number))
		}
	}
	allowed := len(violations) == 0
	reason := strings.Join(reasons, "; ")
	return allowed, violations, reason
}

// parseRules parses the constitution text into rules.
// Hard-coded for MVP.
func parseRules(constitution string) []Rule {
	// For MVP, hard-code the rules based on the known structure
	rules := []Rule{
		{Number: 1, Priority: "Absolute", Text: "Never Cause Irreversible Harm"},
		{Number: 2, Priority: "Absolute", Text: "Enforce Strict Isolation Boundaries"},
		{Number: 3, Priority: "Absolute", Text: "No Unauthorized Host or External Access"},
		{Number: 4, Priority: "High Priority", Text: "Require Human-in-the-Loop for High-Impact Actions"},
		{Number: 5, Priority: "High Priority", Text: "Prevent Memory Poisoning & Prompt Injection"},
		{Number: 6, Priority: "Governance", Text: "Every Self-Modification Must Pass Court Review"},
		{Number: 7, Priority: "Governance", Text: "Immutable, Tamper-Evident Audit Trail"},
		{Number: 8, Priority: "Governance", Text: "Supply-Chain & Model Risk Awareness"},
		{Number: 9, Priority: "Evolution", Text: "Favor Reversible, Measurable Improvements"},
		{Number: 10, Priority: "Override", Text: "Override & Emergency Halt"},
	}
	// TODO: actually parse from text
	return rules
}

// violatesRule checks if an action violates a specific rule.
// Hard-coded checks for MVP.
func violatesRule(action string, rule Rule) bool {
	switch rule.Number {
	case 1:
		// Harm: check for keywords like delete, transfer, etc.
		if strings.Contains(action, "delete") || strings.Contains(action, "transfer") || strings.Contains(action, "harm") {
			return true
		}
	case 2:
		// Isolation: check for shared memory, etc.
		if strings.Contains(action, "shared") || strings.Contains(action, "escape") {
			return true
		}
	case 3:
		// Host access: file, network, exec
		if strings.Contains(action, "file") || strings.Contains(action, "network") || strings.Contains(action, "exec") {
			return true
		}
		// For others, no check for MVP
	}
	return false
}
