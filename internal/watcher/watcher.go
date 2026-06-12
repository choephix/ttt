// Package watcher reports when files open in the editor are modified on disk
// by another process. It wraps fsnotify, watching the parent directories of
// tracked files (the portable, rename-safe pattern) and debouncing bursts of
// events into a single notification per file.
package watcher

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// defaultDebounce coalesces the rapid bursts of events that a single logical
// write produces (including the temp-file + rename pattern the editor itself
// uses to save) into one notification.
const defaultDebounce = 150 * time.Millisecond

// Watcher tracks a set of files and invokes onChange when one of them changes
// on disk. onChange is called from an internal goroutine with the same path
// string that was passed to Sync, so callers can match it back to their own
// bookkeeping.
type Watcher struct {
	fsw      *fsnotify.Watcher
	onChange func(path string)
	debounce time.Duration

	mu     sync.Mutex
	files  map[string]string // cleaned-abs path -> original path as tracked
	dirs   map[string]int    // watched directory -> number of tracked files in it
	timers map[string]*time.Timer
	closed bool
}

// New creates a Watcher and starts its event loop. onChange must be safe to
// call from another goroutine.
func New(onChange func(path string)) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		fsw:      fsw,
		onChange: onChange,
		debounce: defaultDebounce,
		files:    make(map[string]string),
		dirs:     make(map[string]int),
		timers:   make(map[string]*time.Timer),
	}
	go w.run()
	return w, nil
}

// Sync reconciles the tracked set with paths, adding watches for newly opened
// files and dropping them for closed ones. It is a no-op for paths already
// tracked, so it is cheap to call frequently.
func (w *Watcher) Sync(paths []string) {
	want := make(map[string]string, len(paths))
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		want[filepath.Clean(abs)] = p
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	for key := range w.files {
		if _, ok := want[key]; !ok {
			w.untrackLocked(key)
		}
	}
	for key, orig := range want {
		if _, ok := w.files[key]; !ok {
			w.trackLocked(key, orig)
		}
	}
}

func (w *Watcher) trackLocked(key, orig string) {
	dir := filepath.Dir(key)
	if w.dirs[dir] == 0 {
		if err := w.fsw.Add(dir); err != nil {
			return
		}
	}
	w.dirs[dir]++
	w.files[key] = orig
}

func (w *Watcher) untrackLocked(key string) {
	if _, ok := w.files[key]; !ok {
		return
	}
	delete(w.files, key)
	if t := w.timers[key]; t != nil {
		t.Stop()
		delete(w.timers, key)
	}
	dir := filepath.Dir(key)
	if w.dirs[dir] > 0 {
		w.dirs[dir]--
		if w.dirs[dir] == 0 {
			delete(w.dirs, dir)
			_ = w.fsw.Remove(dir)
		}
	}
}

func (w *Watcher) run() {
	for {
		select {
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			// Only Chmod-only events are uninteresting; writes, creates,
			// renames and removes can all change the file's contents.
			if ev.Op == fsnotify.Chmod {
				continue
			}
			w.handle(filepath.Clean(ev.Name))
		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) handle(key string) {
	w.mu.Lock()
	orig, tracked := w.files[key]
	if !tracked || w.closed {
		w.mu.Unlock()
		return
	}
	if t := w.timers[key]; t != nil {
		t.Stop()
	}
	w.timers[key] = time.AfterFunc(w.debounce, func() {
		w.mu.Lock()
		_, still := w.files[key]
		delete(w.timers, key)
		closed := w.closed
		w.mu.Unlock()
		if still && !closed {
			w.onChange(orig)
		}
	})
	w.mu.Unlock()
}

// Close stops watching and releases resources. The Watcher must not be used
// afterwards.
func (w *Watcher) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	for key, t := range w.timers {
		t.Stop()
		delete(w.timers, key)
	}
	w.mu.Unlock()
	return w.fsw.Close()
}
