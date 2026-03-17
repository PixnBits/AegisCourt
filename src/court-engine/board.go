package courtengine

import (
	"fmt"
	"strings"
)

// BoardEntry represents an entry in the NASA board.
type BoardEntry struct {
	Persona        string
	Score          int
	Recommendation string
	Emoji          string
}

// CourtBoard represents the aggregated board.
type CourtBoard struct {
	Entries           []BoardEntry
	AggregateScore    float64
	WeightedScore     float64
	Consensus         string
	ConditionsSummary string
}

// GenerateBoard generates the NASA-style board from reviewer results.
func GenerateBoard(results []ReviewerResult, courtMode string, userProfile map[string]float64) CourtBoard {
	var entries []BoardEntry
	var totalScore float64
	var weightedTotal float64
	var weights float64
	hasReject := false
	var conditions []string

	for _, result := range results {
		score := result.Score
		emoji := "🟢"
		if score < 6 {
			emoji = "🟡"
		}
		if score < 4 {
			emoji = "🔴"
			hasReject = true
		}

		entry := BoardEntry{
			Persona:        result.Persona,
			Score:          score,
			Recommendation: result.Recommendation,
			Emoji:          emoji,
		}
		entries = append(entries, entry)

		weight := 1.0
		if (courtMode == "manual" || result.RiskSeverity == "high") && (result.Persona == "CISO" || result.Persona == "MRM") {
			weight = 1.5
		}
		totalScore += float64(score)
		weightedTotal += float64(score) * weight
		weights += weight

		if strings.Contains(result.Recommendation, "conditions") {
			conditions = append(conditions, fmt.Sprintf("%s: %s", result.Persona, strings.Join(result.Mitigations, ", ")))
		}
	}

	aggregate := totalScore / float64(len(results))
	weighted := weightedTotal / weights

	consensus := "Approve"
	if hasReject || weighted < 6 {
		consensus = "Reject"
	} else if len(conditions) > 0 {
		consensus = "Approve with conditions"
	}

	board := CourtBoard{
		Entries:           entries,
		AggregateScore:    aggregate,
		WeightedScore:     weighted,
		Consensus:         consensus,
		ConditionsSummary: strings.Join(conditions, "; "),
	}

	return board
}

// String returns a human-readable NASA board text.
func (b CourtBoard) String() string {
	var sb strings.Builder
	sb.WriteString("AegisCourt Governance Board\n")
	sb.WriteString("===========================\n\n")
	for _, entry := range b.Entries {
		sb.WriteString(fmt.Sprintf("%s %s: %d - %s\n", entry.Emoji, entry.Persona, entry.Score, entry.Recommendation))
	}
	sb.WriteString(fmt.Sprintf("\nAggregate Score: %.1f\n", b.AggregateScore))
	sb.WriteString(fmt.Sprintf("Weighted Score: %.1f\n", b.WeightedScore))
	sb.WriteString(fmt.Sprintf("Consensus: %s\n", b.Consensus))
	if b.ConditionsSummary != "" {
		sb.WriteString(fmt.Sprintf("Conditions: %s\n", b.ConditionsSummary))
	}
	return sb.String()
}
