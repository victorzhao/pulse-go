package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pulse/internal/fsx"
	"pulse/internal/platform"
	"pulse/internal/store"

	"github.com/egoist/mygo/ui"
)

// stageEntries collects a pane's selected entries into the staging tray.
func (a *App) stageEntries(p *Pane, move bool) {
	if p == nil || p.Virtual == virtualTrash || p.Virtual == virtualComputer {
		return
	}
	entries := a.selectedEntries(p)
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	a.stagePaths(paths, move)
}

// stagePaths collects paths into the staging tray.
func (a *App) stagePaths(paths []string, move bool) {
	if len(paths) == 0 {
		return
	}
	a.addStash(paths, move)
	a.toast = a.tf("stashAdd", len(paths))
}

// addStash adds paths to the staging tray, replacing duplicates.
func (a *App) addStash(paths []string, move bool) {
	for _, path := range paths {
		if path == "" {
			continue
		}
		exists := false
		for i, item := range a.stash.Items {
			if item.Path == path {
				a.stash.Items[i].Move = move
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		a.stash.Items = append(a.stash.Items, stashItem{
			Path:  path,
			Name:  filepath.Base(path),
			IsDir: info.IsDir(),
			Size:  info.Size(),
			Move:  move,
			At:    time.Now(),
		})
	}
}

// clearStash empties the staging tray.
func (a *App) clearStash() {
	a.stash.Items = nil
	a.toast = a.t("stashCleared")
}

// pasteStash copies or moves the staged items into the active pane.
func (a *App) pasteStash() {
	pane := a.activePane()
	if pane == nil {
		return
	}
	if pane.Virtual != virtualNone {
		a.toast = a.t("cannotOpen")
		return
	}
	if len(a.stash.Items) == 0 {
		a.toast = a.t("stashEmptyWarn")
		return
	}
	var copies, moves []string
	for _, item := range a.stash.Items {
		if item.Move {
			moves = append(moves, item.Path)
		} else {
			copies = append(copies, item.Path)
		}
	}
	n := len(a.stash.Items)
	a.stash.Items = nil
	if len(copies) > 0 {
		a.startTransfer(copies, pane.Path, false)
	}
	if len(moves) > 0 {
		a.startTransfer(moves, pane.Path, true)
	}
	a.toast = a.tf("stashPasted", n)
}

// transferJob is a copy or move waiting for the previous one to finish.
type transferJob struct {
	srcs   []string
	dstDir string
	move   bool
}

// startTransfer copies or moves paths into a folder, asking about conflicts
// first. Only one transfer runs at a time; the others wait in a queue.
func (a *App) startTransfer(srcs []string, dstDir string, move bool) {
	if a.Win == nil {
		return
	}
	if a.ops.Running {
		a.pending = append(a.pending, transferJob{srcs: srcs, dstDir: dstDir, move: move})
		return
	}
	clean := make([]string, 0, len(srcs))
	seen := map[string]bool{}
	for _, src := range srcs {
		if src == "" || seen[src] {
			continue
		}
		seen[src] = true
		clean = append(clean, src)
	}
	if len(clean) == 0 {
		return
	}
	items := make([]conflictItem, 0, len(clean))
	for _, src := range clean {
		dst := filepath.Join(dstDir, filepath.Base(src))
		if filepath.Clean(src) == filepath.Clean(dst) {
			if move {
				continue
			}
			dst = uniqueDest(src)
		}
		if platform.Exists(dst) {
			items = append(items, conflictItem{src: src, dst: dst})
		}
	}
	job := transferJob{srcs: clean, dstDir: dstDir, move: move}
	if len(items) > 0 {
		a.beginConflict(job, items, map[string]int{}, 0)
		return
	}
	a.runTransferChoices(job, nil)
}

// conflictItem is one source and the destination that already exists.
type conflictItem struct {
	src, dst string
}

// beginConflict asks about the conflict at index, then the next.
func (a *App) beginConflict(job transferJob, items []conflictItem, choices map[string]int, at int) {
	item := items[at]
	merge := false
	if srcInfo, err := os.Stat(item.src); err == nil && srcInfo.IsDir() {
		if dstInfo, err := os.Stat(item.dst); err == nil && dstInfo.IsDir() {
			merge = true
		}
	}
	a.conflict = conflictState{
		Open:    true,
		Path:    item.dst,
		Dir:     true,
		Src:     item.src,
		Merge:   merge,
		Pending: items,
		Job:     job,
		Choices: choices,
		Index:   at,
	}
	a.conflict.Resume = func(choice int, applyAll bool) {
		if choice < 0 {
			a.toast = a.t("operationCancelled")
			return
		}
		choices[item.src] = choice
		next := at + 1
		if applyAll {
			for i := at + 1; i < len(items); i++ {
				choices[items[i].src] = choice
			}
			next = len(items)
		}
		if next < len(items) {
			a.beginConflict(job, items, choices, next)
			return
		}
		a.runTransferChoices(job, choices)
	}
}

// runTransferChoices runs a job with the choices made for its conflicts.
func (a *App) runTransferChoices(job transferJob, choices map[string]int) {
	a.runTransfer(job.srcs, job.dstDir, job.move, choices)
}

// runTransfer performs the copy or move in the background.
func (a *App) runTransfer(srcs []string, dstDir string, move bool, choices map[string]int) {
	if a.Win == nil {
		return
	}
	a.ops.Gen++
	gen := a.ops.Gen
	ctx, cancel := context.WithCancel(context.Background())
	title := a.t("operationCopying")
	if move {
		title = a.t("operationMoving")
	}
	a.ops = opsState{Running: true, Title: title, Total: len(srcs), Cancel: cancel, Gen: gen}
	go func() {
		type pair struct{ src, dst string }
		var done []pair
		failed := 0
		mergedAny := false
		for i, src := range srcs {
			if ctx.Err() != nil {
				break
			}
			dst := filepath.Join(dstDir, filepath.Base(src))
			if filepath.Clean(src) == filepath.Clean(dst) {
				if move {
					continue
				}
				dst = uniqueDest(src)
			}
			merged := false
			choice := 0
			if choices != nil {
				if c, ok := choices[src]; ok {
					choice = c
				}
			}
			if platform.Exists(dst) {
				switch choice {
				case 1: // skip
					a.Win.Update(func() {
						a.ops.Done = i + 1
						a.ops.Current = filepath.Base(src)
					})
					continue
				case 2: // keep both
					dst = uniqueDest(dst)
				case 3: // merge folders
					srcInfo, e1 := os.Stat(src)
					dstInfo, e2 := os.Stat(dst)
					if e1 == nil && e2 == nil && srcInfo.IsDir() && dstInfo.IsDir() {
						merged = true
					} else if err := os.RemoveAll(dst); err != nil {
						failed++
						continue
					}
				default: // replace
					if err := os.RemoveAll(dst); err != nil {
						failed++
						continue
					}
				}
			}
			var err error
			switch {
			case merged && move:
				err = mergeCopy(src, dst)
				if err == nil {
					err = os.RemoveAll(src)
				}
			case merged:
				err = mergeCopy(src, dst)
			case move:
				err = movePath(src, dst)
			default:
				err = copyPath(src, dst)
			}
			if err != nil {
				failed++
			} else {
				done = append(done, pair{src, dst})
				if merged {
					mergedAny = true
				}
				if move {
					a.Win.Update(func() { a.remapPath(src, dst) })
				}
			}
			a.Win.Update(func() {
				if a.ops.Gen != gen {
					return
				}
				a.ops.Done = i + 1
				a.ops.Current = filepath.Base(src)
			})
		}
		cancelled := ctx.Err() != nil
		a.Win.Update(func() {
			if a.ops.Gen != gen {
				return
			}
			a.ops.Running = false
			if cancelled {
				a.toast = a.t("operationCancelled")
				return
			}
			if failed > 0 {
				a.toast = a.t("operationFailed")
			} else if move {
				a.toast = a.tf("movedItems", len(done))
			} else {
				a.toast = a.tf("copiedItems", len(done))
			}
			a.refreshPanesAt(dstDir)
			if move {
				srcDirs := map[string]bool{}
				for _, src := range srcs {
					srcDirs[filepath.Dir(src)] = true
				}
				for dir := range srcDirs {
					a.refreshPanesAt(dir)
				}
			}
			// Undo puts the items back.
			if len(done) > 0 && !mergedAny {
				label := a.t("undoCopied")
				if move {
					label = a.t("undoMoved")
				}
				undo := undoAction{Label: label}
				if move {
					undo.Run = func() error {
						for _, p := range done {
							if err := movePath(p.dst, p.src); err != nil {
								return err
							}
							a.remapPath(p.dst, p.src)
						}
						return nil
					}
				} else {
					undo.Run = func() error {
						for _, p := range done {
							if err := os.RemoveAll(p.dst); err != nil {
								return err
							}
						}
						return nil
					}
				}
				a.pushUndo(undo)
				a.toastAction = &toastAction{Message: a.toast, Label: a.t("undo"), Run: func() {
					a.runUndo(undo)
				}}
			}
			if len(a.pending) > 0 {
				job := a.pending[0]
				a.pending = a.pending[1:]
				a.startTransfer(job.srcs, job.dstDir, job.move)
			}
		})
	}()
}

// pushUndo keeps the last operations for undo.
func (a *App) pushUndo(u undoAction) {
	a.undo = append(a.undo, u)
	if len(a.undo) > 20 {
		a.undo = a.undo[len(a.undo)-20:]
	}
}

// runUndo runs an undo action and reports it.
func (a *App) runUndo(u undoAction) {
	if u.Run == nil {
		return
	}
	if err := u.Run(); err != nil {
		a.toast = a.t("operationFailed")
		return
	}
	a.toast = a.t("operationDone")
	a.refreshAllPanes()
}

// deleteEntries moves entries to the trash, or deletes them for good.
func (a *App) deleteEntries(p *Pane, entries []fsx.Entry, permanent bool) {
	if len(entries) == 0 {
		return
	}
	if p != nil && (p.Virtual == virtualTrash || p.Virtual == virtualComputer) {
		return
	}
	if permanent {
		a.confirm = confirmState{
			Open: true, Title: a.t("trashConfirm"), Body: a.t("trashConfirmBody"),
			Confirm: a.t("deleteItem"), Danger: true,
			Action: func() { a.doDelete(p, entries, true) },
		}
		return
	}
	if a.cfg.View.ConfirmDelete {
		a.confirm = confirmState{
			Open: true, Title: a.t("confirmDeleteTitle"),
			Body:    a.tf("confirmDeleteBody", len(entries)),
			Confirm: a.t("delete"), Danger: true,
			Action: func() { a.doDelete(p, entries, false) },
		}
		return
	}
	a.doDelete(p, entries, false)
}

// doDelete removes entries, with an undo for those moved to the trash.
func (a *App) doDelete(p *Pane, entries []fsx.Entry, permanent bool) {
	var ids []string
	var origins []string
	for _, e := range entries {
		if permanent {
			if err := os.RemoveAll(e.Path); err != nil {
				a.toast = a.t("deleteFailed")
			} else {
				a.forgetPath(e.Path)
			}
			continue
		}
		item, err := a.trash.Delete(e.Path)
		if err != nil {
			a.toast = a.t("deleteFailed")
			continue
		}
		ids = append(ids, item.ID)
		origins = append(origins, item.Original)
	}
	a.trashAt = time.Time{}
	if len(ids) > 0 {
		a.toast = a.tf("deletedItems", len(ids))
		undo := undoAction{Label: a.t("undoDelete"), Run: func() error {
			for _, id := range ids {
				if err := a.trash.Restore(id); err != nil {
					return err
				}
			}
			return nil
		}}
		a.pushUndo(undo)
		a.toastAction = &toastAction{Message: a.toast, Label: a.t("undo"), Run: func() { a.runUndo(undo) }}
	}
	a.refreshPane(p)
	a.refreshAllPanes()
}

// emptyTrash asks, then permanently empties the recycle bin.
func (a *App) emptyTrash() {
	a.confirm = confirmState{
		Open: true, Title: a.t("emptyTrashConfirm"), Body: a.t("emptyTrashBody"),
		Confirm: a.t("emptyTrash"), Danger: true,
		Action: func() {
			items, _ := a.trash.List()
			if err := a.trash.Empty(); err != nil {
				a.toast = a.t("emptyTrashFailed")
				return
			}
			for _, item := range items {
				a.forgetPath(item.Original)
			}
			a.trashAt = time.Time{}
			a.refreshTrashPanes()
			a.toast = a.t("operationDone")
		},
	}
}

// refreshTrashPanes reloads every recycle bin pane.
func (a *App) refreshTrashPanes() {
	for _, p := range a.allPanes() {
		if p.Virtual == virtualTrash {
			a.loadVirtual(p)
		}
	}
}

// restoreTrashItem puts one recycle bin item back.
func (a *App) restoreTrashItem(id string, p *Pane) {
	if err := a.trash.Restore(id); err != nil {
		a.toast = a.t("restoreFailed")
		return
	}
	a.trashAt = time.Time{}
	a.toast = a.tf("restoredItems", 1)
	a.refreshPane(p)
	a.refreshAllPanes()
}

// beginRename opens the rename dialog for an entry.
func (a *App) beginRename(p *Pane, e fsx.Entry) {
	if p != nil && (p.Virtual == virtualTrash || p.Virtual == virtualComputer) {
		return
	}
	a.rename = renameState{Open: true, Path: e.Path, Name: e.Name, Pane: p}
}

// renameEntry renames a path to a new name typed inline.
func (a *App) renameEntry(p *Pane, path, newName string) {
	newName = strings.TrimSpace(newName)
	if newName == "" || newName == filepath.Base(path) {
		return
	}
	if strings.ContainsAny(newName, `/\`) {
		a.toast = a.t("invalidName")
		a.refreshPane(p)
		return
	}
	dst := filepath.Join(filepath.Dir(path), newName)
	if platform.Exists(dst) {
		a.toast = a.t("nameExists")
		a.refreshPane(p)
		return
	}
	if err := os.Rename(path, dst); err != nil {
		a.toast = a.t("renameFailed")
		a.refreshPane(p)
		return
	}
	a.remapPath(path, dst)
	undo := undoAction{Label: a.t("undoRename"), Run: func() error {
		if err := os.Rename(dst, path); err != nil {
			return err
		}
		a.remapPath(dst, path)
		return nil
	}}
	a.pushUndo(undo)
	a.toastAction = &toastAction{Message: a.tf("renamedTo", newName), Label: a.t("undo"), Run: func() { a.runUndo(undo) }}
	p.ScrollToPath = dst
	a.refreshPane(p)
}

// commitRename renames the entry in the rename dialog.
func (a *App) commitRename() {
	st := a.rename
	if st.Path == "" || st.Name == "" {
		a.rename.Open = false
		return
	}
	newName := strings.TrimSpace(st.Name)
	if newName == "" || strings.ContainsAny(newName, `/\`) {
		a.toast = a.t("invalidName")
		return
	}
	dst := filepath.Join(filepath.Dir(st.Path), newName)
	if newName == filepath.Base(st.Path) {
		a.rename.Open = false
		return
	}
	if platform.Exists(dst) {
		a.toast = a.t("nameExists")
		return
	}
	if err := os.Rename(st.Path, dst); err != nil {
		a.toast = a.t("renameFailed")
		return
	}
	a.remapPath(st.Path, dst)
	undo := undoAction{Label: a.t("undoRename"), Run: func() error {
		if err := os.Rename(dst, st.Path); err != nil {
			return err
		}
		a.remapPath(dst, st.Path)
		return nil
	}}
	a.pushUndo(undo)
	a.toastAction = &toastAction{Message: a.tf("renamedTo", newName), Label: a.t("undo"), Run: func() { a.runUndo(undo) }}
	a.rename.Open = false
	if st.Pane != nil {
		st.Pane.ScrollToPath = dst
		a.refreshPane(st.Pane)
	} else {
		a.refreshAllPanes()
	}
}

// beginNewFolder opens the new folder dialog.
func (a *App) beginNewFolder(p *Pane) {
	if p == nil || p.Virtual != virtualNone {
		return
	}
	name := a.t("newFolderTitle")
	a.newFolder = newFolderState{Open: true, Name: name, Pane: p}
}

// beginNewFile opens the new file dialog.
func (a *App) beginNewFile(p *Pane) {
	if p == nil || p.Virtual != virtualNone {
		return
	}
	a.newFolder = newFolderState{Open: true, Name: a.t("newFile"), Pane: p, File: true}
}

// commitNewFolder creates the folder in the dialog.
func (a *App) commitNewFolder() {
	st := a.newFolder
	if st.Pane == nil || strings.TrimSpace(st.Name) == "" {
		a.newFolder.Open = false
		return
	}
	name := strings.TrimSpace(st.Name)
	if strings.ContainsAny(name, `/\`) {
		a.toast = a.t("invalidName")
		return
	}
	path := filepath.Join(st.Pane.Path, name)
	if platform.Exists(path) {
		a.toast = a.t("nameExists")
		return
	}
	if st.File {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			a.toast = a.t("createFailed")
			return
		}
		f.Close()
	} else if err := os.Mkdir(path, 0o755); err != nil {
		a.toast = a.t("createFailed")
		return
	}
	undo := undoAction{Label: a.t("undoNewFolder"), Run: func() error { return os.Remove(path) }}
	a.pushUndo(undo)
	created := a.t("createdFolder")
	if st.File {
		created = a.t("createdFile")
	}
	a.toastAction = &toastAction{Message: created, Label: a.t("undo"), Run: func() { a.runUndo(undo) }}
	a.newFolder.Open = false
	st.Pane.ScrollToPath = path
	a.refreshPane(st.Pane)
}

// duplicateEntry makes a copy of an entry next to it.
func (a *App) duplicateEntry(p *Pane, e fsx.Entry) {
	dst := uniqueDest(e.Path, a.t("copySuffix"))
	if err := copyPath(e.Path, dst); err != nil {
		a.toast = a.t("copyFailed")
		return
	}
	undo := undoAction{Label: a.t("undoCopied"), Run: func() error { return os.RemoveAll(dst) }}
	a.pushUndo(undo)
	a.toastAction = &toastAction{Message: a.tf("copiedItems", 1), Label: a.t("undo"), Run: func() { a.runUndo(undo) }}
	p.ScrollToPath = dst
	a.refreshPane(p)
}

// refreshPanesAt reloads panes showing a folder.
func (a *App) refreshPanesAt(path string) {
	clean := filepath.Clean(path)
	for _, p := range a.allPanes() {
		if p.Virtual == virtualNone && filepath.Clean(p.Path) == clean {
			a.refreshPane(p)
		}
	}
}

// refreshAllPanes reloads every pane.
func (a *App) refreshAllPanes() {
	for _, p := range a.allPanes() {
		a.refreshPane(p)
	}
}

// uniqueDest returns a free path beside path, with an optional suffix.
func uniqueDest(path string, suffix ...string) string {
	if !platform.Exists(path) {
		return path
	}
	dir, name := filepath.Split(path)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	suf := " (1)"
	if len(suffix) > 0 && suffix[0] != "" {
		suf = suffix[0]
	}
	for i := 1; i < 1000; i++ {
		var cand string
		if i == 1 {
			cand = filepath.Join(dir, base+suf+ext)
		} else {
			cand = filepath.Join(dir, fmt.Sprintf("%s%s (%d)%s", base, suf, i, ext))
		}
		if !platform.Exists(cand) {
			return cand
		}
	}
	return path
}

// copyPath copies a file or folder recursively.
func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Chmod(dst, info.Mode().Perm()); err != nil {
		return err
	}
	return os.Chtimes(dst, info.ModTime(), info.ModTime())
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// mergeCopy copies a folder's contents into an existing folder, merging
// folders and replacing files of the same name.
func mergeCopy(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return copyPath(src, dst)
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		si, err := os.Lstat(s)
		if err != nil {
			return err
		}
		di, err := os.Lstat(d)
		if err == nil {
			if si.IsDir() && di.IsDir() {
				if err := mergeCopy(s, d); err != nil {
					return err
				}
				continue
			}
			if err := os.RemoveAll(d); err != nil {
				return err
			}
		}
		if err := copyPath(s, d); err != nil {
			return err
		}
	}
	return nil
}

// movePath moves a path, copying across volumes when needed.
func movePath(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyPath(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

// compareStash opens the comparison of the two staged files.
func (a *App) compareStash() {
	if len(a.stash.Items) != 2 {
		a.toast = a.t("compareNeedsTwo")
		return
	}
	x, y := a.stash.Items[0], a.stash.Items[1]
	a.openDiff(x.Path, y.Path)
}

// folderPrefOf returns a pane's stored preference.
func (a *App) folderPrefOf(path string) store.FolderPref {
	return a.cfg.FolderPrefs[store.FolderKey(path)]
}

// setTag assigns or clears a tag on paths.
func (a *App) setTag(paths []string, id string) {
	if len(paths) == 0 {
		return
	}
	for _, path := range paths {
		if id == "" {
			delete(a.cfg.TagAssignments, path)
		} else {
			a.cfg.TagAssignments[path] = id
		}
	}
	a.saveSessionSoon()
	for _, p := range a.allPanes() {
		if p.Virtual == virtualTag {
			a.loadVirtual(p)
		}
	}
	if id == "" {
		a.toast = a.tf("tagRemovedFrom", len(paths))
	} else {
		a.toast = a.tf("tagAppliedTo", len(paths))
	}
}

// remapPath moves a path's tag, favorite, quick shortcut and recent entry
// to a new path after a rename or move.
func (a *App) remapPath(old, new string) {
	if old == "" || new == "" || old == new {
		return
	}
	if tag, ok := a.cfg.TagAssignments[old]; ok {
		delete(a.cfg.TagAssignments, old)
		a.cfg.TagAssignments[new] = tag
	}
	name := filepath.Base(new)
	for i := range a.cfg.Favorites {
		if a.cfg.Favorites[i].Path == old {
			a.cfg.Favorites[i].Path = new
			a.cfg.Favorites[i].Name = name
		}
	}
	for i := range a.cfg.Quick {
		if a.cfg.Quick[i].Path == old {
			a.cfg.Quick[i].Path = new
			a.cfg.Quick[i].Name = name
		}
	}
	for i := range a.recent {
		if a.recent[i].Path == old {
			a.recent[i].Path = new
			a.recent[i].Name = name
		}
	}
}

// forgetPath drops every record of a path that no longer exists.
func (a *App) forgetPath(path string) {
	if path == "" {
		return
	}
	delete(a.cfg.TagAssignments, path)
	for i := 0; i < len(a.cfg.Favorites); i++ {
		if a.cfg.Favorites[i].Path == path {
			a.cfg.Favorites = append(a.cfg.Favorites[:i], a.cfg.Favorites[i+1:]...)
			i--
		}
	}
	for i := 0; i < len(a.cfg.Quick); i++ {
		if a.cfg.Quick[i].Path == path {
			a.cfg.Quick = append(a.cfg.Quick[:i], a.cfg.Quick[i+1:]...)
			i--
		}
	}
	for i := 0; i < len(a.recent); i++ {
		if a.recent[i].Path == path {
			a.recent = append(a.recent[:i], a.recent[i+1:]...)
			i--
		}
	}
}

// tagIndex returns a tag's index, -1 when absent.
func (a *App) tagIndex(id string) int {
	for i, t := range a.cfg.Tags {
		if t.ID == id {
			return i
		}
	}
	return -1
}

// deleteTag removes a tag and its assignments.
func (a *App) deleteTag(id string) {
	for i, t := range a.cfg.Tags {
		if t.ID == id {
			a.cfg.Tags = append(a.cfg.Tags[:i], a.cfg.Tags[i+1:]...)
			break
		}
	}
	for path, tag := range a.cfg.TagAssignments {
		if tag == id {
			delete(a.cfg.TagAssignments, path)
		}
	}
	a.saveSessionSoon()
	for _, p := range a.allPanes() {
		if p.Virtual == virtualTag {
			a.loadVirtual(p)
		}
	}
}

// saveTagEdits applies the tag manager's edits.
func (a *App) saveTagEdits() {
	st := a.tags
	if strings.TrimSpace(st.Name) == "" {
		a.tags.Open = false
		return
	}
	name := strings.TrimSpace(st.Name)
	if st.Editing >= 0 && st.Editing < len(a.cfg.Tags) {
		a.cfg.Tags[st.Editing].Name = name
		a.cfg.Tags[st.Editing].Color = st.Color
	} else {
		id := fmt.Sprintf("tag%d", time.Now().UnixNano())
		a.cfg.Tags = append(a.cfg.Tags, store.Tag{ID: id, Name: name, Color: st.Color})
	}
	a.saveSessionSoon()
	a.tags.Open = false
}

// startFolderCompare compares the active tabs of both sides.
func (a *App) startFolderCompare() {
	left, right := a.sidePane(0), a.sidePane(1)
	if left == nil || right == nil {
		return
	}
	if left.Virtual != virtualNone || right.Virtual != virtualNone {
		return
	}
	a.compare = compareState{Open: true, Left: left, Right: right}
	for _, p := range []*Pane{left, right} {
		if p.Compare == nil {
			p.Compare = map[string]compareMark{}
		}
	}
	a.computeCompare(left, right)
}

// computeCompare marks the differences between two folders.
func (a *App) computeCompare(left, right *Pane) {
	left.Compare = map[string]compareMark{}
	right.Compare = map[string]compareMark{}
	lm := map[string]fsx.Entry{}
	rm := map[string]fsx.Entry{}
	for _, e := range left.Entries {
		lm[strings.ToLower(e.Name)] = e
	}
	for _, e := range right.Entries {
		rm[strings.ToLower(e.Name)] = e
	}
	c := a.compare
	c.Same, c.LeftOnly, c.RightOnly, c.Newer, c.Different = 0, 0, 0, 0, 0
	for name, le := range lm {
		re, ok := rm[name]
		if !ok {
			c.LeftOnly++
			left.Compare[le.Path] = compareMark{Status: 1}
			continue
		}
		if le.IsDir && re.IsDir {
			c.Same++
			left.Compare[le.Path] = compareMark{Status: 5}
			right.Compare[re.Path] = compareMark{Status: 5}
			continue
		}
		if le.Size != re.Size {
			c.Different++
			left.Compare[le.Path] = compareMark{Status: 4}
			right.Compare[re.Path] = compareMark{Status: 4}
			continue
		}
		if !le.ModTime.Equal(re.ModTime) {
			c.Newer++
			if le.ModTime.After(re.ModTime) {
				left.Compare[le.Path] = compareMark{Status: 3}
				right.Compare[re.Path] = compareMark{Status: 3}
			} else {
				left.Compare[le.Path] = compareMark{Status: 3}
				right.Compare[re.Path] = compareMark{Status: 3}
			}
			continue
		}
		c.Same++
		left.Compare[le.Path] = compareMark{Status: 5}
		right.Compare[re.Path] = compareMark{Status: 5}
	}
	for name, re := range rm {
		if _, ok := lm[name]; !ok {
			c.RightOnly++
			right.Compare[re.Path] = compareMark{Status: 2}
		}
	}
	left.CompareOn = true
	right.CompareOn = true
	a.compare = c
}

// compareLabel names a comparison status.
func (a *App) compareLabel(s int) string {
	switch s {
	case 1:
		return a.t("compareLeftOnly")
	case 2:
		return a.t("compareRightOnly")
	case 3:
		return a.t("compareNewer")
	case 4:
		return a.t("compareDifferent")
	case 5:
		return a.t("compareSame")
	}
	return ""
}

// compareColor returns the color of a comparison status.
func (a *App) compareColor(s int, p Palette) ui.Color {
	switch s {
	case 1, 2:
		return p.Warning
	case 3:
		return p.TagBlue
	case 4:
		return p.Danger
	}
	return p.Success
}

// errorsJoin is a small helper for reporting several errors.
func errorsJoin(errs []error) error {
	var out []string
	for _, err := range errs {
		if err != nil {
			out = append(out, err.Error())
		}
	}
	if len(out) == 0 {
		return nil
	}
	return errors.New(strings.Join(out, "; "))
}
