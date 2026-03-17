package proposal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func getProposalsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".aegiscourt", "proposals")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func SaveDraft(d *Draft) (string, error) {
	if d.ID == "" {
		d.ID = fmt.Sprintf("draft-%s", strings.ReplaceAll(strings.ReplaceAll(time.Now().UTC().Format("20060102T150405"), "-", ""), ":", ""))
	}
	d.SetTimestamps()

	dir, err := getProposalsDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, d.ID+".json")

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(d); err != nil {
		return "", err
	}

	return d.ID, nil
}

func LoadDraft(id string) (*Draft, error) {
	dir, err := getProposalsDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, id+".json")

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var d Draft
	if err := json.NewDecoder(file).Decode(&d); err != nil {
		return nil, err
	}

	return &d, nil
}