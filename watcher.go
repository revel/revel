package revel

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/fsnotify.v1"
)

// Listener is an interface for receivers of filesystem events.
type Listener interface {
	// Refresh is invoked by the watcher on relevant filesystem events.
	// If the listener returns an error, it is served to the user on the current request.
	Refresh() *Error
}

// DiscerningListener allows the receiver to selectively watch files.
type DiscerningListener interface {
	Listener
	WatchDir(info os.FileInfo) bool
	WatchFile(basename string) bool
}

// Watcher allows listeners to register to be notified of changes under a given
// directory.
type Watcher struct {
	// Parallel arrays of watcher/listener pairs.
	watchers     []*fsnotify.Watcher
	listeners    []Listener
	forceRefresh bool
	lastError    int
	notifyMutex  sync.Mutex
}

func NewWatcher() *Watcher {
	return &Watcher{
		forceRefresh: true,
		lastError:    -1,
	}
}

// Listen registers for events within the given root directories (recursively).
func (w *Watcher) Listen(listener Listener, roots ...string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		ERROR.Fatal(err)
	}

	// Replace the unbuffered Event channel with a buffered one.
	// Otherwise multiple change events only come out one at a time, across
	// multiple page views.  (There appears no way to "pump" the events out of
	// the watcher)
	watcher.Events = make(chan fsnotify.Event, 100)
	watcher.Errors = make(chan error, 10)

	// Walk through all files / directories under the root, adding each to watcher.
	for _, p := range roots {
		// is the directory / file a symlink?
		f, err := os.Lstat(p)
		if err == nil && f.Mode()&os.ModeSymlink == os.ModeSymlink {
			realPath, err := filepath.EvalSymlinks(p)
			if err != nil {
				panic(err)
			}
			p = realPath
		}

		fi, err := os.Stat(p)
		if err != nil {
			ERROR.Println("Failed to stat watched path", p, ":", err)
			continue
		}

		// If it is a file, watch that specific file.
		if !fi.IsDir() {
			err = watcher.Add(p)
			if err != nil {
				ERROR.Println("Failed to watch", p, ":", err)
			}
			continue
		}

		var watcherWalker func(path string, info os.FileInfo, err error) error

		watcherWalker = func(path string, info os.FileInfo, err error) error {
			if err != nil {
				ERROR.Println("Error walking path:", err)
				return nil
			}

			if info.IsDir() {
				if dl, ok := listener.(DiscerningListener); ok {
					if !dl.WatchDir(info) {
						return filepath.SkipDir
					}
				}

				err = watcher.Add(path)
				if err != nil {
					ERROR.Println("Failed to watch", path, ":", err)
				}
			}
			return nil
		}

		// Else, walk the directory tree.
		err = Walk(p, watcherWalker)
		if err != nil {
			ERROR.Println("Failed to walk directory", p, ":", err)
		}
	}

	if w.eagerRebuildEnabled() {
		// Create goroutine to notify file changes in real time
		go w.NotifyWhenUpdated(listener, watcher)
	}

	w.watchers = append(w.watchers, watcher)
	w.listeners = append(w.listeners, listener)
}

// NotifyWhenUpdated notifies the watcher when a file event is received.
func (w *Watcher) NotifyWhenUpdated(listener Listener, watcher *fsnotify.Watcher) {
	for {
		select {
		case ev := <-watcher.Events:
			if w.rebuildRequired(ev, listener) {
				// Serialize listener.Refresh() calls.
				w.notifyMutex.Lock()
				listener.Refresh()
				w.notifyMutex.Unlock()
			}
		case <-watcher.Errors:
			continue
		}
	}
}

// Notify causes the watcher to forward any change events to listeners.
// It returns the first (if any) error returned.
func (w *Watcher) Notify() *Error {
	// Serialize Notify() calls.
	w.notifyMutex.Lock()
	defer w.notifyMutex.Unlock()

	for i, watcher := range w.watchers {
		listener := w.listeners[i]

		// Pull all pending events / errors from the watcher.
		refresh := false
		for {
			select {
			case ev := <-watcher.Events:
				if w.rebuildRequired(ev, listener) {
					refresh = true
				}
				continue
			case <-watcher.Errors:
				continue
			default:
				// No events left to pull
			}
			break
		}

		if w.forceRefresh || refresh || w.lastError == i {
			err := listener.Refresh()
			if err != nil {
				w.lastError = i
				return err
			}
		}
	}

	w.forceRefresh = false
	w.lastError = -1
	return nil
}

// If watch.mode is set to eager, the application is rebuilt immediately
// when a source file is changed.
// This feature is available only in dev mode.
func (w *Watcher) eagerRebuildEnabled() bool {
	return Config.BoolDefault("mode.dev", true) &&
		Config.BoolDefault("watch", true) &&
		Config.StringDefault("watch.mode", "normal") == "eager"
}

func (w *Watcher) rebuildRequired(ev fsnotify.Event, listener Listener) bool {
	// Ignore changes to dotfiles.
	if strings.HasPrefix(path.Base(ev.Name), ".") {
		return false
	}

	if dl, ok := listener.(DiscerningListener); ok {
		if !dl.WatchFile(ev.Name) || ev.Op&fsnotify.Chmod == fsnotify.Chmod {
			return false
		}
	}
	return true
}

var WatchFilter = func(c *Controller, fc []Filter) {
	if MainWatcher != nil {
		err := MainWatcher.Notify()
		if err != nil {
			c.Result = c.RenderError(err)
			return
		}
	}
	fc[0](c, fc[1:])
}
