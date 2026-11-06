package app

import "github.com/egoist/mygo/ui"

// flatButton draws a button without a face until the pointer rests on it.
func flatButton(c *ui.Context, p Palette, w, h float32) ui.Element {
	b := ui.Button(c, "").Width(w).Height(h).Radius(5).Padding(0).
		Background(ui.Transparent).Border(0, ui.Transparent)
	if b.Hovered() {
		b.Background(p.SurfaceHi)
	}
	if b.Pressed() {
		b.Background(p.SurfaceDown)
	}
	return b
}

// flatIconButton draws a flat button with an icon and a tooltip.
func flatIconButton(c *ui.Context, p Palette, iconName, tip string, w, h float32) ui.Element {
	return flatButton(c, p, w, h).Children(func() {
		ui.Icon(c, icon(iconName)).FontSize(15).TextColor(p.TextMuted)
	}).Tooltip(tip).Label(tip)
}

// flatMenuButton draws a flat button with an icon, a label and a menu.
func flatMenuButton(c *ui.Context, p Palette, iconName, label string, build func(m *ui.Menu)) ui.Element {
	b := ui.Button(c, "").Height(28).Radius(5).Padding(6, 8).Background(ui.Transparent).Border(0, ui.Transparent)
	if b.Hovered() {
		b.Background(p.SurfaceHi)
	}
	if b.Pressed() {
		b.Background(p.SurfaceDown)
	}
	return b.Children(func() {
		ui.Row(c).Gap(5).Children(func() {
			ui.Icon(c, icon(iconName)).FontSize(13).TextColor(p.TextMuted)
			ui.Text(c, label).FontSize(11).TextColor(p.Text).SingleLine()
			ui.Icon(c, icon("chevron-down")).FontSize(10).TextColor(p.TextFaint)
		})
	}).PopoverMenu(build)
}

// flatTextButton draws a flat button showing a label.
func flatTextButton(c *ui.Context, p Palette, label string, w, h float32) ui.Element {
	b := ui.Button(c, label).Width(w).Height(h).Radius(5).FontSize(10.5).
		Background(ui.Transparent).Border(0, ui.Transparent).TextColor(p.TextMuted)
	if b.Hovered() {
		b.Background(p.SurfaceHi)
	}
	return b
}
