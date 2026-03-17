package court

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pixnbits/aegiscourt/pkg/keys"
)

func courtDir() (string, error) {
	dir, err := keys.AegisCourtDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "court"), nil
}

func SaveResult(result *CourtResult) error {
	dir, err := courtDir()
	if err != nil {
		return err
	}
	if err := keys.EnsureDir(dir); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, result.ProposalID+".json")
	return os.WriteFile(path, data, 0600)
}

func LoadResult(proposalID string) (*CourtResult, error) {
	dir, err := courtDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, proposalID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("court result not found for %s", proposalID)
	}
	var result CourtResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func ListResults(statusFilter string) ([]*CourtResult, error) {
	dir, err := courtDir()
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
	var results []*CourtResult
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-5]
		r, err := LoadResult(id)
		if err != nil {
			continue
		}
		if statusFilter == "" || string(r.Status) == statusFilter {
			results = append(results, r)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartedAt < results[j].StartedAt
	})
	return results, nil
}
