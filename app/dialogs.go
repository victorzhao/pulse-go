package app

import (
	"context"
	"path/filepath"
	"time"

	"pulse/internal/fsx"
	"pulse/internal/store"

	"github.com/egoist/mygo/ui"
)

// renameModal asks for a new name.
func (a *App) renameModal(c *ui.Context, p Palette) {
	if !a.rename.Open {
		return
	}
	ui.Modal(c, &a.rename.Open, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			ui.Text(c, a.t("renameTitle")).FontSize(15).Bold()
			ti := ui.TextInput(c, &a.rename.Name).Placeholder(a.t("renamePlaceholder")).AutoFocus()
			if ti.Submitted() {
				a.commitRename()
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, a.t("cancel")).Clicked() {
					a.rename.Open = false
				}
				if ui.PrimaryButton(c, a.t("rename")).Clicked() {
					a.commitRename()
				}
			})
		})
	})
}

// newFolderModal asks for a folder name.
func (a *App) newFolderModal(c *ui.Context, p Palette) {
	if !a.newFolder.Open {
		return
	}
	ui.Modal(c, &a.newFolder.Open, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			ui.Text(c, a.t("newFolderTitle")).FontSize(15).Bold()
			ti := ui.TextInput(c, &a.newFolder.Name).Placeholder(a.t("newFolderPlaceholder")).AutoFocus()
			if ti.Submitted() {
				a.commitNewFolder()
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, a.t("cancel")).Clicked() {
					a.newFolder.Open = false
				}
				if ui.PrimaryButton(c, a.t("create")).Clicked() {
					a.commitNewFolder()
				}
			})
		})
	})
}

// propertiesModal shows the properties of the chosen entries.
func (a *App) propertiesModal(c *ui.Context, p Palette) {
	if !a.props.Open {
		return
	}
	ui.Modal(c, &a.props.Open, func() {
		ui.Column(c).Width(560).Gap(10).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, a.t("propertiesTitle")).FontSize(15).Bold()
				if len(a.props.Entries) == 1 {
					ui.Image(c, kindIcon(a.props.Entries[0].Kind)).Size(20, 20)
				}
				ui.Spacer(c)
			})
			if len(a.props.Entries) == 0 {
				ui.Text(c, a.t("noProperties")).TextColor(p.TextMuted)
			} else if len(a.props.Entries) > 1 {
				entries := a.props.Entries
				var total int64
				folders, files := 0, 0
				dirs := map[string]bool{}
				for _, e := range entries {
					if e.IsDir {
						folders++
					} else {
						files++
						total += e.Size
					}
					dirs[filepath.Dir(e.Path)] = true
				}
				location := a.t("multipleLocations")
				if len(dirs) == 1 {
					for d := range dirs {
						location = d
					}
				}
				ui.Text(c, a.tf("totalItems", len(entries))).FontSize(13).Bold()
				ui.Divider(c)
				ui.Column(c).Gap(5).Children(func() {
					a.infoRow(c, p, a.t("locationLabel"), location)
					a.infoRow(c, p, a.t("folders"), itoa(folders))
					a.infoRow(c, p, a.t("files"), itoa(files))
					a.infoRow(c, p, a.t("sizeLabel"), fsx.FormatSize(total))
				})
			} else {
				e := a.props.Entries[0]
				ui.Text(c, e.Name).FontSize(13).Bold().SingleLine()
				ui.Text(c, e.Path).FontSize(10.5).TextColor(p.TextMuted).SingleLine()
				ui.Divider(c)
				ui.Column(c).Gap(5).Children(func() {
					a.infoRow(c, p, a.t("type"), a.typeLabel(e))
					if e.IsDir {
						if a.props.Busy {
							a.infoRow(c, p, a.t("sizeLabel"), a.t("calculating"))
						} else if a.props.FolderSize > 0 || a.props.FolderFiles > 0 {
							a.infoRow(c, p, a.t("sizeLabel"), fsx.FormatSize(a.props.FolderSize))
							a.infoRow(c, p, a.t("containsLabel"), a.tf("folderContains", a.props.FolderFiles, a.props.FolderDirs))
						}
					} else {
						a.infoRow(c, p, a.t("sizeLabel"), fsx.FormatSize(e.Size))
					}
					a.infoRow(c, p, a.t("modified"), a.fullDate(e.ModTime))
					a.infoRow(c, p, a.t("attributesLabel"), a.attributesText(e))
				})
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, a.t("close")).Clicked() {
					a.props.Open = false
				}
			})
		})
	})
}

// attributesText writes a file's attributes.
func (a *App) attributesText(e fsx.Entry) string {
	var parts []string
	if e.Mode&0200 == 0 {
		parts = append(parts, a.t("readOnlyLabel"))
	}
	if e.Hidden {
		parts = append(parts, a.t("hiddenLabel"))
	}
	if e.Mode&0111 != 0 {
		parts = append(parts, a.t("executableMode"))
	}
	if len(parts) == 0 {
		return a.t("none")
	}
	out := ""
	for i, s := range parts {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

// showProperties opens the properties dialog for entries.
func (a *App) showProperties(entries []fsx.Entry) {
	if a.Win == nil {
		return
	}
	a.props = propsState{Open: true, Entries: entries}
	if len(entries) == 1 && entries[0].IsDir {
		a.props.Busy = true
		a.props.Gen++
		gen := a.props.Gen
		dir := entries[0].Path
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			size, files, dirs, _ := fsx.DirStats(ctx, dir, 0)
			a.Win.Update(func() {
				if a.props.Gen != gen {
					return
				}
				a.props.Busy = false
				a.props.FolderSize = size
				a.props.FolderFiles = files
				a.props.FolderDirs = dirs
			})
		}()
	}
}

// confirmModal asks a yes/no question.
func (a *App) confirmModal(c *ui.Context, p Palette) {
	if !a.confirm.Open {
		return
	}
	ui.Modal(c, &a.confirm.Open, func() {
		ui.Column(c).Width(430).Gap(12).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				if a.confirm.Danger {
					ui.Icon(c, icon("warning")).FontSize(18).TextColor(p.Danger)
				}
				ui.Text(c, a.confirm.Title).FontSize(15).Bold().SingleLine()
			})
			ui.Text(c, a.confirm.Body).FontSize(11.5).TextColor(p.TextMuted).MaxLines(4)
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, a.t("cancel")).Clicked() {
					a.confirm.Open = false
				}
				label := a.confirm.Confirm
				if label == "" {
					label = a.t("ok")
				}
				var btn ui.Element
				if a.confirm.Danger {
					btn = ui.Button(c, label).Background(p.Danger).TextColor(ui.Hex("#ffffff"))
				} else {
					btn = ui.PrimaryButton(c, label)
				}
				if btn.Clicked() {
					act := a.confirm.Action
					a.confirm.Open = false
					if act != nil {
						act()
					}
				}
			})
		})
	})
}

// conflictModal asks what to do when a destination exists.
func (a *App) conflictModal(c *ui.Context, p Palette) {
	if !a.conflict.Open {
		return
	}
	m := ui.Modal(c, &a.conflict.Open, func() {
		ui.Column(c).Width(460).Gap(12).Children(func() {
			ui.Text(c, a.t("conflictTitle")).FontSize(15).Bold()
			ui.Text(c, a.tf("conflictBody", filepath.Base(a.conflict.Path))).FontSize(11.5).TextColor(p.TextMuted).SingleLine()
			ui.Text(c, a.conflict.Path).FontSize(10).TextColor(p.TextFaint).SingleLine()
			ui.Checkbox(c, &a.conflict.ApplyAll, a.t("conflictApplyAll"))
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, a.t("cancel")).Clicked() {
					a.conflict.Open = false
					if a.conflict.Resume != nil {
						a.conflict.Resume(-1, false)
					}
				}
				if ui.Button(c, a.t("skip")).Clicked() {
					a.conflict.Open = false
					if a.conflict.Resume != nil {
						a.conflict.Resume(1, a.conflict.ApplyAll)
					}
				}
				if a.conflict.Merge {
					if ui.Button(c, a.t("merge")).Clicked() {
						a.conflict.Open = false
						if a.conflict.Resume != nil {
							a.conflict.Resume(3, a.conflict.ApplyAll)
						}
					}
				}
				if ui.Button(c, a.t("conflictKeepBoth")).Clicked() {
					a.conflict.Open = false
					if a.conflict.Resume != nil {
						a.conflict.Resume(2, a.conflict.ApplyAll)
					}
				}
				if ui.PrimaryButton(c, a.t("overwrite")).Clicked() {
					a.conflict.Open = false
					if a.conflict.Resume != nil {
						a.conflict.Resume(0, a.conflict.ApplyAll)
					}
				}
			})
		})
	})
	if m.Dismissed() {
		if a.conflict.Resume != nil {
			a.conflict.Resume(-1, false)
		}
	}
}

// tagsModal manages the color labels.
func (a *App) tagsModal(c *ui.Context, p Palette) {
	if !a.tags.Open {
		return
	}
	ui.Modal(c, &a.tags.Open, func() {
		ui.Column(c).Width(520).Gap(10).Children(func() {
			ui.Text(c, a.t("manageTags")).FontSize(15).Bold()
			ui.Text(c, a.t("tagsHelp")).FontSize(10.5).TextColor(p.TextMuted).MaxLines(2)
			ui.Column(c).Gap(4).Children(func() {
				for i, tag := range a.cfg.Tags {
					i := i
					tag := tag
					row := ui.Row(c).Height(30).Padding(4, 6).Gap(8).Radius(5)
					if a.tags.Editing == i {
						row.Background(fade(p.Accent, 0.14))
					}
					row.Children(func() {
						ui.Box(c).Size(10, 10).Radius(5).Background(ui.Hex(tag.Color))
						ui.Text(c, tag.Name).FontSize(12).Grow(1).SingleLine().TextColor(p.Text)
						if ui.Button(c, "").Width(22).Height(22).Radius(4).Children(func() {
							ui.Icon(c, icon("rename")).FontSize(12)
						}).Clicked() {
							a.tags.Editing = i
							a.tags.Name = tag.Name
							a.tags.Color = tag.Color
						}
						if ui.Button(c, "").Width(22).Height(22).Radius(4).Children(func() {
							ui.Icon(c, icon("delete")).FontSize(12)
						}).Clicked() {
							a.deleteTag(tag.ID)
							if a.tags.Editing == i {
								a.tags.Editing = -1
							}
						}
					})
				}
			})
			ui.Divider(c)
			title := a.t("newTag")
			if a.tags.Editing >= 0 {
				title = a.t("editTag")
			}
			ui.Text(c, title).FontSize(12).Bold()
			ui.TextInput(c, &a.tags.Name).Placeholder(a.t("tagName"))
			ui.Row(c).Gap(6).Children(func() {
				for _, col := range []string{"#e81123", "#f7630c", "#107c10", "#ffb900", "#8764b8", "#0078d4", "#767676"} {
					col := col
					sw := ui.Box(c).Size(24, 24).Radius(12).Background(ui.Hex(col)).Border(2, p.Border).Cursor(ui.CursorPointer)
					if a.tags.Color == col {
						sw.Border(2, p.Text)
					}
					sw.OnClick(func() { a.tags.Color = col })
				}
			})
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, a.t("cancel")).Clicked() {
					a.tags.Open = false
				}
				if ui.Button(c, a.t("reset")).Clicked() {
					a.tags.Editing = -1
					a.tags.Name = ""
					a.tags.Color = "#0078d4"
				}
				if ui.PrimaryButton(c, a.t("save")).Clicked() {
					if a.tags.Color == "" {
						a.tags.Color = "#0078d4"
					}
					a.saveTagEdits()
				}
			})
		})
	})
}

// defaultTagColor picks a color for a new tag.
func defaultTagColor(tags []store.Tag) string {
	colors := []string{"#e81123", "#f7630c", "#107c10", "#ffb900", "#8764b8", "#0078d4", "#767676"}
	return colors[len(tags)%len(colors)]
}
