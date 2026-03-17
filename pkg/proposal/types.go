package proposal

import (
	_ "embed"
	"fmt"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed schema.json
var schemaJSON string

type Draft struct {
	ID                string                 `json:"id,omitempty"`
	Type              string                 `json:"type"`
	Title             string                 `json:"title"`
	Motivation        string                 `json:"motivation"`
	ProposedChange    interface{}            `json:"proposed_change"` // string or object
	ExpectedImpact    *ExpectedImpact        `json:"expected_impact,omitempty"`
	RiskLevel         string                 `json:"risk_level,omitempty"`
	RisksAndMitigations []string             `json:"risks_and_mitigations,omitempty"`
	RollbackPlan      string                 `json:"rollback_plan"`
	ValidationPlan    string                 `json:"validation_plan,omitempty"`
	ConstitutionCheck string                 `json:"constitution_check,omitempty"`
	LLMAssistUsed     string                 `json:"llm_assist_used,omitempty"`
	CreatedAt         string                 `json:"created_at,omitempty"`
	LastModifiedAt    string                 `json:"last_modified_at,omitempty"`
}

type ExpectedImpact struct {
	SuccessGainPercent float64  `json:"success_gain_percent,omitempty"`
	ResourceDelta      string   `json:"resource_delta,omitempty"`
	OtherBenefits      []string `json:"other_benefits,omitempty"`
}

func (d *Draft) Validate() error {
	schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
	documentLoader := gojsonschema.NewGoLoader(d)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}
	if !result.Valid() {
		return fmt.Errorf("validation errors: %v", result.Errors())
	}
	return nil
}

func (d *Draft) SetTimestamps() {
	now := time.Now().UTC().Format(time.RFC3339)
	if d.CreatedAt == "" {
		d.CreatedAt = now
	}
	d.LastModifiedAt = now
}
