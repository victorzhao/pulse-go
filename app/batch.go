package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pulse/internal/platform"

	"github.com/egoist/mygo/ui"
)

// beginBatchRename opens the batch rename dialog for the chosen entries.
func (a *App) beginBatchRename(p *Pane) {
	if p == nil || p.Virtual == virtualTrash || p.Virtual == virtualComputer {
		return
	}
	entries := a.selectedEntries(p)
	if len(entries) == 0 {
		return
	}
	a.batch = batchRenameState{
		Open:    true,
		Pane:    p,
		Entries: entries,
		Start:   1,
		Digits:  2,
	}
	a.batchPreview()
}

// batchPreview computes the new names and validates them.
func (a *App) batchPreview() {
	st := &a.batch
	if len(st.Entries) == 0 {
		st.Preview = nil
		st.Error = a.t("batchNoChange")
		return
	}
	if st.Digits < 1 {
		st.Digits = 1
	}
	if st.Start < 0 {
		st.Start = 0
	}
	preview := make([]string, len(st.Entries))
	targets := map[string]bool{}
	changed := false
	invalid := false
	for i, e := range st.Entries {
		base, ext := e.Name, ""
		if !st.IncludeExt {
			ext = filepath.Ext(base)
			base = strings.TrimSuffix(base, ext)
		}
		if st.Find != "" {
			base = strings.ReplaceAll(base, st.Find, st.Replace)
		}
		base = st.Prefix + base + st.Suffix
		if st.Numbering {
			base += fmt.Sprintf("%0*d", st.Digits, st.Start+i)
		}
		name := base + ext
		if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
			invalid = true
		}
		preview[i] = name
		if name != e.Name {
			changed = true
		}
		key := strings.ToLower(filepath.Join(filepath.Dir(e.Path), name))
		if targets[key] {
			invalid = true
		}
		targets[key] = true
	}
	// Targets that already exist, unless they are one of the sources.
	sources := map[string]bool{}
	for _, e := range st.Entries {
		sources[strings.ToLower(e.Path)] = true
	}
	conflict := false
	for i, e := range st.Entries {
		if preview[i] == e.Name {
			continue
		}
		target := filepath.Join(filepath.Dir(e.Path), preview[i])
		if platform.Exists(target) && !sources[strings.ToLower(target)] {
			conflict = true
		}
	}
	st.Preview = preview
	switch {
	case invalid:
		st.Error = a.t("batchInvalid")
	case conflict:
		st.Error = a.t("batchConflict")
	case !changed:
		st.Error = a.t("batchNoChange")
	default:
		st.Error = ""
	}
}

// applyBatchRename renames the chosen entries as previewed.
func (a *App) applyBatchRename() {
	a.batchPreview()
	st := a.batch
	if st.Error != "" || !st.Open {
		return
	}
	pairs := make([][2]string, 0, len(st.Entries))
	for i, e := range st.Entries {
		if st.Preview[i] == e.Name {
			continue
		}
		pairs = append(pairs, [2]string{e.Path, filepath.Join(filepath.Dir(e.Path), st.Preview[i])})
	}
	if len(pairs) == 0 {
		a.batch.Open = false
		a.toast = a.t("batchNoChange")
		return
	}
	if err := renameMany(pairs); err != nil {
		a.toast = a.t("renameFailed")
		return
	}
	for _, pair := range pairs {
		a.remapPath(pair[0], pair[1])
	}
	a.batch.Open = false
	reverse := make([][2]string, len(pairs))
	for i, pair := range pairs {
		reverse[i] = [2]string{pair[1], pair[0]}
	}
	undo := undoAction{Label: a.t("undoRename"), Run: func() error {
		if err := renameMany(reverse); err != nil {
			return err
		}
		for _, pair := range reverse {
			a.remapPath(pair[0], pair[1])
		}
		return nil
	}}
	a.pushUndo(undo)
	a.toastAction = &toastAction{Message: a.tf("batchRenamed", len(pairs)), Label: a.t("undo"), Run: func() { a.runUndo(undo) }}
	if st.Pane != nil {
		a.refreshPane(st.Pane)
	} else {
		a.refreshAllPanes()
	}
}

// renameMany renames several paths, through temporary names so that swaps
// and case-only changes work.
func renameMany(pairs [][2]string) error {
	temps := make([]string, len(pairs))
	for i, pair := range pairs {
		dir := filepath.Dir(pair[1])
		for n := 0; ; n++ {
			temp := filepath.Join(dir, fmt.Sprintf(".pulse-rn-%d-%d", time.Now().UnixNano(), i*100+n))
			if !platform.Exists(temp) {
				temps[i] = temp
				break
			}
		}
	}
	done := 0
	for i, pair := range pairs {
		if err := os.Rename(pair[0], temps[i]); err != nil {
			for j := 0; j < done; j++ {
				_ = os.Rename(temps[j], pairs[j][0])
			}
			return err
		}
		done++
	}
	done = 0
	for i, pair := range pairs {
		if err := os.Rename(temps[i], pair[1]); err != nil {
			for j := 0; j < done; j++ {
				_ = os.Rename(pairs[j][1], pairs[j][0])
			}
			for j := done; j < len(pairs); j++ {
				_ = os.Rename(temps[j], pairs[j][0])
			}
			return err
		}
		done++
	}
	return nil
}

// batchRenameModal shows the batch rename dialog.
func (a *App) batchRenameModal(c *ui.Context, p Palette) {
	if !a.batch.Open {
		return
	}
	a.batchPreview()
	ui.Modal(c, &a.batch.Open, func() {
		ui.Column(c).Width(640).Height(520).Gap(8).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Icon(c, icon("rename")).FontSize(17).TextColor(p.Accent)
				ui.Text(c, a.t("batchRename")).FontSize(15).Bold()
				ui.Spacer(c)
				ui.Text(c, a.tf("totalItems", len(a.batch.Entries))).FontSize(10.5).TextColor(p.TextMuted)
			})
			ui.Row(c).Gap(8).Children(func() {
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, a.t("findText")).FontSize(10).TextColor(p.TextMuted)
					ui.TextInput(c, &a.batch.Find).Placeholder(a.t("findText"))
				})
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, a.t("replaceWith")).FontSize(10).TextColor(p.TextMuted)
					ui.TextInput(c, &a.batch.Replace).Placeholder(a.t("replaceWith"))
				})
			})
			ui.Row(c).Gap(8).Children(func() {
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, a.t("prefix")).FontSize(10).TextColor(p.TextMuted)
					ui.TextInput(c, &a.batch.Prefix).Placeholder(a.t("prefix"))
				})
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, a.t("suffix")).FontSize(10).TextColor(p.TextMuted)
					ui.TextInput(c, &a.batch.Suffix).Placeholder(a.t("suffix"))
				})
			})
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Checkbox(c, &a.batch.Numbering, a.t("numbering"))
				ui.Text(c, a.t("startAt")).FontSize(10).TextColor(p.TextMuted)
				start := float64(a.batch.Start)
				if ui.NumberInput(c, &start, 0, 99999, 1).Width(80).Changed() {
					a.batch.Start = int(start)
				}
				ui.Text(c, a.t("digits")).FontSize(10).TextColor(p.TextMuted)
				digits := float64(a.batch.Digits)
				if ui.NumberInput(c, &digits, 1, 8, 1).Width(60).Changed() {
					a.batch.Digits = int(digits)
				}
				ui.Checkbox(c, &a.batch.IncludeExt, a.t("includeExtension"))
			})
			ui.Text(c, a.t("batchPreview")).FontSize(11).Bold().TextColor(p.TextMuted)
			if a.batch.Error != "" {
				ui.Text(c, a.batch.Error).FontSize(10.5).TextColor(p.Danger)
			}
			ui.Scroll(c).Grow(1).Border(1, p.BorderSoft).Radius(6).Padding(4).Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					for i, e := range a.batch.Entries {
						if i >= 500 {
							break
						}
						newName := ""
						if i < len(a.batch.Preview) {
							newName = a.batch.Preview[i]
						}
						ui.Row(c).Height(24).Gap(6).Padding(1, 4).Children(func() {
							ui.Text(c, e.Name).FontSize(10.5).TextColor(p.TextMuted).Grow(1).SingleLine()
							ui.Icon(c, icon("chevron-right")).FontSize(10).TextColor(p.TextFaint)
							col := p.Text
							if newName == e.Name {
								col = p.TextFaint
							}
							ui.Text(c, newName).FontSize(10.5).TextColor(col).Grow(1).SingleLine()
						})
					}
				})
			})
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, a.t("cancel")).Clicked() {
					a.batch.Open = false
				}
				if ui.PrimaryButton(c, a.t("applyRename")).Disabled(a.batch.Error != "").Clicked() {
					a.applyBatchRename()
				}
			})
		})
	})
}
