package revel

import (
	"github.com/howeyc/fsnotify"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
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
	watcher.Event = make(chan *fsnotify.FileEvent, 100)
	watcher.Error = make(chan error, 10)

	// Walk through all files / directories under the root, adding each to watcher.
	for _, p := range roots {
		fi, err := os.Stat(p)
		if err != nil {
			ERROR.Println("Failed to stat watched path", p, ":", err)
			continue
		}

		// If it is a file, watch that specific file.
		if !fi.IsDir() {
			err = watcher.Watch(p)
			if err != nil {
				ERROR.Println("Failed to watch", p, ":", err)
			}
			TRACE.Println("Watching:", p)
			continue
		}

		// Else, walk the directory tree.
		filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
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

				err = watcher.Watch(path)
				if err != nil {
					ERROR.Println("Failed to watch", path, ":", err)
				}
				TRACE.Println("Watching:", path)
			}
			return nil
		})
	}

	w.watchers = append(w.watchers, watcher)
	w.listeners = append(w.listeners, listener)
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
			case ev := <-watcher.Event:
				// Ignore changes to dotfiles.
				if !strings.HasPrefix(path.Base(ev.Name), ".") {
					if dl, ok := listener.(DiscerningListener); ok {
						if !dl.WatchFile(ev.Name) || ev.IsAttrib() {
							continue
						}
					}

					refresh = true
				}
				continue
			case <-watcher.Error:
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
