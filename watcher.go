// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/fsnotify/fsnotify.v1"
	"time"
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
	serial              bool                // true to process events in serial
	watchers            []*fsnotify.Watcher // Parallel arrays of watcher/listener pairs.
	listeners           []Listener          // List of listeners for watcher
	forceRefresh        bool                // True to force the refresh
	lastError           int                 // The last error found
	notifyMutex         sync.Mutex          // The mutext to serialize watches
	refreshTimer        *time.Timer         // The timer to countdown the next refresh
	timerMutex          *sync.Mutex         // A mutex to prevent concurrent updates
	refreshChannel      chan *Error         // The error channel to listen to when waiting for a refresh
	refreshChannelCount int                 // The number of clients listening on the channel
	refreshTimerMS      time.Duration       // The number of milliseconds between refreshing builds
}

func NewWatcher() *Watcher {
	return &Watcher{
		forceRefresh:        true,
		lastError:           -1,
		refreshTimerMS:      time.Duration(Config.IntDefault("watch.rebuild.delay", 10)),
		timerMutex:          &sync.Mutex{},
		refreshChannel:      make(chan *Error, 10),
		refreshChannelCount: 0,
	}
}

// Listen registers for events within the given root directories (recursively).
func (w *Watcher) Listen(listener Listener, roots ...string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		utilLog.Fatal("Watcher: Failed to create watcher", "error", err)
	}

	// Replace the unbuffered Event channel with a buffered one.
	// Otherwise multiple change events only come out one at a time, across
	// multiple page views.  (There appears no way to "pump" the events out of
	// the watcher)
	// This causes a notification when you do a check in go, since you are modifying a buffer in use
	watcher.Events = make(chan fsnotify.Event, 100)
	watcher.Errors = make(chan error, 10)

	// Walk through all files / directories under the root, adding each to watcher.
	for _, p := range roots {
		// is the directory / file a symlink?
		f, err := os.Lstat(p)
		if err == nil && f.Mode()&os.ModeSymlink == os.ModeSymlink {
			var realPath string
			realPath, err = filepath.EvalSymlinks(p)
			if err != nil {
				panic(err)
			}
			p = realPath
		}

		fi, err := os.Stat(p)
		if err != nil {
			utilLog.Error("Watcher: Failed to stat watched path, code will continue but auto updates will not work", "path", p, "error", err)
			continue
		}

		// If it is a file, watch that specific file.
		if !fi.IsDir() {
			err = watcher.Add(p)
			if err != nil {
				utilLog.Error("Watcher: Failed to watch, code will continue but auto updates will not work", "path", p, "error", err)
			}
			continue
		}

		var watcherWalker func(path string, info os.FileInfo, err error) error

		watcherWalker = func(path string, info os.FileInfo, err error) error {
			if err != nil {
				utilLog.Error("Watcher: Error walking path:", "error", err)
				return nil
			}

			if info.IsDir() {
				if dl, ok := listener.(DiscerningListener); ok {
					if !dl.WatchDir(info) {
						return filepath.SkipDir
					}
				}

				err := watcher.Add(path)
				if err != nil {
					utilLog.Error("Watcher: Failed to watch this path, code will continue but auto updates will not work", "path", path, "error", err)
				}
			}
			return nil
		}

		// Else, walk the directory tree.
		err = Walk(p, watcherWalker)
		if err != nil {
			utilLog.Error("Watcher: Failed to walk directory, code will continue but auto updates will not work", "path", p, "error", err)
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
				if w.serial {
					// Serialize listener.Refresh() calls.
					w.notifyMutex.Lock()

					if err := listener.Refresh(); err != nil {
						utilLog.Error("Watcher: Listener refresh reported error:", "error", err)
					}
					w.notifyMutex.Unlock()
				} else {
					// Run refresh in parallel
					go func() {
						w.notifyInProcess(listener)
					}()
				}
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
			var err *Error
			if w.serial {
				err = listener.Refresh()
			} else {
				err = w.notifyInProcess(listener)
			}

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

// Build a queue for refresh notifications
// this will not return until one of the queue completes
func (w *Watcher) notifyInProcess(listener Listener) (err *Error) {
	shouldReturn := false
	// This code block ensures that either a timer is created
	// or that a process would be added the the h.refreshChannel
	func() {
		w.timerMutex.Lock()
		defer w.timerMutex.Unlock()
		// If we are in the process of a rebuild, forceRefresh will always be true
		w.forceRefresh = true
		if w.refreshTimer != nil {
			utilLog.Info("Found existing timer running, resetting")
			w.refreshTimer.Reset(time.Millisecond * w.refreshTimerMS)
			shouldReturn = true
			w.refreshChannelCount++
		} else {
			w.refreshTimer = time.NewTimer(time.Millisecond * w.refreshTimerMS)
		}
	}()

	// If another process is already waiting for the timer this one
	// only needs to return the output from the channel
	if shouldReturn {
		return <-w.refreshChannel
	}
	utilLog.Info("Waiting for refresh timer to expire")
	<-w.refreshTimer.C
	w.timerMutex.Lock()

	// Ensure the queue is properly dispatched even if a panic occurs
	defer func() {
		for x := 0; x < w.refreshChannelCount; x++ {
			w.refreshChannel <- err
		}
		w.refreshChannelCount = 0
		w.refreshTimer = nil
		w.timerMutex.Unlock()
	}()

	err = listener.Refresh()
	if err != nil {
		utilLog.Info("Watcher: Recording error last build, setting rebuild on", "error", err)
	} else {
		w.lastError = -1
		w.forceRefresh = false
	}
	utilLog.Info("Rebuilt, result", "error", err)
	return
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
	if strings.HasPrefix(filepath.Base(ev.Name), ".") {
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
