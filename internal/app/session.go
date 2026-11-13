package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/session"
	"github.com/xddcode/markgo/internal/storage"
)

const recoveryDelay = 750 * time.Millisecond
const recoveryMaxDelay = 5 * time.Second

type workspaceSession struct {
	sessionStore    *session.Store
	sessionWorkers  sync.WaitGroup
	sessionRevision uint64
	sessionRunning  bool
	sessionError    string
	sessionObserved []documentStamp
	sessionFirst    time.Time
	sessionLast     time.Time
	backedUp        map[string]uint64
	recovered       map[string]string
	recent          []string
	recentOpen      bool
	recentWasOpen   bool
	recentQuery     string
	recentMissing   map[string]string
	exitPrompt      bool
	exitPromptError bool
	exitQuit        bool
	returnToWriting bool
}

type documentStamp struct {
	id, name, path, workspace string
	version                   uint64
	dirty, active             bool
	selection                 editor.Selection
	scroll                    float32
}

type recoveryCapture struct {
	state session.State
	text  []mdeditor.SaveSnapshot
}

// LoadSession runs before the event loop, never from a view. If parsing fails,
// writes remain detached so a newer app can still read the original journal.
func (a *App) LoadSession() error {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		a.sessionError = err.Error()
		return err
	}
	return a.loadSession(&session.Store{Path: filepath.Join(dir, "session.json")})
}

func (a *App) loadSession(store *session.Store) error {
	state, err := store.Load()
	if err != nil {
		a.sessionError = err.Error()
		return err
	}
	a.sessionStore = store
	a.recent = slices.Clone(state.Recent)
	var files []*mdeditor.File
	for _, doc := range state.Documents {
		var file *mdeditor.File
		var snapshot storage.Snapshot
		var data []byte
		var readErr error
		if doc.Path != "" {
			snapshot, data, readErr = storage.Open(doc.Path)
		}
		reason := ""
		switch {
		case doc.Dirty:
			if doc.Path != "" && readErr == nil && string(data) == doc.Source {
				// A disk save completed just before the checkpoint was updated.
				file = mdeditor.NewFile(doc.Name, string(data))
			} else {
				file = mdeditor.RecoverFile(doc.Name, doc.Source, doc.Original)
				reason = "Your unfinished writing is ready. Review it, then save when you are ready."
				if doc.Path != "" && (readErr != nil || string(data) != doc.Original) {
					reason = "The original is unavailable or has changed. Your draft is intact; save a copy to keep both."
					snapshot = storage.Snapshot{}
				}
			}
		case doc.Path == "":
			file = mdeditor.NewFile(doc.Name, doc.Source)
		case readErr != nil:
			a.recentMissing[doc.Path] = readErr.Error()
			a.rememberRecent(doc.Path)
			continue
		default:
			file = mdeditor.NewFile(doc.Name, string(data))
		}
		file.ID = doc.ID
		file.SetPath(doc.Path)
		file.State.Select(doc.Anchor, doc.Head)
		file.State.SetScrollY(float64(doc.ScrollY))
		files = append(files, file)
		if snapshot.Path != "" {
			a.disk[file.ID] = snapshot
		}
		if reason != "" {
			a.recovered[file.ID] = reason
			if doc.Path != "" && snapshot.Path == "" {
				a.conflicts[file.ID] = true
			}
		}
		if doc.Dirty {
			a.backedUp[file.ID] = file.State.Buffer.Version()
		}
	}
	if len(files) > 0 {
		a.files = files
		a.selected = files[0].ID
		for _, file := range files {
			if file.ID == state.Active {
				a.selected = file.ID
			}
		}
	}
	// Do not reuse an ID belonging to a restored document, including guides.
	for _, file := range a.files {
		var number int
		if _, err := fmt.Sscanf(file.ID, "document-%d", &number); err == nil {
			a.nextID = max(a.nextID, number)
		}
	}
	a.sidebar, a.reading, a.preview = state.Sidebar, state.Reading, !state.Source
	if state.Outline {
		a.sideView = 1
	}
	if state.Folder != "" {
		folder, err := storage.ListFolder(state.Folder)
		if err == nil {
			a.folder = &folder
		} else {
			a.notice = "The previous folder could not be reopened. Your documents are still available."
		}
	}
	return nil
}

func (a *App) rememberRecent(path string) {
	if path == "" {
		return
	}
	a.recent = slices.DeleteFunc(a.recent, func(p string) bool { return p == path })
	a.recent = append([]string{path}, a.recent...)
	a.recent = a.recent[:min(len(a.recent), session.MaxRecent)]
}

func (a *App) sessionStamp() []documentStamp {
	workspace := fmt.Sprintf("%t:%t:%t:%d:%v", a.sidebar, a.reading, a.preview, a.sideView, a.recent)
	if a.folder != nil {
		workspace += a.folder.Path
	}
	stamps := make([]documentStamp, 0, len(a.files))
	for _, file := range a.files {
		stamps = append(stamps, documentStamp{
			id: file.ID, name: file.Name, path: file.Path, workspace: workspace,
			version: file.State.Buffer.Version(), dirty: file.Modified(), active: file.ID == a.selected,
			selection: file.State.Selection(), scroll: float32(file.State.ScrollY()),
		})
	}
	return stamps
}

// scheduleRecovery observes cheap versions, not document text. The only work
// started in a build is guarded by a deadline and a single in-flight write.
func (a *App) scheduleRecovery(c *ui.Context) {
	if a.sessionStore == nil || a.window == nil || a.closing || a.allowExit || a.sessionError != "" {
		return
	}
	delay, ready := a.recoveryDue(c.Now())
	if delay > 0 {
		c.After(delay)
	}
	if ready {
		a.writeRecovery(a.captureRecovery(false), nil)
	}
}

func (a *App) recoveryDue(now time.Time) (time.Duration, bool) {
	stamp := a.sessionStamp()
	if !slices.Equal(stamp, a.sessionObserved) {
		a.sessionObserved = stamp
		if a.sessionFirst.IsZero() {
			a.sessionFirst = now
		}
		a.sessionLast = now
	}
	if a.sessionFirst.IsZero() || a.sessionRunning {
		return 0, false
	}
	due := a.sessionLast.Add(recoveryDelay)
	if limit := a.sessionFirst.Add(recoveryMaxDelay); due.After(limit) {
		due = limit
	}
	if now.Before(due) {
		return due.Sub(now), false
	}
	a.sessionFirst = time.Time{}
	return 0, true
}

func (a *App) captureRecovery(discard bool) recoveryCapture {
	state := session.State{Schema: session.SchemaVersion, Active: a.selected, Recent: slices.Clone(a.recent),
		Sidebar: a.sidebar, Reading: a.reading, Source: !a.preview, Outline: a.sideView == 1}
	if a.folder != nil {
		state.Folder = a.folder.Path
	}
	capture := recoveryCapture{state: state}
	for _, file := range a.files {
		if discard && file.Path == "" {
			continue
		}
		sel := file.State.Selection()
		doc := session.Document{ID: file.ID, Name: file.Name, Path: file.Path, Dirty: !discard && file.Modified(),
			Version: file.State.Buffer.Version(), Anchor: sel.Anchor, Head: sel.Head, ScrollY: float32(file.State.ScrollY())}
		capture.state.Documents = append(capture.state.Documents, doc)
		capture.text = append(capture.text, file.CaptureForSave())
	}
	return capture
}

// serializeRecovery is called on a worker. Source snapshots and original disk
// bytes travel together, including mixed endings and the exact saved version.
func (capture recoveryCapture) serialize() (session.State, error) {
	state := capture.state
	state.Documents = slices.Clone(state.Documents)
	size := 0
	for i := range state.Documents {
		doc := &state.Documents[i]
		if doc.Dirty || doc.Path == "" {
			// CRLF output can take twice the normalized snapshot size.
			size += 2*capture.text[i].Text.Len() + len(capture.text[i].OriginalSource())
			if size > session.MaxBytes {
				return state, fmt.Errorf("recovery exceeds the 64 MiB budget; save your documents to disk")
			}
			data, err := capture.text[i].NormalizedBytes()
			if err != nil {
				return state, err
			}
			doc.Source = string(data)
			doc.Original = capture.text[i].OriginalSource()
		}
	}
	return state, nil
}

func (a *App) writeRecovery(capture recoveryCapture, done func(error)) {
	a.sessionRevision++
	a.sessionRunning = true
	revision, store, window := a.sessionRevision, a.sessionStore, a.window
	a.sessionWorkers.Go(func() {
		state, err := capture.serialize()
		if err == nil {
			err = store.Save(revision, state)
		}
		if window == nil {
			return
		}
		window.Update(func() {
			a.acceptRecovery(revision, state, err)
			if done != nil {
				done(err)
			}
		})
	})
}

func (a *App) acceptRecovery(revision uint64, state session.State, err error) {
	if revision != a.sessionRevision {
		return
	}
	a.sessionRunning = false
	a.sessionError = ""
	if err != nil {
		a.sessionError = err.Error()
		return
	}
	a.backedUp = make(map[string]uint64)
	for _, doc := range state.Documents {
		a.backedUp[doc.ID] = doc.Version
	}
}

func (capture recoveryCapture) without(id string) recoveryCapture {
	for i, doc := range capture.state.Documents {
		if doc.ID == id {
			capture.state.Documents = slices.Delete(capture.state.Documents, i, i+1)
			capture.text = slices.Delete(capture.text, i, i+1)
			if capture.state.Active == id {
				capture.state.Active = ""
			}
			break
		}
	}
	return capture
}

func (a *App) discardDocument(id string) {
	if a.sessionStore == nil {
		a.removeDocument(id)
		return
	}
	a.closing, a.fileBusy = true, true
	a.writeRecovery(a.captureRecovery(false).without(id), func(err error) {
		a.closing, a.fileBusy = false, false
		if err != nil {
			a.notice = "Could not discard the recovery copy. Your document remains open."
			return
		}
		a.removeDocument(id)
	})
}

// WaitForSession is used only after the event loop, never from a view.
func (a *App) WaitForSession() { a.sessionWorkers.Wait() }

func (a *App) clearRecent() {
	a.recent, a.recentQuery = nil, ""
	clear(a.recentMissing)
}

func (a *App) forgetRecent(path string) {
	a.recent = slices.DeleteFunc(a.recent, func(p string) bool { return p == path })
	delete(a.recentMissing, path)
}
