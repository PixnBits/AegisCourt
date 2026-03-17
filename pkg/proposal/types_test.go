package proposal

import (
	"testing"
)

func TestDraftValidate(t *testing.T) {
	// Valid draft
	valid := Draft{
		Type:           "add-tool",
		Title:          "Add mediated UTC time tool",
		Motivation:     "Agent frequently needs current time for scheduling/reminders but open web_search risks unnecessary network exposure and Rule 1 violations.",
		ProposedChange: "New tool: utc_time\nParameters: none\nOutput: ISO8601 UTC string",
		RollbackPlan:   "Remove tool registration from agent config; revert kernel mediation diff.",
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("Valid draft failed validation: %v", err)
	}

	// Invalid: missing required field
	invalid := valid
	invalid.Type = ""
	if err := invalid.Validate(); err == nil {
		t.Error("Invalid draft passed validation")
	}

	// Invalid: title too short
	invalid2 := valid
	invalid2.Title = "Short"
	if err := invalid2.Validate(); err == nil {
		t.Error("Invalid title passed validation")
	}
}