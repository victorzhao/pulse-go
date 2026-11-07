package app

import (
	"testing"

	"pulse/internal/platform"
	"pulse/internal/store"
)

func TestSessionRestoresBothSides(t *testing.T) {
	a := &App{cfg: store.Default()}
	a.cfg.General.RestoreTabs = true
	a.cfg.Session = store.Session{
		Version: 2,
		Tabs: []store.TabState{
			{Active: 0, Panes: []store.PaneState{{Path: platform.Home()}}},
			{Active: 0, Panes: []store.PaneState{{Path: platform.Home()}}},
		},
		ActiveSide:   1,
		SplitPercent: 40,
	}
	a.initSession()
	if a.left == nil || a.right == nil || len(a.left.Panes) != 1 || len(a.right.Panes) != 1 {
		t.Fatalf("sides = %+v %+v", a.left, a.right)
	}
	if a.activeSide != 1 || a.splitPercent != 40 {
		t.Fatalf("active side %d split %v", a.activeSide, a.splitPercent)
	}
}

func TestSessionMigratesGlobalTabs(t *testing.T) {
	a := &App{cfg: store.Default()}
	a.cfg.General.RestoreTabs = true
	a.cfg.Session = store.Session{
		Tabs: []store.TabState{
			{Active: 1, Panes: []store.PaneState{{Path: platform.Home()}, {Path: platform.Home()}}},
			{Active: 0, Panes: []store.PaneState{{Path: platform.Home()}}},
		},
		Active: 0,
	}
	a.initSession()
	if len(a.left.Panes) != 3 {
		t.Fatalf("left tabs = %d", len(a.left.Panes))
	}
	if len(a.right.Panes) != 1 {
		t.Fatalf("right tabs = %d", len(a.right.Panes))
	}
}
