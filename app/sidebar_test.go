package app

import (
	"slices"
	"testing"
)

func TestMoveSideSection(t *testing.T) {
	a := testApp(t)
	a.moveSideSection("tags", "starred")
	if a.side.order[0] != "tags" {
		t.Fatalf("order = %v", a.side.order)
	}
	a.moveSideSection("network", "recent")
	idx := slices.Index(a.side.order, "network")
	ri := slices.Index(a.side.order, "recent")
	if idx != ri-1 {
		t.Fatalf("order = %v", a.side.order)
	}
}

func TestNormalizeSideOrder(t *testing.T) {
	got := normalizeSideOrder([]string{"tags", "unknown", "tags"})
	if got[0] != "tags" || len(got) != 6 {
		t.Fatalf("order = %v", got)
	}
	seen := map[string]bool{}
	for _, id := range got {
		seen[id] = true
	}
	for _, id := range []string{"starred", "quickAccess", "recent", "tags", "computer", "network"} {
		if !seen[id] {
			t.Fatalf("missing %s in %v", id, got)
		}
	}
}
