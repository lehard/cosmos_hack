package platform

import (
	"encoding/json"
	"slices"
	"testing"

	"ant/internal/contracts/schemas"
)

// Виды сущностей совпадают с перечислением контракта SSE (AD-20: один источник).
func TestEntityKindsMatchContract(t *testing.T) {
	raw, err := schemas.FS.ReadFile("contracts/events/common/sse-entity-changed.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Properties struct {
			Entity struct {
				Enum []string `json:"enum"`
			} `json:"entity"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if got := EntityKind("").Values(); !slices.Equal(got, s.Properties.Entity.Enum) {
		t.Fatalf("EntityKind %v ≠ контракт %v", got, s.Properties.Entity.Enum)
	}
}
