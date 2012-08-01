package rev

import (
	"github.com/howeyc/fsnotify"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Listener is an interface for receivers of filesystem events.
type Listener interface {
	// Refresh is invoked by the watcher on relevant filesystem events.
	// If the listener returns an error, it is served to the user on the current request.
	Refresh() *Error
}

// Watcher allows listeners to register to be notified of changes under a given
// directory.
type Watcher struct {
	// Parallel arrays of watcher/listener pairs.
	watchers  []*fsnotify.Watcher
	listeners []Listener
	lastError int
}

func NewWatcher() *Watcher {
	return &Watcher{
		lastError: -1,
	}
}

// Listen registers for events within the given root directories (recursively).
// The caller may specify directory names to skip (not watch or recurse into).
func (w *Watcher) Listen(listener Listener, roots []string, skip []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		ERROR.Fatal(err)
	}

	// Replace the unbuffered Event channel with a buffered one.
	// Otherwise multiple change events only come out one at a time, across
	// multiple page views.
	watcher.Event = make(chan *fsnotify.FileEvent, 10)
	watcher.Error = make(chan error, 10)

	// Walk through all files / directories under the root, adding each to watcher.
	for _, p := range roots {
		filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				ERROR.Println("Error walking path:", err)
				return nil
			}
			if info.IsDir() {
				if ContainsString(skip, info.Name()) {
					return filepath.SkipDir
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
	for i, watcher := range w.watchers {
		listener := w.listeners[i]

		// Pull all pending events / errors from the watcher.
		refresh := false
		for {
			select {
			case ev := <-watcher.Event:
				// Ignore changes to dotfiles.
				if !strings.HasPrefix(path.Base(ev.Name), ".") {
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

		if refresh || w.lastError == i {
			err := listener.Refresh()
			if err != nil {
				w.lastError = i
				return err
			}
		}
	}

	w.lastError = -1
	return nil
}
