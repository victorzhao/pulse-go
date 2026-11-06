package app

import (
	"image"
	"os"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// Palette is Pulse's resolved look for the current appearance settings.
type Palette struct {
	Dark bool

	BG          ui.Color
	BGTop       ui.Color
	BGBottom    ui.Color
	Panel       ui.Color
	PanelSolid  ui.Color
	Surface     ui.Color
	SurfaceHi   ui.Color
	SurfaceDown ui.Color
	Border      ui.Color
	BorderSoft  ui.Color
	Text        ui.Color
	TextMuted   ui.Color
	TextFaint   ui.Color
	Accent      ui.Color
	AccentHi    ui.Color
	AccentDown  ui.Color
	AccentText  ui.Color
	Danger      ui.Color
	Warning     ui.Color
	Success     ui.Color
	Selection   ui.Color
	Focus       ui.Color
	Scrollbar   ui.Color
	TagRed      ui.Color
	TagOrange   ui.Color
	TagGreen    ui.Color
	TagYellow   ui.Color
	TagPurple   ui.Color
	TagBlue     ui.Color
	TagGray     ui.Color

	RowH    float32
	Radius  float32
	Font    float32
	Spacing float32
}

// alphaByte returns the color with its alpha multiplied by a.
func fade(c ui.Color, a float32) ui.Color {
	c.A = uint8(float32(c.A) * a)
	return c
}

// palette resolves the configured appearance. Transparency and blur only
// matter when there is something behind the panels.
func (a *App) palette() Palette {
	app := a.cfg.Appearance
	dark := app.Theme == "dark"
	if app.Theme == "system" {
		dark = a.systemDark
	}
	accent := hexColor(app.Accent)
	p := Palette{Dark: dark}
	if dark {
		p.BG = ui.Hex("#15151a")
		p.BGTop = ui.Hex("#23252e")
		p.BGBottom = ui.Hex("#101014")
		p.Panel = ui.RGBA(30, 31, 38, 1)
		p.PanelSolid = ui.Hex("#1c1d24")
		p.Surface = ui.RGBA(255, 255, 255, 0.06)
		p.SurfaceHi = ui.RGBA(255, 255, 255, 0.10)
		p.SurfaceDown = ui.RGBA(255, 255, 255, 0.14)
		p.Border = ui.RGBA(255, 255, 255, 0.12)
		p.BorderSoft = ui.RGBA(255, 255, 255, 0.07)
		p.Text = ui.Hex("#f2f3f7")
		p.TextMuted = ui.Hex("#a0a2ad")
		p.TextFaint = ui.Hex("#74767f")
		p.Scrollbar = ui.RGBA(255, 255, 255, 0.22)
	} else {
		p.BG = ui.Hex("#eef0f5")
		p.BGTop = ui.Hex("#f8f9fc")
		p.BGBottom = ui.Hex("#e3e6ee")
		p.Panel = ui.RGBA(255, 255, 255, 1)
		p.PanelSolid = ui.Hex("#ffffff")
		p.Surface = ui.RGBA(0, 0, 0, 0.045)
		p.SurfaceHi = ui.RGBA(0, 0, 0, 0.075)
		p.SurfaceDown = ui.RGBA(0, 0, 0, 0.11)
		p.Border = ui.RGBA(0, 0, 0, 0.11)
		p.BorderSoft = ui.RGBA(0, 0, 0, 0.06)
		p.Text = ui.Hex("#1c1d24")
		p.TextMuted = ui.Hex("#6c6e7a")
		p.TextFaint = ui.Hex("#9a9ca6")
		p.Scrollbar = ui.RGBA(0, 0, 0, 0.22)
	}
	// Let the background show through the panels as configured.
	t := float32(a.cfg.Appearance.Transparency) / 100
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	if t > 0 {
		panel := p.Panel
		panel.A = uint8(float32(panel.A) * (1 - t*0.75))
		p.Panel = panel
		surf := p.Surface
		surf.A = uint8(float32(surf.A) * (1 - t*0.2))
		p.Surface = surf
	}

	p.Accent = accent
	p.AccentHi = accent.Mix(ui.RGBA(255, 255, 255, 0.35), 0.18)
	p.AccentDown = accent.Mix(ui.RGBA(0, 0, 0, 0.35), 0.18)
	if luminance(accent) < 0.55 {
		p.AccentText = ui.Hex("#ffffff")
	} else {
		p.AccentText = ui.Hex("#101014")
	}
	p.Danger = ui.Hex("#e5484d")
	p.Warning = ui.Hex("#f5a623")
	p.Success = ui.Hex("#3fb950")
	p.Selection = fade(accent, 0.32)
	p.Focus = accent
	p.TagRed = ui.Hex("#e81123")
	p.TagOrange = ui.Hex("#f7630c")
	p.TagGreen = ui.Hex("#107c10")
	p.TagYellow = ui.Hex("#ffb900")
	p.TagPurple = ui.Hex("#8764b8")
	p.TagBlue = ui.Hex("#0078d4")
	p.TagGray = ui.Hex("#767676")

	switch a.cfg.Appearance.RowHeight {
	case "compact":
		p.RowH = 26
	case "roomy":
		p.RowH = 40
	default:
		p.RowH = 32
	}
	p.Radius = 7
	p.Font = 13
	p.Spacing = 4
	return p
}

func luminance(c ui.Color) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
}

// theme turns the palette into the ui theme the widgets use.
func (p Palette) theme() *ui.Theme {
	t := &ui.Theme{
		Dark:           p.Dark,
		Background:     p.PanelSolid,
		Surface:        p.Surface,
		SurfaceHover:   p.SurfaceHi,
		SurfacePressed: p.SurfaceDown,
		Border:         p.Border,
		Text:           p.Text,
		TextMuted:      p.TextMuted,
		Accent:         p.Accent,
		AccentHover:    p.AccentHi,
		AccentPressed:  p.AccentDown,
		AccentText:     p.AccentText,
		Danger:         p.Danger,
		Warning:        p.Warning,
		Success:        p.Success,
		Selection:      p.Selection,
		Focus:          p.Focus,
		Scrollbar:      p.Scrollbar,
		ScrollbarWidth: 8,
		Radius:         p.Radius,
		Spacing:        p.Spacing,
		FontSize:       p.Font,
	}
	t.Inverse = p.Text
	t.InverseText = p.BG
	return t
}

// tagColor maps a tag id to its color.
func (a *App) tagColor(id string) ui.Color {
	for _, tag := range a.cfg.Tags {
		if tag.ID == id {
			return hexColor(tag.Color)
		}
	}
	return ui.Transparent
}

// wallpaper is the decoded background picture, if any.
type wallpaper struct {
	mu   sync.Mutex
	bmp  *ui.Bitmap
	path string
	// tried marks a path that was loaded or failed, so it is not retried
	// every frame.
	tried bool
}

// loadWallpaper decodes the configured picture on a goroutine and hands it
// to the window.
func (a *App) loadWallpaper() {
	path := a.cfg.Appearance.Wallpaper
	if path == "" || a.wp.tried && a.wp.path == path {
		return
	}
	a.wp.tried = true
	a.wp.path = path
	go func() {
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		a.Win.Update(func() {
			if bmp, err := ui.DecodeBitmap(data); err == nil {
				a.wp.bmp = bmp
			}
		})
	}()
}

// paintBackground paints the window's gradient backdrop.
func (a *App) paintBackground(painter *ui.Painter, r ui.Rect, p Palette) {
	painter.FillGradient(r, ui.LinearGradient{From: p.BGTop, To: p.BGBottom, Angle: 115}, 0)
	painter.FillRadialGradient(r, ui.RadialGradient{
		Center: ui.Point{X: 0.78, Y: 0.12},
		Stops: []ui.GradientStop{
			{Color: fade(p.Accent, 0.16), At: 0},
			{Color: fade(p.Accent, 0), At: 0.7},
		},
	}, 0)
}

// hexColor parses a color, falling back to the default accent.
func hexColor(s string) ui.Color {
	s = strings.TrimSpace(s)
	if len(s) == 7 || len(s) == 4 || len(s) == 9 {
		if strings.HasPrefix(s, "#") {
			return ui.Hex(s)
		}
	}
	return ui.Hex("#0078d4")
}

// wallpaperSize reports the decoded picture's size, for tests.
func (a *App) wallpaperSize() image.Point {
	if a.wp.bmp == nil {
		return image.Point{}
	}
	w, h := a.wp.bmp.Size()
	return image.Point{X: w, Y: h}
}
