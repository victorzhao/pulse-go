package app

import (
	"pulse/internal/platform"
	"pulse/internal/store"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// settingsModal shows the settings dialog.
func (a *App) settingsModal(c *ui.Context, p Palette) {
	if !a.settings.Open {
		return
	}
	m := ui.Modal(c, &a.settings.Open, func() {
		ui.Column(c).Width(720).Height(560).Gap(10).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Icon(c, icon("settings")).FontSize(18).TextColor(p.Accent)
				ui.Text(c, a.t("settingsTitle")).FontSize(17).Bold()
				ui.Spacer(c)
			})
			ui.Tabs(c, &a.settings.Tab, a.t("appearance"), a.t("viewSettings"), a.t("general"), a.t("language"), a.t("aboutPage"))
			ui.Scroll(c).Grow(1).Children(func() {
				switch a.settings.Tab {
				case 0:
					a.appearanceSettings(c, p)
				case 1:
					a.viewSettings(c, p)
				case 2:
					a.generalSettings(c, p)
				case 3:
					a.languageSettings(c, p)
				case 4:
					a.aboutSettings(c, p)
				}
			})
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, a.t("resetSettings")).Clicked() {
					a.confirm = confirmState{
						Open: true, Title: a.t("resetSettings"), Body: a.t("resetConfirm"),
						Confirm: a.t("reset"), Danger: true,
						Action: func() {
							lang := a.cfg.Language
							a.cfg = store.Default()
							a.cfg.Language = lang
							a.settings.Open = true
							a.settings.Tab = 0
							a.saveSessionSoon()
						},
					}
				}
				ui.Spacer(c)
				if ui.Button(c, a.t("close")).Clicked() {
					a.settings.Open = false
					a.saveSessionSoon()
				}
			})
		})
	})
	if m.Dismissed() {
		a.saveSessionSoon()
	}
}

// sectionTitle draws a settings section heading.
func (a *App) sectionTitle(c *ui.Context, p Palette, text, help string) {
	ui.Column(c).Gap(2).Padding(0, 0, 6).Children(func() {
		ui.Text(c, text).FontSize(12.5).Bold().TextColor(p.Text)
		if help != "" {
			ui.Text(c, help).FontSize(10.5).TextColor(p.TextMuted).MaxLines(2)
		}
	})
}

// settingRow lays out a label and its control.
func settingRow(c *ui.Context, p Palette, label string, fn func()) {
	ui.Row(c).FillWidth().Padding(2, 0).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(11.5).Width(190).TextColor(p.TextMuted)
		ui.Box(c).Grow(1).Children(fn)
	})
}

// appearanceSettings builds the appearance page.
func (a *App) appearanceSettings(c *ui.Context, p Palette) {
	ui.Column(c).Gap(8).Children(func() {
		a.sectionTitle(c, p, a.t("appearance"), a.t("appearanceHelp"))
		themeIdx := 0
		switch a.cfg.Appearance.Theme {
		case "light":
			themeIdx = 1
		case "dark":
			themeIdx = 2
		}
		settingRow(c, p, a.t("theme"), func() {
			if ui.Segmented(c, &themeIdx, a.t("themeSystem"), a.t("themeLight"), a.t("themeDark")).Changed() {
				switch themeIdx {
				case 1:
					a.cfg.Appearance.Theme = "light"
				case 2:
					a.cfg.Appearance.Theme = "dark"
				default:
					a.cfg.Appearance.Theme = "system"
				}
			}
		})
		settingRow(c, p, a.t("accentColor"), func() {
			ui.Row(c).Gap(6).Children(func() {
				for _, col := range []string{"#0078d4", "#8764b8", "#107c10", "#f7630c", "#e81123", "#c239b3", "#0099bc", "#767676"} {
					col := col
					sw := ui.Box(c).Size(26, 26).Radius(13).Background(ui.Hex(col)).Border(2, p.Border).Cursor(ui.CursorPointer)
					if a.cfg.Appearance.Accent == col {
						sw.Border(2, p.Text)
					}
					sw.OnClick(func() {
						a.cfg.Appearance.Accent = col
						a.saveSessionSoon()
					})
				}
			})
		})
		settingRow(c, p, a.t("wallpaper"), func() {
			ui.Row(c).Gap(8).Children(func() {
				label := a.cfg.Appearance.Wallpaper
				if label == "" {
					label = a.t("none")
				}
				ui.Text(c, label).FontSize(10.5).TextColor(p.TextMuted).SingleLine().Grow(1)
				if ui.Button(c, a.t("chooseImage")).Clicked() {
					paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
						Parent:  a.Win,
						Title:   a.t("wallpaper"),
						Filters: []mygo.FileFilter{{Name: "Images", Extensions: []string{"png", "jpg", "jpeg", "gif", "bmp", "webp"}}},
					})
					if err == nil && len(paths) > 0 {
						a.cfg.Appearance.Wallpaper = paths[0]
						a.wp.tried = false
						a.loadWallpaper()
						a.saveSessionSoon()
					}
				}
				if ui.Button(c, a.t("clearImage")).Disabled(a.cfg.Appearance.Wallpaper == "").Clicked() {
					a.cfg.Appearance.Wallpaper = ""
					a.wp.bmp = nil
					a.wp.tried = false
					a.saveSessionSoon()
				}
			})
		})
		settingRow(c, p, a.t("transparency"), func() {
			v := float64(a.cfg.Appearance.Transparency)
			if ui.Slider(c, &v, 0, 100).Width(260).Changed() {
				a.cfg.Appearance.Transparency = int(v)
			}
		})
		settingRow(c, p, a.t("blur"), func() {
			v := float64(a.cfg.Appearance.Blur)
			if ui.Slider(c, &v, 0, 100).Width(260).Changed() {
				a.cfg.Appearance.Blur = int(v)
			}
		})
		rowIdx := 1
		switch a.cfg.Appearance.RowHeight {
		case "compact":
			rowIdx = 0
		case "roomy":
			rowIdx = 2
		}
		settingRow(c, p, a.t("rowHeight"), func() {
			if ui.Segmented(c, &rowIdx, a.t("rowCompact"), a.t("rowStandard"), a.t("rowRoomy")).Changed() {
				switch rowIdx {
				case 0:
					a.cfg.Appearance.RowHeight = "compact"
				case 2:
					a.cfg.Appearance.RowHeight = "roomy"
				default:
					a.cfg.Appearance.RowHeight = "standard"
				}
			}
		})
		renderIdx := 0
		switch a.cfg.Appearance.TextRender {
		case "clear":
			renderIdx = 1
		case "smooth":
			renderIdx = 2
		}
		settingRow(c, p, a.t("textRender"), func() {
			if ui.Segmented(c, &renderIdx, a.t("textAuto"), a.t("textClear"), a.t("textSmooth")).Changed() {
				switch renderIdx {
				case 1:
					a.cfg.Appearance.TextRender = "clear"
				case 2:
					a.cfg.Appearance.TextRender = "smooth"
				default:
					a.cfg.Appearance.TextRender = "auto"
				}
			}
		})
	})
}

// viewSettings builds the view page.
func (a *App) viewSettings(c *ui.Context, p Palette) {
	ui.Column(c).Gap(6).Children(func() {
		a.sectionTitle(c, p, a.t("viewSettings"), a.t("viewHelp"))
		ui.Checkbox(c, &a.cfg.View.Zebra, a.t("zebra"))
		ui.Checkbox(c, &a.cfg.View.ExtensionColors, a.t("extensionColors"))
		ui.Checkbox(c, &a.cfg.View.SizeBars, a.t("sizeBars"))
		ui.Checkbox(c, &a.cfg.View.SmartDates, a.t("smartDates"))
		if ui.Checkbox(c, &a.cfg.View.FoldersFirst, a.t("foldersFirstSetting")).Changed() {
			if p := a.activePane(); p != nil {
				a.applyPane(p)
			}
		}
		if ui.Checkbox(c, &a.cfg.View.ShowHidden, a.t("showHidden")).Changed() {
			if p := a.activePane(); p != nil {
				a.applyPane(p)
			}
		}
		if ui.Checkbox(c, &a.cfg.View.ComputeFolderSize, a.t("computeFolderSize")).Changed() {
			if p := a.activePane(); p != nil {
				a.computeSizes(p)
			}
		}
		if ui.Checkbox(c, &a.cfg.View.AutoRefresh, a.t("autoRefresh")).Changed() {
			a.refreshWatchers()
		}
		ui.Checkbox(c, &a.cfg.View.VerticalTabs, a.t("verticalTabs"))
		ui.Checkbox(c, &a.cfg.View.ShowDriveBars, a.t("driveBars"))
		ui.Checkbox(c, &a.cfg.View.ConfirmDelete, a.t("confirmDelete"))
		if ui.Checkbox(c, &a.detailsOn, a.t("showDetails")).Changed() {
			a.cfg.View.ShowDetails = a.detailsOn
		}
	})
}

// generalSettings builds the general page.
func (a *App) generalSettings(c *ui.Context, p Palette) {
	ui.Column(c).Gap(6).Children(func() {
		a.sectionTitle(c, p, a.t("general"), a.t("generalHelp"))
		ui.Checkbox(c, &a.cfg.General.RestoreTabs, a.t("restoreTabs"))
		ui.Checkbox(c, &a.cfg.General.SingleClickOpen, a.t("singleClickOpen"))
		if ui.Checkbox(c, &a.cfg.General.Tray, a.t("tray")).Changed() {
			a.applyTray()
		}
		if ui.Checkbox(c, &a.cfg.General.Startup, a.t("startup")).Changed() {
			if err := platform.SetAutostart(a.cfg.General.Startup); err != nil {
				a.toast = a.t("saveFailed")
			}
		}
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, a.t("defaultLocation")).FontSize(11.5).Width(190).TextColor(p.TextMuted)
			ui.Text(c, a.cfg.General.DefaultLocation).FontSize(10.5).TextColor(p.Text).SingleLine().Grow(1)
			if ui.Button(c, a.t("browse")).Clicked() {
				paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.Win, Title: a.t("defaultLocation"), Directory: true})
				if err == nil && len(paths) > 0 {
					a.cfg.General.DefaultLocation = paths[0]
				}
			}
		})
	})
}

// languageSettings builds the language page.
func (a *App) languageSettings(c *ui.Context, p Palette) {
	ui.Column(c).Gap(6).Children(func() {
		a.sectionTitle(c, p, a.t("language"), a.t("languageHelp"))
		cur := a.cfg.Language
		ui.RadioGroup(c, func() {
			ui.Radio(c, &cur, "", a.t("systemLanguage"))
			ui.Radio(c, &cur, "zh-Hans", a.t("simplifiedChinese"))
			ui.Radio(c, &cur, "zh-Hant", a.t("traditionalChinese"))
			ui.Radio(c, &cur, "en", a.t("english"))
		})
		if cur != a.cfg.Language {
			a.setLocale(cur)
		}
	})
}

// aboutSettings builds the about page.
func (a *App) aboutSettings(c *ui.Context, p Palette) {
	ui.Column(c).Gap(8).Children(func() {
		ui.Row(c).Gap(12).Children(func() {
			ui.Icon(c, icon("logo")).Size(48, 48).TextColor(p.Accent)
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, a.t("app.name")).FontSize(16).Bold()
				ui.Text(c, a.tf("version")+" 0.1.0 · MyGO").FontSize(10.5).TextColor(p.TextMuted)
			})
		})
		ui.Text(c, a.t("app.about")).FontSize(11).TextColor(p.TextMuted).MaxLines(3)
		ui.Divider(c)
		a.sectionTitle(c, p, a.t("keyboard"), "")
		ui.Column(c).Gap(3).Children(func() {
			for _, row := range [][2]string{
				{"Ctrl+T", a.t("shortcutNewTab")},
				{"Ctrl+W", a.t("shortcutCloseTab")},
				{"Alt+← / Alt+→", a.t("shortcutBack") + " / " + a.t("shortcutForward")},
				{"Alt+↑", a.t("shortcutUp")},
				{"Ctrl+L", a.t("shortcutAddress")},
				{"Ctrl+F", a.t("shortcutSearch")},
				{"Ctrl+K", a.t("shortcutGlobal")},
				{"Ctrl+Shift+F", a.t("shortcutAdvanced")},
				{"Ctrl+Shift+N", a.t("shortcutNewFolder")},
				{"F2", a.t("shortcutRename")},
				{"Delete", a.t("shortcutDelete")},
				{"F5", a.t("shortcutRefresh")},
				{"Ctrl+A", a.t("shortcutSelectAll")},
				{"Ctrl+C / Ctrl+X", a.t("shortcutCopy") + " / " + a.t("shortcutCut")},
				{"Ctrl+V", a.t("shortcutPaste")},
				{"Ctrl+1…5", a.t("shortcutLayouts")},
				{"Ctrl+B", a.t("shortcutSidebar")},
				{"Ctrl+D", a.t("shortcutDetails")},
				{"Alt+1…5", a.t("shortcutViews")},
			} {
				ui.Row(c).Gap(10).Children(func() {
					ui.Text(c, row[0]).FontSize(10.5).Width(130).TextColor(p.Text).FontFeatures("tnum")
					ui.Text(c, row[1]).FontSize(10.5).TextColor(p.TextMuted)
				})
			}
		})
	})
}
