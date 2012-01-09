// A inotify-style filesystem watcher for Mac, using kqueue.
// It's purpose is watching and notifying for source code changes.
package harness

import (
	"log"
	"os"
	"syscall"
_	"go/ast"
	"strings"
	"path/filepath"
)

// Watches app directories and sends events when .go files are
// added/deleted/modified.
type Watcher struct {
	kqueue int  // file descriptor

	// Book-keeping
	dirFileMap map[int] *os.File // FD to directory File
	dirContentsMap map[int] []os.FileInfo  // FD to directory FileInfos (filtered to .go files only)

	// Output
	Event chan *WatcherEvent
	Error chan error
}

type WatcherEvent struct {
	DirNames []string  // Directory paths where changes were detected.
}

func filterFileInfos(fileInfos []os.FileInfo) (filteredFileInfos []os.FileInfo) {
	filteredFileInfos = make([]os.FileInfo, 0, len(fileInfos))
	for _, fileInfo := range fileInfos {
		if strings.HasSuffix(fileInfo.Name(), ".go") {
			filteredFileInfos = append(filteredFileInfos, fileInfo)
		}
	}
	return
}

func setupWatcher(path string) *Watcher {
	// Create a kqueue
	kqueue, err := syscall.Kqueue()
	if err != nil {
		log.Fatalf("Failed to create kqueue: %s", err)
	}

	// Iteratively descend into the directories, using a queue.
	var dirFileMap map[int] *os.File = make(map[int] *os.File)
	var dirContentsMap map[int] []os.FileInfo = make(map[int] []os.FileInfo)
	var dirQueue []string = []string{path}
	for len(dirQueue) > 0 {
		// Get the next directory to add to the watcher.
		dirPath := dirQueue[0]
		dirQueue = dirQueue[1:]
		dir, err := os.Open(dirPath)
		if err != nil {
			log.Fatalf("Failed to open app path: %s", err)
		}

		// Read / filter / store the fileinfos.
		fileInfos, err := dir.Readdir(-1)
		if err != nil {
			log.Fatalf("Failed to read directory: %s", err)
		}
		for _, fileInfo := range fileInfos {
			if fileInfo.IsDir() {
				dirQueue = append(dirQueue, filepath.Join(dirPath, fileInfo.Name()))
			}
		}
		dirFileMap[dir.Fd()] = dir
		dirContentsMap[dir.Fd()] = filterFileInfos(fileInfos)

		// Register the kevent (a write to the directory) and verify the receipt.
		var kbuf [1]syscall.Kevent_t
		ev := &kbuf[0]
		syscall.SetKevent(&kbuf[0], dir.Fd(), syscall.EVFILT_VNODE, syscall.EV_ADD|syscall.EV_RECEIPT|syscall.EV_CLEAR)
		ev.Fflags = syscall.NOTE_WRITE
		n, err := syscall.Kevent(kqueue, kbuf[0:], kbuf[0:], nil)
		if err != nil {
			log.Fatalf("Kevent failed: %s", err)
		}

		if n != 1 ||
			(ev.Flags&syscall.EV_ERROR) == 0 ||
			int(ev.Ident) != dir.Fd() ||
			int(ev.Filter) != syscall.EVFILT_VNODE {
			log.Fatalf("Kevent failed")
		}
		log.Println("Listening for changes:", dirPath)
	}
	return &Watcher{kqueue, dirFileMap, dirContentsMap, make(chan *WatcherEvent), make(chan error)}
}

// Return a watcher that sends an event each time a sub-directory changes.
func NewWatcher(path string) *Watcher {
	// A goroutine is used to subsequently generate watcher events.
	// The setup must be done within the goroutine since file descriptors do not
	// necessarily transfer to forked threads.
	// Therefore, we have an "extra" channel to return the constructed watcher.
	watcherChan := make(chan *Watcher)
	go func() {
		watcher := setupWatcher(path)
		watcherChan <- watcher
		watchForever(watcher)
	}()
	return <-watcherChan
}

// This function relays events from the watched directories.
func watchForever(watcher *Watcher) {
	var events [10]syscall.Kevent_t
	for {
		// Block, waiting for the next kevent.
		n, err := syscall.Kevent(watcher.kqueue, nil, events[0:], nil)
		if err != nil {
			watcher.Error <- err
			continue
		}

		// Typically there will only be one event returned.
		var dirNames []string
		for nEvent := 0; nEvent < n; nEvent++ {
			// Figure out which FileInfos changed.
			// ASSUMPTION: fileinfos are returned in the same order.
			// Loop through arrays in parallel.
			event := events[nEvent]
			fd := int(event.Ident)
			var oldFileInfos []os.FileInfo = watcher.dirContentsMap[fd]
			var dir *os.File = watcher.dirFileMap[fd]

			// Open a fresh file.. not sure why this is necessary.
			// Otherwise, Readdir returns "readdirent: invalid argument" error, on
			// some (not all) changes.
			dir, err = os.Open(dir.Name())
			if err != nil {
				log.Fatalln("Failed to open dir:", dir.Name())
			}
			defer dir.Close()
			newFileInfos, err := dir.Readdir(-1)
			if err != nil {
				log.Fatalf("Read dir failed: %s", err)
			}
			newFileInfos = filterFileInfos(newFileInfos)

			// Check that there are still the same number of files.
			if len(oldFileInfos) != len(newFileInfos) {
				log.Println("Detected file add or delete:", dir.Name())
				dirNames = append(dirNames, dir.Name())
				watcher.dirContentsMap[fd] = newFileInfos
				continue
			}

			// Check the name / modification time on each entry.
			for nFileInfo := 0; nFileInfo < len(newFileInfos); nFileInfo++ {
				oldf := oldFileInfos[nFileInfo]
				newf := newFileInfos[nFileInfo]
				if oldf.Name() != newf.Name() {
					log.Printf("File rename detected: %s -> %s\n", oldf.Name(), newf.Name())
				} else if !oldf.ModTime().Equal(newf.ModTime()) {
					log.Printf("File modification detected: %s modified %s\n",
						oldf.Name(), newf.ModTime().String())
				} else {
					continue  // No change.
				}

				// There was a change.
				// Record the dir name and skip checking the rest of the directory.
				dirNames = append(dirNames, dir.Name())
				break
			}
		}

		if len(dirNames) > 0 {
			watcher.Event <- &WatcherEvent{dirNames}
		}
	}
}
