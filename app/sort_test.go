package app

import (
	"testing"

	"pulse/internal/fsx"

	"github.com/egoist/mygo/ui"
)

func TestMultiColumnSort(t *testing.T) {
	entries := []fsx.Entry{
		{Name: "a.txt", Size: 2},
		{Name: "b.txt", Size: 1},
		{Name: "a.txt", Size: 5},
	}
	orders := []ui.SortOrder{
		{Column: "name"},
		{Column: "size", Descending: true},
	}
	sortEntries(entries, orders, "name", false, false, nil)
	if entries[0].Size != 5 || entries[1].Size != 2 || entries[2].Name != "b.txt" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestNaturalCompare(t *testing.T) {
	if naturalCompare("file2", "file10") >= 0 {
		t.Fatal("numeric order failed")
	}
	if naturalCompare("A", "a") != 0 {
		t.Fatal("case-insensitive compare failed")
	}
}
