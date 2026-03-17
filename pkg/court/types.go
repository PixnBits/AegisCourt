package court

import (
	"time"

	jsonpatch "github.com/evanphx/json-patch"
)

type Proposal struct {
	ID          string
	Type        string
	Name        string
	Description string
	Diff        jsonpatch.Patch
	CreatedAt   time.Time
	Status      string
}

type Engine struct {
	// stub
}
