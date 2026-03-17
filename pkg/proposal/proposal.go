package proposal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pixnbits/aegiscourt/pkg/keys"
)

func ValidateDraft(d *Draft) []string {
	var errs []string

	validTypes := map[string]bool{
		"add-tool": true, "add-skill": true, "change-prompt": true,
		"amend-rule": true, "upgrade-memory": true, "other": true,
	}
	if d.Type == "" {
		errs = append(errs, "type is required")
	} else if !validTypes[d.Type] {
		errs = append(errs, fmt.Sprintf("invalid type: %q", d.Type))
	}

	if len(d.Title) < 8 {
		errs = append(errs, "title must be at least 8 characters")
	} else if len(d.Title) > 140 {
		errs = append(errs, "title must be at most 140 characters")
	}

	if len(d.Motivation) < 20 {
		errs = append(errs, "motivation must be at least 20 characters")
	} else if len(d.Motivation) > 1000 {
		errs = append(errs, "motivation must be at most 1000 characters")
	}

	if d.ProposedChange == nil {
		errs = append(errs, "proposed_change is required")
	} else {
		switch v := d.ProposedChange.(type) {
		case string:
			if len(v) < 10 {
				errs = append(errs, "proposed_change string must be at least 10 characters")
			}
		case map[string]any:
			// structured change is valid
		default:
			errs = append(errs, "proposed_change must be a string or object")
		}
	}

	if len(d.RollbackPlan) < 20 {
		errs = append(errs, "rollback_plan must be at least 20 characters")
	}

	if d.RiskLevel != "" {
		validRisks := map[string]bool{"low": true, "medium": true, "high": true}
		if !validRisks[d.RiskLevel] {
			errs = append(errs, fmt.Sprintf("invalid risk_level: %q", d.RiskLevel))
		}
	}

	if d.LLMAssistUsed != "" {
		validAssist := map[string]bool{"none": true, "light": true, "full": true}
		if !validAssist[d.LLMAssistUsed] {
			errs = append(errs, fmt.Sprintf("invalid llm_assist_used: %q", d.LLMAssistUsed))
		}
	}

	return errs
}

func DraftsDir() (string, error) {
	dir, err := keys.AegisCourtDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "proposals"), nil
}

func GenerateDraftID() string {
	return fmt.Sprintf("draft-%s", time.Now().UTC().Format("20060102T150405"))
}

func SaveDraft(d *Draft) error {
	dir, err := DraftsDir()
	if err != nil {
		return err
	}
	if err := keys.EnsureDir(dir); err != nil {
		return err
	}
	if d.ID == "" {
		d.ID = GenerateDraftID()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	d.LastModifiedAt = time.Now().UTC()

	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal draft: %w", err)
	}

	path := filepath.Join(dir, d.ID+".json")
	return os.WriteFile(path, data, 0600)
}

func LoadDraft(id string) (*Draft, error) {
	dir, err := DraftsDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("draft not found: %s", id)
	}
	var d Draft
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("corrupt draft: %w", err)
	}
	return &d, nil
}

func ListDrafts() ([]*Draft, error) {
	dir, err := DraftsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var drafts []*Draft
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-5]
		d, err := LoadDraft(id)
		if err == nil {
			drafts = append(drafts, d)
		}
	}
	return drafts, nil
}

func DraftToJSON(d *Draft) (string, error) {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
