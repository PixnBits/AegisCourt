package proposal

import "time"

type Draft struct {
	ID                  string          `json:"id,omitempty"`
	Type                string          `json:"type" validate:"required,oneof=add-tool add-skill change-prompt amend-rule upgrade-memory other"`
	Title               string          `json:"title" validate:"required,min=8,max=140"`
	Motivation          string          `json:"motivation" validate:"required,min=20,max=1000"`
	ProposedChange      any             `json:"proposed_change" validate:"required"` // string or map[string]any
	ExpectedImpact      *ExpectedImpact `json:"expected_impact,omitempty"`
	RiskLevel           string          `json:"risk_level,omitempty" validate:"omitempty,oneof=low medium high"`
	RisksAndMitigations []string        `json:"risks_and_mitigations,omitempty"`
	RollbackPlan        string          `json:"rollback_plan" validate:"required,min=20"`
	ValidationPlan      string          `json:"validation_plan,omitempty"`
	ConstitutionCheck   string          `json:"constitution_check,omitempty"`
	LLMAssistUsed       string          `json:"llm_assist_used,omitempty" validate:"omitempty,oneof=none light full"`
	CreatedAt           time.Time       `json:"created_at"`
	LastModifiedAt      time.Time       `json:"last_modified_at,omitempty"`
}

type ExpectedImpact struct {
	SuccessGainPercent float64  `json:"success_gain_percent,omitempty"`
	ResourceDelta      string   `json:"resource_delta,omitempty"`
	OtherBenefits      []string `json:"other_benefits,omitempty"`
}
