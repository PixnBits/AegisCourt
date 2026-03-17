package reviewers

import (
	_ "embed"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed schema.json
var schemaJSON string

type ReviewerOutput struct {
	Score                int      `json:"score"`
	Recommendation       string   `json:"recommendation"`
	KeyConcerns          []string `json:"key_concerns"`
	RequiredMitigations  []string `json:"required_mitigations"`
	Pros                 []string `json:"pros"`
	Cons                 []string `json:"cons"`
	Rationale            string   `json:"rationale"`
}

func (r *ReviewerOutput) Validate() error {
	schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
	documentLoader := gojsonschema.NewGoLoader(r)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}
	if !result.Valid() {
		return fmt.Errorf("validation errors: %v", result.Errors())
	}
	return nil
}