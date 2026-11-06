package app

import (
	"path/filepath"

	"pulse/internal/fsx"
	"pulse/internal/platform"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// detailsPanel shows the selected item, or the folder when nothing is
// selected.
func (a *App) detailsPanel(c *ui.Context, p Palette) {
	panel := ui.Column(c).Fill().Clip().Background(p.Panel).BorderWidth(0, 1, 0, 0).BorderColor(p.Border)
	panel.Children(func() {
		pane := a.activePane()
		if pane == nil {
			return
		}
		entries := a.selectedEntries(pane)
		if len(entries) == 0 {
			a.folderDetails(c, p, pane)
			return
		}
		if len(entries) > 1 {
			a.multiDetails(c, p, pane)
			return
		}
		a.itemDetails(c, p, pane, &entries[0])
	})
}

// itemDetails draws one item's details.
func (a *App) itemDetails(c *ui.Context, p Palette, pane *Pane, e *fsx.Entry) {
	ui.Scroll(c).Grow(1).Padding(12).Gap(10).Children(func() {
		// The preview card: a thumbnail for pictures, else the kind icon.
		card := ui.Box(c).FillWidth().Height(168).Radius(8).Background(p.Surface).Border(1, p.BorderSoft)
		card.Children(func() {
			if e.Kind == fsx.KindImage {
				if bmp := a.thumbFor(*e); bmp != nil {
					ui.Image(c, bmp).Fill().Fit(ui.Contain).Padding(6)
					return
				}
			}
			ui.Column(c).Fill().Center().Children(func() {
				ui.Image(c, kindIcon(e.Kind)).Size(64, 64)
				ui.Text(c, a.typeName(e.Kind)).FontSize(11).TextColor(p.TextMuted)
			})
		})
		// Name, type and size.
		ui.Row(c).Gap(8).Children(func() {
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, e.Name).FontSize(15).Bold().SingleLine().TextColor(p.Text)
				meta := a.typeLabel(*e)
				if !e.IsDir {
					meta += " · " + fsx.FormatSize(e.Size)
				}
				ui.Text(c, meta).FontSize(10.5).TextColor(p.TextMuted).SingleLine()
			})
			if ui.Button(c, "").Width(26).Height(26).Radius(6).Children(func() {
				if a.isFavorite(e.Path) {
					ui.Icon(c, icon("star-filled")).FontSize(15)
				} else {
					ui.Icon(c, icon("star")).FontSize(15)
				}
			}).Tooltip(a.t("starred")).Clicked() {
				a.toggleFavorite(*e)
			}
			ui.Button(c, "").Width(26).Height(26).Radius(6).Children(func() {
				ui.Icon(c, icon("tag")).FontSize(15)
			}).Tooltip(a.t("addTagButton")).PopoverMenu(func(m *ui.Menu) {
				for _, tag := range a.cfg.Tags {
					if m.Item(tag.Name).Radio(a.tagOf(e.Path) == tag.ID).Chosen() {
						a.setTag([]string{e.Path}, tag.ID)
					}
				}
				if a.tagOf(e.Path) != "" {
					m.Separator()
					if m.Item(a.t("removeTag")).Chosen() {
						a.setTag([]string{e.Path}, "")
					}
				}
			})
		})
		// Actions.
		ui.Row(c).FillWidth().Gap(6).Children(func() {
			openBtn := ui.PrimaryButton(c, "").Grow(1).Height(52).Radius(8).Padding(0, 2).Children(func() {
				ui.Column(c).Gap(3).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icon("folder-open")).FontSize(18)
					ui.Text(c, a.t("open")).FontSize(9.5).SingleLine()
				})
			})
			if openBtn.Clicked() {
				a.activateEntry(pane, *e)
			}
			if ui.Button(c, "").Grow(1).Height(52).Radius(8).Children(func() {
				ui.Column(c).Gap(3).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icon("plus")).FontSize(18)
					ui.Text(c, a.t("openInNewTabShort")).FontSize(9.5).SingleLine()
				})
			}).Clicked() {
				if e.IsDir {
					a.openInNewTab(e.Path)
				} else {
					a.openInNewTab(filepath.Dir(e.Path))
				}
			}
			if ui.Button(c, "").Grow(1).Height(52).Radius(8).Children(func() {
				ui.Column(c).Gap(3).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icon("copy")).FontSize(18)
					ui.Text(c, a.t("copyPathShort")).FontSize(9.5).SingleLine()
				})
			}).Clicked() {
				mygo.Clipboard.WriteText(e.Path)
			}
			ui.Button(c, "").Grow(1).Height(52).Radius(8).Padding(0, 2).Children(func() {
				ui.Column(c).Gap(3).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icon("more")).FontSize(18)
					ui.Text(c, a.t("moreActions")).FontSize(9.5).SingleLine()
				})
			}).PopoverMenu(func(m *ui.Menu) {
				if m.Item(a.t("rename")).Shortcut(0, ui.KeyF2).Chosen() {
					a.beginRename(pane, *e)
				}
				if m.Item(a.t("delete")).Shortcut(0, ui.KeyDelete).Chosen() {
					a.deleteEntries(pane, []fsx.Entry{*e}, false)
				}
				if m.Item(a.t("duplicate")).Chosen() {
					a.duplicateEntry(pane, *e)
				}
				m.Separator()
				if m.Item(a.t("properties")).Chosen() {
					a.showProperties([]fsx.Entry{*e})
				}
				if !e.IsDir {
					if m.Item(a.t("openWith")).Chosen() {
						_ = platform.OpenWith(e.Path)
					}
				}
				if m.Item(a.t("openLocation")).Chosen() {
					_ = platform.Reveal(e.Path)
				}
			})
		})
		// Basic information.
		ui.Collapsible(c, a.t("folderInfo"), &a.detailsBasicOpen, func() {
			ui.Column(c).Gap(4).Children(func() {
				a.infoRow(c, p, a.t("locationLabel"), filepath.Dir(e.Path))
				a.infoRow(c, p, a.t("type"), a.typeLabel(*e))
				if e.IsDir {
					if s, ok := pane.Sizes[e.Path]; ok {
						a.infoRow(c, p, a.t("sizeLabel"), fsx.FormatSize(s))
					}
				} else {
					a.infoRow(c, p, a.t("sizeLabel"), fsx.FormatSize(e.Size))
				}
				a.infoRow(c, p, a.t("modifiedLabel"), a.fullDate(e.ModTime))
				if e.Kind == fsx.KindDisk {
					if d, ok := a.findDrive(pane, e.Path); ok {
						a.infoRow(c, p, a.t("driveType"), a.driveTypeName(d))
						a.infoRow(c, p, a.t("capacityLabel"), fsx.FormatSize(d.Total))
						a.infoRow(c, p, a.t("freeLabel"), fsx.FormatSize(d.Free))
						a.capacityBar(c, p, d.Total-d.Free, d.Total, 200)
					}
				}
				if e.Link {
					a.infoRow(c, p, a.t("attributesLabel"), a.t("symlinkMode"))
				}
			})
		})
		// Tags.
		ui.Collapsible(c, a.t("tagsLabel"), &a.detailsTagsOpen, func() {
			ui.Row(c).Wrap().Gap(5).Children(func() {
				assigned := a.tagOf(e.Path)
				if assigned == "" {
					ui.Text(c, a.t("noTags")).FontSize(11).TextColor(p.TextFaint)
				}
				for _, tag := range a.cfg.Tags {
					if tag.ID != assigned {
						continue
					}
					ui.Text(c, tag.Name).FontSize(10.5).TextColor(ui.Hex(tag.Color)).
						Background(fade(ui.Hex(tag.Color), 0.18)).Padding(2, 8).Radius(999)
				}
				ui.Button(c, "+ "+a.t("addTagButton")).FontSize(10.5).PopoverMenu(func(m *ui.Menu) {
					for _, tag := range a.cfg.Tags {
						if m.Item(tag.Name).Radio(assigned == tag.ID).Chosen() {
							a.setTag([]string{e.Path}, tag.ID)
						}
					}
					if assigned != "" {
						m.Separator()
						if m.Item(a.t("removeTag")).Chosen() {
							a.setTag([]string{e.Path}, "")
						}
					}
				})
			})
		})
		// Attributes.
		ui.Collapsible(c, a.t("attributesLabel"), &a.detailsAttrOpen, func() {
			ui.Column(c).Gap(4).Children(func() {
				mode := e.Mode
				a.infoRow(c, p, a.t("readOnlyLabel"), yesNo(mode&0200 == 0, a))
				a.infoRow(c, p, a.t("hiddenLabel"), yesNo(e.Hidden, a))
				if mode&0111 != 0 {
					a.infoRow(c, p, a.t("executableMode"), a.t("yes"))
				}
			})
		})
		if pane.TrashIDs[e.Path] != "" {
			if ui.Button(c, a.t("restoreItem")).Clicked() {
				a.restoreTrashItem(pane.TrashIDs[e.Path], pane)
			}
		}
	})
}

// multiDetails draws a summary of several chosen items.
func (a *App) multiDetails(c *ui.Context, p Palette, pane *Pane) {
	entries := a.selectedEntries(pane)
	var total int64
	folders, files := 0, 0
	dirs := map[string]bool{}
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
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
	ui.Scroll(c).Grow(1).Padding(12).Gap(10).Children(func() {
		ui.Box(c).FillWidth().Height(120).Radius(8).Background(p.Surface).Border(1, p.BorderSoft).Children(func() {
			ui.Column(c).Fill().Center().Gap(6).Children(func() {
				ui.Icon(c, icon("select-all")).FontSize(34).TextColor(p.Accent)
				ui.Text(c, a.tf("statusSelected", len(entries))).FontSize(14).Bold().TextColor(p.Text)
				ui.Text(c, a.tf("statusSize", fsx.FormatSize(total))).FontSize(11).TextColor(p.TextMuted)
			})
		})
		ui.Column(c).Gap(4).Children(func() {
			a.infoRow(c, p, a.t("locationLabel"), location)
			a.infoRow(c, p, a.t("folders"), itoa(folders))
			a.infoRow(c, p, a.t("files"), itoa(files))
			a.infoRow(c, p, a.t("sizeLabel"), fsx.FormatSize(total))
		})
		ui.Row(c).FillWidth().Gap(6).Children(func() {
			if ui.PrimaryButton(c, a.t("openAll")).Grow(1).Height(34).Clicked() {
				for _, e := range entries {
					if err := platform.Open(e.Path); err != nil {
						a.toast = a.t("openFailed")
						break
					}
				}
			}
			ui.Button(c, a.t("tagSelected")).Grow(1).Height(34).PopoverMenu(func(m *ui.Menu) {
				for _, tag := range a.cfg.Tags {
					if m.Item(tag.Name).Chosen() {
						a.setTag(paths, tag.ID)
					}
				}
			})
			if ui.Button(c, a.t("delete")).Height(34).Clicked() {
				a.deleteEntries(pane, entries, false)
			}
		})
		if ui.Button(c, a.t("properties")).Clicked() {
			a.showProperties(entries)
		}
	})
}

// folderDetails draws the pane's folder when nothing is selected.
func (a *App) folderDetails(c *ui.Context, p Palette, pane *Pane) {
	ui.Scroll(c).Grow(1).Padding(12).Gap(10).Children(func() {
		title := a.tabTitle(pane)
		var iconName = "folder"
		switch pane.Virtual {
		case virtualComputer:
			iconName = "computer"
		case virtualTrash:
			iconName = "recycle"
		case virtualTag:
			iconName = "tag"
		case virtualStarred:
			iconName = "star"
		case virtualRecent:
			iconName = "clock"
		case virtualNetwork:
			iconName = "network"
		case virtualSearch:
			iconName = "search"
		}
		ui.Box(c).FillWidth().Height(120).Radius(8).Background(p.Surface).Border(1, p.BorderSoft).Children(func() {
			ui.Column(c).Fill().Center().Gap(6).Children(func() {
				ui.Icon(c, icon(iconName)).FontSize(34).TextColor(p.Accent)
				ui.Text(c, title).FontSize(14).Bold().SingleLine().TextColor(p.Text)
				if pane.Virtual == virtualNone {
					ui.Text(c, pane.Path).FontSize(10).TextColor(p.TextMuted).SingleLine()
				}
			})
		})
		ui.Text(c, a.t("noSelection")).FontSize(11.5).TextColor(p.TextFaint)
		folders, files := a.kindCounts(pane)
		ui.Column(c).Gap(4).Children(func() {
			a.infoRow(c, p, a.t("items"), itoa(len(pane.Entries)))
			a.infoRow(c, p, a.t("folders"), itoa(folders))
			a.infoRow(c, p, a.t("files"), itoa(files))
			if pane.Virtual == virtualNone {
				a.infoRow(c, p, a.t("sizeLabel"), fsx.FormatSize(a.sizeSum(pane)))
			}
			if pane.Virtual == virtualTrash {
				a.infoRow(c, p, a.t("trashInfo"), a.tf("trashCount", len(pane.Entries)))
				ui.Text(c, a.t("trashRestoreHint")).FontSize(10.5).TextColor(p.TextMuted)
			}
			if pane.Virtual == virtualComputer {
				ui.Text(c, a.t("computerHint")).FontSize(10.5).TextColor(p.TextMuted)
			}
			if pane.Virtual == virtualTag {
				ui.Text(c, a.t("tagHint")).FontSize(10.5).TextColor(p.TextMuted)
			}
			if pane.Virtual == virtualNone && a.cfg.View.ComputeFolderSize {
				ui.Text(c, a.t("folderSizeHint")).FontSize(10.5).TextColor(p.TextMuted)
			}
			ui.Text(c, a.t("selectionHint")).FontSize(10.5).TextColor(p.TextFaint)
		})
		if pane.Virtual == virtualTrash && len(pane.Entries) > 0 {
			if ui.Button(c, a.t("emptyTrash")).Clicked() {
				a.emptyTrash()
			}
		}
		if pane.Virtual == virtualNone && pane.Path != "" {
			if ui.Button(c, a.t("terminal")).Clicked() {
				if err := platform.OpenTerminal(pane.Path); err != nil {
					a.toast = a.t("terminalFailed")
				}
			}
		}
	})
}

// infoRow draws a label and its value.
func (a *App) infoRow(c *ui.Context, p Palette, label, value string) {
	ui.Row(c).FillWidth().Gap(10).Children(func() {
		ui.Text(c, label).Width(78).FontSize(10.5).TextColor(p.TextMuted).SingleLine()
		ui.Text(c, value).FontSize(10.5).TextColor(p.Text).Grow(1).SingleLine()
	})
}

func yesNo(v bool, a *App) string {
	if v {
		return a.t("yes")
	}
	return a.t("no")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
