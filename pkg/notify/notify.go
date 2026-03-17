package notify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pixnbits/aegiscourt/pkg/keys"
)

type Notification struct {
	Timestamp  string `json:"timestamp"`
	Type       string `json:"type"`
	ProposalID string `json:"proposal_id,omitempty"`
	Message    string `json:"message"`
}

func notifyPath() (string, error) {
	dir, err := keys.AegisCourtDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "notifications.jsonl"), nil
}

func Emit(notifType, proposalID, message string) error {
	path, err := notifyPath()
	if err != nil {
		return err
	}
	if err := keys.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	n := Notification{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Type:       notifType,
		ProposalID: proposalID,
		Message:    message,
	}
	data, err := json.Marshal(n)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

func List() ([]Notification, error) {
	path, err := notifyPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var notifs []Notification
	for _, line := range splitLines(data) {
		var n Notification
		if err := json.Unmarshal([]byte(line), &n); err == nil {
			notifs = append(notifs, n)
		}
	}
	return notifs, nil
}

func splitLines(data []byte) []string {
	var lines []string
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			line := string(data[start:i])
			if len(line) > 0 {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(data) {
		s := string(data[start:])
		if len(s) > 0 {
			lines = append(lines, s)
		}
	}
	return lines
}

func Recent(n int) ([]Notification, error) {
	all, err := List()
	if err != nil {
		return nil, err
	}
	if len(all) <= n {
		return all, nil
	}
	return all[len(all)-n:], nil
}

func Unread() (int, error) {
	path, err := notifyPath()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return len(splitLines(data)), nil
}

func ClearAll() error {
	path, err := notifyPath()
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func EmitCourtStarted(proposalID, title string) error {
	return Emit("court_started", proposalID, fmt.Sprintf("Court review started for: %s", title))
}

func EmitCourtCompleted(proposalID, recommendation string, score float64) error {
	return Emit("court_completed", proposalID, fmt.Sprintf("Court completed: %s (score: %.0f)", recommendation, score))
}

func EmitVoteCast(proposalID, action string) error {
	return Emit("vote_cast", proposalID, fmt.Sprintf("Vote cast: %s", action))
}

func EmitProposalSubmitted(proposalID, title string) error {
	return Emit("proposal_submitted", proposalID, fmt.Sprintf("Proposal submitted: %s", title))
}
