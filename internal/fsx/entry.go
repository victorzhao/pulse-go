// Package fsx reads directories, detects file kinds, computes folder sizes
// and searches names, all without blocking the UI thread.
package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry is one item of a directory.
type Entry struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
	Hidden  bool
	Ext     string
	Kind    Kind
	Link    bool
}

// Kind is a broad category of file, used for icons and type names.
type Kind int

const (
	KindFolder Kind = iota
	KindText
	KindCode
	KindImage
	KindVideo
	KindAudio
	KindArchive
	KindDocument
	KindPDF
	KindSheet
	KindSlides
	KindExecutable
	KindFont
	KindDisk
	KindOther
)

// ReadDir lists a directory, with each entry's metadata.
func ReadDir(dir string) ([]Entry, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(items))
	for _, item := range items {
		name := item.Name()
		if name == "." || name == ".." {
			continue
		}
		path := filepath.Join(dir, name)
		e := Entry{Name: name, Path: path, IsDir: item.IsDir(), Hidden: IsHiddenPath(name, path)}
		if info, err := item.Info(); err == nil {
			e.Size = info.Size()
			e.ModTime = info.ModTime()
			e.Mode = info.Mode()
			e.Link = info.Mode()&os.ModeSymlink != 0
			if e.IsDir {
				e.Size = 0
			}
		}
		if !e.IsDir {
			e.Ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
			e.Kind = KindOf(e.Ext)
		} else {
			e.Kind = KindFolder
		}
		out = append(out, e)
	}
	return out, nil
}

// Stat fills the metadata of one path.
func Stat(path string) (Entry, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Entry{}, err
	}
	name := filepath.Base(path)
	e := Entry{
		Name:    name,
		Path:    path,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Mode:    info.Mode(),
		Hidden:  IsHiddenPath(name, path),
		Link:    info.Mode()&os.ModeSymlink != 0,
	}
	if e.IsDir {
		e.Size = 0
		e.Kind = KindFolder
	} else {
		e.Ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
		e.Kind = KindOf(e.Ext)
	}
	return e, nil
}

// IsHidden reports whether a name starts with a dot or a lock prefix.
func IsHidden(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$")
}

// IsHiddenPath reports whether a path is hidden by its name or by the
// platform's attributes.
func IsHiddenPath(name, path string) bool {
	return IsHidden(name) || osHidden(path)
}

var kindByExt = map[string]Kind{}

func init() {
	add := func(k Kind, exts ...string) {
		for _, e := range exts {
			kindByExt[e] = k
		}
	}
	add(KindText, "txt", "log", "ini", "cfg", "conf", "nfo", "readme", "license", "md", "markdown", "rst")
	add(KindCode,
		"go", "c", "h", "cpp", "cc", "cxx", "hpp", "cs", "java", "kt", "kts", "py", "rb", "php", "js", "mjs", "cjs", "ts", "tsx", "jsx",
		"rs", "swift", "m", "mm", "scala", "lua", "pl", "sh", "bash", "zsh", "fish", "ps1", "bat", "cmd", "sql", "html", "htm", "css", "scss", "less",
		"xml", "json", "jsonc", "yaml", "yml", "toml", "vue", "svelte", "dart", "r", "asm", "s", "v", "zig", "nim", "ex", "exs", "erl", "hs", "ml", "clj",
		"gradle", "cmake", "mk", "dockerfile", "gitignore", "editorconfig", "env")
	add(KindImage, "png", "jpg", "jpeg", "jpe", "gif", "bmp", "webp", "svg", "ico", "tif", "tiff", "heic", "heif", "avif", "psd", "ai", "raw", "cr2", "nef", "arw", "dng")
	add(KindVideo, "mp4", "m4v", "mov", "avi", "mkv", "webm", "wmv", "flv", "mpg", "mpeg", "ts", "m2ts", "3gp", "ogv", "rmvb", "rm", "asf", "vob")
	add(KindAudio, "mp3", "wav", "flac", "aac", "m4a", "ogg", "oga", "opus", "wma", "ape", "wv", "tta", "dsf", "dff", "ac3", "dts", "amr", "mid", "midi", "aiff", "aif", "mka")
	add(KindArchive, "zip", "rar", "7z", "tar", "gz", "bz2", "xz", "zst", "lz", "lzma", "cab", "iso", "dmg", "tgz", "tbz", "txz", "jar", "war", "apk", "ipa", "deb", "rpm", "msi", "pkg", "xar", "cpio", "arj", "lha", "z")
	add(KindDocument, "doc", "docx", "rtf", "odt", "pages", "wps", "epub", "mobi", "azw3", "tex")
	add(KindPDF, "pdf")
	add(KindSheet, "xls", "xlsx", "xlsm", "csv", "tsv", "ods", "numbers", "et")
	add(KindSlides, "ppt", "pptx", "odp", "key", "dps")
	add(KindExecutable, "exe", "msi", "app", "com", "dll", "so", "dylib", "bin", "elf", "scr", "pif", "cpl", "sys", "drv", "deb", "rpm")
	add(KindFont, "ttf", "otf", "ttc", "woff", "woff2", "eot", "fon", "fnt")
}

// KindOf maps an extension to a kind.
func KindOf(ext string) Kind {
	if k, ok := kindByExt[ext]; ok {
		return k
	}
	return KindOther
}

// KindKey returns a stable key for grouping by kind.
func (k Kind) Key() string {
	switch k {
	case KindFolder:
		return "folder"
	case KindText:
		return "text"
	case KindCode:
		return "code"
	case KindImage:
		return "image"
	case KindVideo:
		return "video"
	case KindAudio:
		return "audio"
	case KindArchive:
		return "archive"
	case KindDocument:
		return "document"
	case KindPDF:
		return "pdf"
	case KindSheet:
		return "sheet"
	case KindSlides:
		return "slides"
	case KindExecutable:
		return "executable"
	case KindFont:
		return "font"
	case KindDisk:
		return "disk"
	}
	return "other"
}

// ExtGroup is the class of an extension for coloring its label.
func ExtGroup(ext string) string {
	return KindOf(ext).Key()
}

// FormatSize writes a byte count the way Pulse does: 183 B, 2.7 KB, 227.0 KB,
// 3.5 TB.
func FormatSize(n int64) string {
	const unit = 1024
	if n < unit {
		return itoa(n) + " B"
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	v := float64(n) / float64(div)
	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	return ftoa(v, 1) + " " + units[exp]
}

func itoa(n int64) string {
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

func ftoa(v float64, digits int) string {
	scale := 1.0
	for range digits {
		scale *= 10
	}
	scaled := int64(v*scale + 0.5)
	whole := scaled / int64(scale)
	frac := scaled % int64(scale)
	if digits == 0 {
		return itoa(whole)
	}
	fs := itoa(frac)
	for len(fs) < digits {
		fs = "0" + fs
	}
	return itoa(whole) + "." + fs
}
