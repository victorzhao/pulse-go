package app

import (
	"pulse/internal/fsx"

	"github.com/egoist/mygo/ui"
)

func mono(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#000000" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

func color(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">` + body + `</svg>`))
}

var uiIcons = map[string]*ui.SVG{
	"logo":          mono(`<circle cx="12" cy="12" r="9"/><path d="M4 13h3l2-5 3 9 2-4h6"/>`),
	"back":          mono(`<path d="M19 12H5"/><path d="m12 19-7-7 7-7"/>`),
	"forward":       mono(`<path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>`),
	"up":            mono(`<path d="M12 19V5"/><path d="m5 12 7-7 7 7"/>`),
	"refresh":       mono(`<path d="M21 12a9 9 0 1 1-2.6-6.4"/><path d="M21 3v6h-6"/>`),
	"search":        mono(`<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>`),
	"settings":      mono(`<path d="M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.9.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.9.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.9-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.9V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1Z"/>`),
	"info":          mono(`<circle cx="12" cy="12" r="9"/><path d="M12 16v-5"/><path d="M12 8h.01"/>`),
	"minimize":      mono(`<path d="M5 12h14"/>`),
	"maximize":      mono(`<rect x="5" y="5" width="14" height="14" rx="1"/>`),
	"restore":       mono(`<rect x="7" y="7" width="12" height="12" rx="1"/><path d="M5 15V5h10"/>`),
	"close":         mono(`<path d="M6 6l12 12M18 6 6 18"/>`),
	"plus":          mono(`<path d="M12 5v14M5 12h14"/>`),
	"minus":         mono(`<path d="M5 12h14"/>`),
	"x":             mono(`<path d="M6 6l12 12M18 6 6 18"/>`),
	"chevron-down":  mono(`<path d="m6 9 6 6 6-6"/>`),
	"chevron-right": mono(`<path d="m9 6 6 6-6 6"/>`),
	"chevron-up":    mono(`<path d="m6 15 6-6 6 6"/>`),
	"check":         mono(`<path d="m5 12 4 4L19 6"/>`),
	"star":          mono(`<path d="m12 3 2.7 5.6 6.1.9-4.4 4.3 1 6.1-5.4-2.9-5.4 2.9 1-6.1L3.2 9.5l6.1-.9z"/>`),
	"star-filled":   color(`<path fill="#f5b400" d="m12 3 2.7 5.6 6.1.9-4.4 4.3 1 6.1-5.4-2.9-5.4 2.9 1-6.1L3.2 9.5l6.1-.9z"/>`),
	"clock":         mono(`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`),
	"desktop":       mono(`<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/>`),
	"download":      mono(`<path d="M12 3v12"/><path d="m7 10 5 5 5-5"/><path d="M4 19h16"/>`),
	"home":          mono(`<path d="m3 10 9-7 9 7"/><path d="M5 9v11h14V9"/>`),
	"picture":       mono(`<rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="8.5" cy="9.5" r="1.5"/><path d="m21 15-5-5L5 20"/>`),
	"music":         mono(`<path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>`),
	"video":         mono(`<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m10 9 5 3-5 3z"/>`),
	"recycle":       mono(`<path d="M4 7h16"/><path d="M9 7V4h6v3"/><path d="M6 7l1 13h10l1-13"/><path d="M10 11v5M14 11v5"/>`),
	"tag":           mono(`<path d="M3 12V5a2 2 0 0 1 2-2h7l9 9-9 9z"/><circle cx="8" cy="8" r="1.5"/>`),
	"disk":          mono(`<rect x="3" y="5" width="18" height="14" rx="2"/><circle cx="7" cy="12" r="1"/><path d="M11 10h6M11 14h4"/>`),
	"network":       mono(`<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 3 2.5 15 0 18M12 3c-2.5 3-2.5 15 0 18"/>`),
	"cloud":         mono(`<path d="M6 18a4 4 0 0 1 .5-8A6 6 0 0 1 18 9a4 4 0 0 1 0 9z"/>`),
	"lock":          mono(`<rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>`),
	"new-folder":    mono(`<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><path d="M12 11v6M9 14h6"/>`),
	"cut":           mono(`<circle cx="6" cy="6" r="2.5"/><circle cx="6" cy="18" r="2.5"/><path d="M8 7.5 20 18M8 16.5 20 6"/>`),
	"copy":          mono(`<rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3"/>`),
	"paste":         mono(`<rect x="5" y="5" width="14" height="16" rx="2"/><path d="M9 5V3h6v2"/><path d="M9 11h6M9 15h4"/>`),
	"rename":        mono(`<path d="M4 20h4l11-11a2.8 2.8 0 0 0-4-4L4 16z"/><path d="m13 6 5 5"/>`),
	"delete":        mono(`<path d="M4 7h16"/><path d="M9 7V4h6v3"/><path d="M6 7l1 13h10l1-13"/>`),
	"sort":          mono(`<path d="M4 7h10M4 12h7M4 17h4"/><path d="M18 6v12M15 15l3 3 3-3"/>`),
	"filter":        mono(`<path d="M3 5h18l-7 8v6l-4 2v-8z"/>`),
	"group":         mono(`<rect x="3" y="4" width="8" height="7" rx="1"/><rect x="13" y="4" width="8" height="7" rx="1"/><rect x="3" y="13" width="8" height="7" rx="1"/><rect x="13" y="13" width="8" height="7" rx="1"/>`),
	"list":          mono(`<path d="M8 6h13M8 12h13M8 18h13"/><path d="M3.5 6h.01M3.5 12h.01M3.5 18h.01"/>`),
	"tiles":         mono(`<rect x="3" y="3" width="8" height="8" rx="1"/><rect x="13" y="3" width="8" height="8" rx="1"/><rect x="3" y="13" width="8" height="8" rx="1"/><rect x="13" y="13" width="8" height="8" rx="1"/>`),
	"icons":         mono(`<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>`),
	"details":       mono(`<path d="M3 5h18M3 12h18M3 19h18"/>`),
	"content":       mono(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 9h18M8 9v11"/>`),
	"sidebar":       mono(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16"/>`),
	"details-pane":  mono(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M15 4v16"/>`),
	"external":      mono(`<path d="M14 4h6v6"/><path d="m20 4-9 9"/><path d="M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5"/>`),
	"terminal":      mono(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="m7 9 3 3-3 3M13 15h4"/>`),
	"more":          mono(`<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>`),
	"compare":       mono(`<path d="M8 4v16M4 8l4-4 4 4M16 20V4M12 16l4 4 4-4"/>`),
	"diff":          mono(`<path d="M4 5h7M4 12h7M4 19h7"/><path d="M14 5h6M14 12h6M14 19h6"/>`),
	"clear":         mono(`<path d="M4 7h16M9 7V4h6v3"/><path d="M6 7l1 13h10l1-13"/>`),
	"eye":           mono(`<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/>`),
	"eye-off":       mono(`<path d="M3 3l18 18"/><path d="M10.6 6.2A10 10 0 0 1 22 12s-1 2-3 3.6M6.5 7.5C4 9.2 2 12 2 12s3.5 7 10 7a10 10 0 0 0 4.3-1"/>`),
	"pin":           mono(`<path d="M12 17v5"/><path d="M8 3h8l-1 6 3 3v2H6v-2l3-3z"/>`),
	"properties":    mono(`<path d="M4 6h16M4 12h16M4 18h10"/><circle cx="18" cy="18" r="2"/>`),
	"select-all":    mono(`<rect x="4" y="4" width="16" height="16" rx="2"/><path d="m8 12 3 3 5-6"/>`),
	"sun":           mono(`<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>`),
	"moon":          mono(`<path d="M20 14.5A8.5 8.5 0 1 1 9.5 4a7 7 0 0 0 10.5 10.5z"/>`),
	"monitor":       mono(`<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/>`),
	"palette":       mono(`<path d="M12 21a9 9 0 1 1 0-18c5 0 9 3 9 7 0 3-2 4-4 4h-1a2 2 0 0 0-1 3.5c.5.6.3 1.5-.5 2a5 5 0 0 1-2.5 1.5Z"/><circle cx="8" cy="10" r="1"/><circle cx="12" cy="7" r="1"/><circle cx="16" cy="10" r="1"/>`),
	"layout-single": mono(`<rect x="3" y="4" width="18" height="16" rx="2"/>`),
	"layout-h":      mono(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M12 4v16"/>`),
	"layout-v":      mono(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 12h18"/>`),
	"layout-triple": mono(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16M15 4v16"/>`),
	"layout-quad":   mono(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M12 4v16M3 12h18"/>`),
	"folder":        mono(`<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>`),
	"folder-open":   mono(`<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v2H3z"/><path d="M3 9h18l-2 9H5z"/>`),
	"file":          mono(`<path d="M6 3h8l4 4v14H6z"/><path d="M14 3v4h4"/>`),
	"computer":      mono(`<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/>`),
	"recent":        mono(`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`),
	"keyboard":      mono(`<rect x="2" y="6" width="20" height="12" rx="2"/><path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M7 14h10"/>`),
	"globe":         mono(`<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 3 2.5 15 0 18M12 3c-2.5 3-2.5 15 0 18"/>`),
	"warning":       mono(`<path d="M12 3 2 20h20z"/><path d="M12 9v5M12 17h.01"/>`),
	"save":          mono(`<path d="M5 3h12l4 4v14H5z"/><path d="M8 3v6h8V3M8 21v-7h8v7"/>`),
	"undo":          mono(`<path d="M4 9h10a5 5 0 0 1 0 10H8"/><path d="m8 5-4 4 4 4"/>`),
	"redo":          mono(`<path d="M20 9H10a5 5 0 0 0 0 10h6"/><path d="m16 5 4 4-4 4"/>`),
	"empty":         mono(`<path d="M4 7h16v13H4z"/><path d="M4 7l3-3h10l3 3"/><path d="M8 12h8"/>`),
	"drive":         mono(`<rect x="3" y="6" width="18" height="12" rx="2"/><circle cx="7" cy="12" r="1"/>`),
	"link":          mono(`<path d="M10 13a5 5 0 0 0 7 0l2-2a5 5 0 0 0-7-7l-1 1"/><path d="M14 11a5 5 0 0 0-7 0l-2 2a5 5 0 0 0 7 7l1-1"/>`),
	"shield":        mono(`<path d="M12 3 4 6v6c0 5 3.5 8 8 9 4.5-1 8-4 8-9V6z"/>`),
}

var quickIcons = map[string]*ui.SVG{
	"home":      color(`<path fill="#4a90e2" d="m3 10 9-7 9 7v10a1 1 0 0 1-1 1h-5v-7H9v7H4a1 1 0 0 1-1-1z"/>`),
	"desktop":   color(`<rect x="3" y="4" width="18" height="12" rx="2" fill="#4a90e2"/><path fill="#4a90e2" d="M8 20h8v1H8z"/><path fill="#dbe9fb" d="M12 16h0"/>`),
	"documents": color(`<path fill="#f4c64a" d="M3 6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>`),
	"downloads": color(`<path fill="#4caf50" d="M12 3v10m0 0 4-4m-4 4-4-4"/><rect x="3" y="16" width="18" height="4" rx="1" fill="#4caf50"/>`),
	"pictures":  color(`<rect x="3" y="4" width="18" height="16" rx="2" fill="#4caf50"/><circle cx="8.5" cy="9.5" r="1.8" fill="#ffe082"/><path fill="#ffffff" d="m4 19 6-6 4 4 3-3 3 3v2z"/>`),
	"music":     color(`<path fill="#c2477c" d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3" fill="#c2477c"/><circle cx="18" cy="16" r="3" fill="#c2477c"/>`),
	"videos":    color(`<rect x="3" y="5" width="18" height="14" rx="2" fill="#7b52c7"/><path fill="#ffffff" d="m10 9 5 3-5 3z"/>`),
	"recycle":   color(`<path fill="#9aa4b2" d="M4 7h16l-1 13H5z"/><rect x="9" y="4" width="6" height="3" rx="1" fill="#9aa4b2"/>`),
	"clock":     color(`<circle cx="12" cy="12" r="9" fill="#9aa4b2"/><path stroke="#ffffff" stroke-width="2" fill="none" d="M12 7v5l3 2"/>`),
	"computer":  color(`<rect x="3" y="4" width="18" height="12" rx="2" fill="#4a90e2"/><rect x="5" y="6" width="14" height="8" rx="1" fill="#dbe9fb"/><path fill="#4a90e2" d="M8 20h8v1H8z"/>`),
}

var fileIcons = map[fsx.Kind]*ui.SVG{
	fsx.KindFolder:     color(`<path fill="#d9a521" d="M2.5 6.8A2 2 0 0 1 4.5 5h4.2l1.9 2h8.9a2 2 0 0 1 2 2v9.2a2 2 0 0 1-2 2H4.5a2 2 0 0 1-2-2z"/><path fill="#f4c64a" d="M2.5 9.5h19v8.7a2 2 0 0 1-2 2H4.5a2 2 0 0 1-2-2z"/>`),
	fsx.KindText:       color(`<path fill="#ffffff" stroke="#c9ccd4" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#e6e8ee" d="M14 2.5 18.5 7H14z"/><path stroke="#8d919c" stroke-width="1.4" d="M8.5 11h7M8.5 14h7M8.5 17h4"/>`),
	fsx.KindCode:       color(`<path fill="#eaf2ff" stroke="#a9c4e8" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#c7dcf7" d="M14 2.5 18.5 7H14z"/><path fill="none" stroke="#2f6fbf" stroke-width="1.6" d="m9.5 12-2 2 2 2M14.5 12l2 2-2 2"/>`),
	fsx.KindImage:      color(`<path fill="#eaf7ea" stroke="#a6d2a6" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#cbe9cb" d="M14 2.5 18.5 7H14z"/><circle cx="9.5" cy="11" r="1.3" fill="#f0b429"/><path fill="#4caf50" d="m7 17 4-4 3 3 2-2 3 3v1H7z"/>`),
	fsx.KindVideo:      color(`<path fill="#f2ecfb" stroke="#bda9e0" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#ddcdf5" d="M14 2.5 18.5 7H14z"/><path fill="#7b52c7" d="m9.5 11 6 3.5-6 3.5z"/>`),
	fsx.KindAudio:      color(`<path fill="#fdf0f5" stroke="#e0aac0" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#f3cddc" d="M14 2.5 18.5 7H14z"/><path fill="none" stroke="#c2477c" stroke-width="1.5" d="M10 17v-5l4-1v5"/><circle cx="8.8" cy="17.2" r="1.4" fill="#c2477c"/><circle cx="12.8" cy="16.2" r="1.4" fill="#c2477c"/>`),
	fsx.KindArchive:    color(`<path fill="#f8efe0" stroke="#c9a97a" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#ecd9b8" d="M14 2.5 18.5 7H14z"/><path fill="none" stroke="#9a6b2f" stroke-width="1.4" d="M11 4v3M11 9v3M11 15v3"/><rect x="10" y="17" width="2" height="2" fill="#9a6b2f"/>`),
	fsx.KindDocument:   color(`<path fill="#eef4fd" stroke="#a9c0e0" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#cfdff5" d="M14 2.5 18.5 7H14z"/><path stroke="#4a7bbd" stroke-width="1.4" d="M8.5 10h7M8.5 13h7M8.5 16h5"/>`),
	fsx.KindPDF:        color(`<path fill="#fdeeee" stroke="#d99a9a" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#f5cccc" d="M14 2.5 18.5 7H14z"/><rect x="7.5" y="12" width="9" height="5" rx="1" fill="#d13438"/><path stroke="#ffffff" stroke-width="1.1" d="M9 14.5h6"/>`),
	fsx.KindSheet:      color(`<path fill="#eff9f0" stroke="#a6d2a6" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#d2ecd2" d="M14 2.5 18.5 7H14z"/><path stroke="#3f8f45" stroke-width="1.3" d="M8 11h8M8 14h8M8 17h8M11.5 10v8M15 10v8"/>`),
	fsx.KindSlides:     color(`<path fill="#fff4ea" stroke="#e0b48a" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#f7ddc4" d="M14 2.5 18.5 7H14z"/><rect x="8" y="10" width="8" height="6" rx="1" fill="#e07a2f"/><path stroke="#ffffff" stroke-width="1.2" d="M10 14v-2M12 14v-3M14 14v-1"/>`),
	fsx.KindExecutable: color(`<path fill="#eaf3ff" stroke="#9fbde0" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#cfe2f8" d="M14 2.5 18.5 7H14z"/><rect x="8" y="11" width="8" height="6" rx="1" fill="#2b6fb3"/><path fill="#ffffff" d="m11 12.5 3 1.5-3 1.5z"/>`),
	fsx.KindFont:       color(`<path fill="#f3f1fa" stroke="#b6afe0" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#ded9f3" d="M14 2.5 18.5 7H14z"/><path fill="#5b4bb0" d="m8 18 4-9 4 9h-2l-1-2.5h-2L10 18zm3.2-4.2h1.6L12 11z"/>`),
	fsx.KindDisk:       color(`<path fill="#e9edf2" stroke="#aab3bf" d="M3 7.5h18v9H3z"/><rect x="4.5" y="9" width="15" height="6" rx="1" fill="#b8c2cf"/><circle cx="7" cy="12" r="1" fill="#5a6472"/><path stroke="#5a6472" stroke-width="1" d="M11 11.5h5M11 13.5h3"/>`),
	fsx.KindOther:      color(`<path fill="#f6f7f9" stroke="#c9ccd4" d="M6 2.5h8l4.5 4.5v14.5H6z"/><path fill="#e6e8ee" d="M14 2.5 18.5 7H14z"/>`),
}

// icon returns a monochrome UI icon by name.
func icon(name string) *ui.SVG {
	if s, ok := uiIcons[name]; ok {
		return s
	}
	return uiIcons["file"]
}

// kindIcon returns the colored icon of a file kind.
func kindIcon(kind fsx.Kind) *ui.SVG {
	if s, ok := fileIcons[kind]; ok {
		return s
	}
	return fileIcons[fsx.KindOther]
}
