// Pulse is a file manager built with MyGo's pure-Go native UI: tabs, panes,
// details, tiles, icons, lists, a staging tray, tags, folder comparison and
// settings, without a web view.
package main

import (
	_ "embed"
	"log"
	"os"
	"time"

	"pulse/app"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

//go:embed resources/icon.png
var iconPNG []byte

type shell struct {
	win      *mygo.Window
	ui       *app.App
	quitting bool
}

func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		os.Exit(0)
	}
	mygo.App.SetName("Pulse")

	s := &shell{}
	mygo.App.OnSecondInstance(func(args []string, workingDir string) {
		if s.win != nil && !s.win.IsDestroyed() {
			s.win.Show()
			s.win.Focus()
		}
	})
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows && s.win != nil && !s.win.IsDestroyed() {
			s.win.Show()
		}
	})
	mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
		s.quitting = true
		if s.ui != nil {
			s.ui.SaveSession()
		}
	})

	mygo.App.WhenReady(func() {
		s.win = mygo.NewWindow(mygo.WindowOptions{
			Title:             "Pulse",
			Width:             1280,
			Height:            820,
			MinWidth:          920,
			MinHeight:         600,
			StateKey:          "pulse-main",
			Hidden:            true,
			AutoHideMenuBar:   true,
			TitleBarStyle:     mygo.TitleBarHidden,
			HideWindowButtons: true,
			BackgroundColor:   "light-dark(#eef0f5, #15151a)",
			Content: ui.View(func(c *ui.Context) {
				if s.ui != nil {
					s.ui.View(c)
				}
			}),
		})
		_ = s.win.SetIcon(iconPNG)
		s.ui = app.New(s.win)
		s.ui.SetIcon(iconPNG)
		s.ui.SetupMenu()
		if !s.ui.StartHidden() || !s.ui.HasTray() {
			s.win.Show()
		}
		if err := mygo.GlobalShortcut.Register("Alt+Space", func() {
			s.ui.ShowSearch()
		}); err != nil {
			log.Println("global search shortcut:", err)
		}
		if ms := os.Getenv("PULSE_QUIT_AFTER"); ms != "" {
			go func() {
				time.Sleep(2 * time.Second)
				mygo.App.Quit()
			}()
		}
		s.win.OnClose(func(e *mygo.CloseEvent) {
			if s.quitting {
				return
			}
			e.PreventDefault()
			if s.ui.HasTray() {
				s.ui.SaveSession()
				s.ui.HideWindow()
				return
			}
			mygo.App.Quit()
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
