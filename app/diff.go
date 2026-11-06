package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pulse/internal/fsx"

	"github.com/egoist/mygo/ui"
)

// openDiff compares two text files.
func (a *App) openDiff(pathA, pathB string) {
	if a.Win == nil {
		return
	}
	infoA, err := os.Stat(pathA)
	if err != nil {
		a.toast = a.t("readFailed")
		return
	}
	infoB, err := os.Stat(pathB)
	if err != nil {
		a.toast = a.t("readFailed")
		return
	}
	if infoA.Size() > 8*1024*1024 || infoB.Size() > 8*1024*1024 {
		a.toast = a.t("fileTooLarge")
		return
	}
	a.diff = diffState{Open: true, Title: filepath.Base(pathA) + " ↔ " + filepath.Base(pathB)}
	go func() {
		dataA, errA := os.ReadFile(pathA)
		dataB, errB := os.ReadFile(pathB)
		a.Win.Update(func() {
			if errA != nil || errB != nil {
				a.diff.Open = false
				a.toast = a.t("readFailed")
				return
			}
			if isBinary(dataA) || isBinary(dataB) {
				a.diff.Open = false
				a.toast = a.t("binaryFile")
				return
			}
			a.diff.A = string(dataA)
			a.diff.B = string(dataB)
			a.recomputeDiff()
		})
	}()
}

// isBinary reports whether data looks like a binary file.
func isBinary(data []byte) bool {
	limit := len(data)
	if limit > 8192 {
		limit = 8192
	}
	for i := 0; i < limit; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

// recomputeDiff computes the rows of the diff dialog.
func (a *App) recomputeDiff() {
	d := a.diff
	if d.A == "" && d.B == "" {
		d.Rows = nil
		a.diff = d
		return
	}
	rows := fsx.DiffLines(d.A, d.B)
	if d.IgnoreWS || d.IgnoreCase {
		la := splitTextLines(d.A)
		lb := splitTextLines(d.B)
		na := make([]string, len(la))
		nb := make([]string, len(lb))
		for i, s := range la {
			na[i] = normalizeDiffLine(s, d.IgnoreWS, d.IgnoreCase)
		}
		for i, s := range lb {
			nb[i] = normalizeDiffLine(s, d.IgnoreWS, d.IgnoreCase)
		}
		nr := fsx.DiffLines(strings.Join(na, "\n"), strings.Join(nb, "\n"))
		for i := range nr {
			if nr[i].ANo > 0 && nr[i].ANo <= len(la) {
				nr[i].A = la[nr[i].ANo-1]
			}
			if nr[i].BNo > 0 && nr[i].BNo <= len(lb) {
				nr[i].B = lb[nr[i].BNo-1]
			}
		}
		rows = nr
	}
	d.Rows = rows
	d.At = 0
	a.diff = d
}

func splitTextLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	out := strings.Split(s, "\n")
	if len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

func normalizeDiffLine(s string, ws, caseInsensitive bool) string {
	if ws {
		s = strings.Join(strings.Fields(s), " ")
	}
	if caseInsensitive {
		s = strings.ToLower(s)
	}
	return s
}

// diffRowsVisible returns the rows to show.
func (a *App) diffRowsVisible() []fsx.DiffRow {
	if !a.diff.OnlyDiff {
		return a.diff.Rows
	}
	out := make([]fsx.DiffRow, 0, len(a.diff.Rows))
	for _, r := range a.diff.Rows {
		if r.Kind != ' ' {
			out = append(out, r)
		}
	}
	return out
}

// diffModal shows the text comparison.
func (a *App) diffModal(c *ui.Context, p Palette) {
	if !a.diff.Open {
		return
	}
	ui.Modal(c, &a.diff.Open, func() {
		ui.Column(c).Width(900).Height(620).Gap(8).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Icon(c, icon("diff")).FontSize(17).TextColor(p.Accent)
				ui.Text(c, a.diff.Title).FontSize(14).Bold().SingleLine().Grow(1)
				if ui.Button(c, a.t("close")).Clicked() {
					a.diff.Open = false
				}
			})
			if a.diff.Error != "" {
				ui.Text(c, a.diff.Error).FontSize(11).TextColor(p.Danger)
			}
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				mode := a.diff.Mode
				if ui.Segmented(c, &mode, a.t("sideBySide"), a.t("unified")).Changed() {
					a.diff.Mode = mode
				}
				only := a.diff.OnlyDiff
				if ui.Checkbox(c, &only, a.t("onlyDifferences")).Changed() {
					a.diff.OnlyDiff = only
				}
				ws := a.diff.IgnoreWS
				if ui.Checkbox(c, &ws, a.t("ignoreWhitespace")).Changed() {
					a.diff.IgnoreWS = ws
					a.recomputeDiff()
				}
				ic := a.diff.IgnoreCase
				if ui.Checkbox(c, &ic, a.t("ignoreCase")).Changed() {
					a.diff.IgnoreCase = ic
					a.recomputeDiff()
				}
				ui.Spacer(c)
				equal, removed, added := fsx.DiffStats(a.diff.Rows)
				ui.Text(c, a.tf("diffEqual", equal)).FontSize(10).TextColor(p.Success)
				ui.Text(c, a.tf("diffRemoved", removed)).FontSize(10).TextColor(p.Danger)
				ui.Text(c, a.tf("diffAdded", added)).FontSize(10).TextColor(p.Success)
			})
			if len(a.diff.Rows) == 0 {
				ui.Text(c, a.t("identical")).FontSize(12).TextColor(p.TextMuted)
			} else {
				rows := a.diffRowsVisible()
				a.diff.List.Key = func(i int) any { return i }
				list := ui.List(c, &a.diff.List, len(rows), func(i int) {
					if i < 0 || i >= len(rows) {
						return
					}
					r := rows[i]
					if a.diff.Mode == 1 {
						a.diffUnifiedRow(c, p, r)
					} else {
						a.diffSideRow(c, p, r)
					}
				}).Grow(1)
				_ = list
			}
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, "◀ "+a.t("prevDifference")).Clicked() {
					a.jumpDiff(-1)
				}
				if ui.Button(c, a.t("nextDifference")+" ▶").Clicked() {
					a.jumpDiff(1)
				}
				ui.Spacer(c)
				ui.Text(c, a.tf("lines", len(a.diff.Rows))).FontSize(10).TextColor(p.TextMuted)
			})
		})
	})
}

// diffSideRow draws one row side by side.
func (a *App) diffSideRow(c *ui.Context, p Palette, r fsx.DiffRow) {
	ui.Row(c).FillWidth().Height(21).Children(func() {
		a.diffHalf(c, p, r.ANo, r.A, r.Kind == '-', true)
		ui.Box(c).Width(1).Height(21).Background(p.BorderSoft)
		a.diffHalf(c, p, r.BNo, r.B, r.Kind == '+', false)
	})
}

// diffHalf draws one side of a side-by-side row.
func (a *App) diffHalf(c *ui.Context, p Palette, no int, text string, changed, left bool) {
	bg := ui.Transparent
	if changed {
		bg = fade(p.Danger, 0.16)
	}
	half := ui.Row(c).Grow(1).Height(21).Gap(6).Padding(0, 6).Clip()
	half.Background(bg)
	half.Children(func() {
		label := ""
		if no > 0 {
			label = fmt.Sprint(no)
		}
		ui.Text(c, label).Width(40).FontSize(9.5).TextColor(p.TextFaint).TextAlign(ui.End).FontFeatures("tnum")
		ui.Text(c, text).FontSize(10.5).Font("Consolas").SingleLine().Grow(1).TextColor(p.Text)
	})
}

// diffUnifiedRow draws one row in the unified view.
func (a *App) diffUnifiedRow(c *ui.Context, p Palette, r fsx.DiffRow) {
	bg := ui.Transparent
	sign := " "
	text := r.A
	no := r.ANo
	switch r.Kind {
	case '-':
		bg = fade(p.Danger, 0.16)
		sign = "−"
		text = r.A
	case '+':
		bg = fade(p.Success, 0.16)
		sign = "+"
		text = r.B
		no = r.BNo
	}
	row := ui.Row(c).FillWidth().Height(21).Gap(6).Padding(0, 6).Clip().Background(bg)
	row.Children(func() {
		ui.Text(c, sign).Width(14).FontSize(10.5).Font("Consolas").TextColor(p.TextMuted)
		label := ""
		if no > 0 {
			label = fmt.Sprint(no)
		}
		ui.Text(c, label).Width(40).FontSize(9.5).TextColor(p.TextFaint).TextAlign(ui.End).FontFeatures("tnum")
		ui.Text(c, text).FontSize(10.5).Font("Consolas").SingleLine().Grow(1).TextColor(p.Text)
	})
}

// jumpDiff moves to the next or previous difference.
func (a *App) jumpDiff(dir int) {
	rows := a.diffRowsVisible()
	if len(rows) == 0 {
		return
	}
	at := a.diff.At + dir
	for at >= 0 && at < len(rows) {
		if rows[at].Kind != ' ' {
			a.diff.At = at
			a.diff.List.ScrollIntoView(at)
			return
		}
		at += dir
	}
}

// compareModal shows the folder comparison summary.
func (a *App) compareModal(c *ui.Context, p Palette) {
	if !a.compare.Open {
		return
	}
	ui.Modal(c, &a.compare.Open, func() {
		ui.Column(c).Width(620).Height(480).Gap(8).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Icon(c, icon("compare")).FontSize(17).TextColor(p.Accent)
				ui.Text(c, a.t("compareTitle")).FontSize(15).Bold()
				ui.Spacer(c)
				if ui.Button(c, a.t("close")).Clicked() {
					a.compare.Open = false
				}
			})
			st := a.compare
			ui.Row(c).Gap(14).Children(func() {
				a.compareStat(c, p, a.t("compareSame"), st.Same, p.Success)
				a.compareStat(c, p, a.t("compareLeftOnly"), st.LeftOnly, p.Warning)
				a.compareStat(c, p, a.t("compareRightOnly"), st.RightOnly, p.Warning)
				a.compareStat(c, p, a.t("compareNewer"), st.Newer, p.TagBlue)
				a.compareStat(c, p, a.t("compareDifferent"), st.Different, p.Danger)
			})
			ui.Divider(c)
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(3).Children(func() {
					for _, e := range st.Left.Entries {
						mark, ok := st.Left.Compare[e.Path]
						if !ok || mark.Status == 5 {
							continue
						}
						ui.Row(c).Height(24).Gap(8).Padding(2, 4).Children(func() {
							ui.Text(c, a.compareLabel(mark.Status)).FontSize(10).Width(80).TextColor(a.compareColor(mark.Status, p))
							ui.Text(c, e.Name).FontSize(11).Grow(1).SingleLine().TextColor(p.Text)
							ui.Text(c, fsx.FormatSize(e.Size)).FontSize(10).TextColor(p.TextFaint)
						})
					}
					for _, e := range st.Right.Entries {
						mark, ok := st.Right.Compare[e.Path]
						if !ok || mark.Status != 2 {
							continue
						}
						ui.Row(c).Height(24).Gap(8).Padding(2, 4).Children(func() {
							ui.Text(c, a.compareLabel(mark.Status)).FontSize(10).Width(80).TextColor(a.compareColor(mark.Status, p))
							ui.Text(c, e.Name).FontSize(11).Grow(1).SingleLine().TextColor(p.Text)
							ui.Text(c, fsx.FormatSize(e.Size)).FontSize(10).TextColor(p.TextFaint)
						})
					}
				})
			})
		})
	})
}

func (a *App) compareStat(c *ui.Context, p Palette, label string, n int, col ui.Color) {
	ui.Column(c).Gap(1).Children(func() {
		ui.Text(c, fmt.Sprint(n)).FontSize(17).Bold().TextColor(col)
		ui.Text(c, label).FontSize(9.5).TextColor(p.TextMuted)
	})
}
