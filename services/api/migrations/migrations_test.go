package migrations

import (
	"strings"
	"testing"
)

func TestValidate_Passes(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	list, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) < 2 {
		t.Fatalf("migrations = %d, want at least 2", len(list))
	}
	if list[0].Version != "000001" || list[0].Name != "baseline" {
		t.Fatalf("first = %s_%s, want 000001_baseline", list[0].Version, list[0].Name)
	}
	for _, mig := range list {
		if strings.TrimSpace(mig.Up) == "" || strings.TrimSpace(mig.Down) == "" {
			t.Fatalf("%s: empty up/down", mig.Version)
		}
	}
}

func TestList_VersionsAreSorted(t *testing.T) {
	list, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for i := 1; i < len(list); i++ {
		if list[i-1].Version >= list[i].Version {
			t.Fatalf("unsorted: %s before %s", list[i-1].Version, list[i].Version)
		}
	}
}
