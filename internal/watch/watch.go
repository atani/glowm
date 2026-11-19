// Package watch detects changes to a file by polling its modification time and
// size, debouncing rapid edits so consumers re-render only once the file has
// settled.
package watch

import (
	"os"
	"time"
)

// DefaultPoll is how often the file is stat'd.
const DefaultPoll = 250 * time.Millisecond

// DefaultDebounce is how long the file must stop changing before a change is
// reported, so a re-render does not run against a half-written file.
const DefaultDebounce = 150 * time.Millisecond

// sig is a cheap fingerprint of a file's state. A change in modification time,
// size, or existence counts as a change.
type sig struct {
	mod    time.Time
	size   int64
	exists bool
}

func statSig(path string) sig {
	fi, err := os.Stat(path)
	if err != nil {
		return sig{exists: false}
	}
	return sig{mod: fi.ModTime(), size: fi.Size(), exists: true}
}

func (a sig) equal(b sig) bool {
	return a.exists == b.exists && a.size == b.size && a.mod.Equal(b.mod)
}

// Watcher polls a path and sends on C after the file changes and then stays
// stable for the debounce window. C is buffered (size 1): if a consumer is busy,
// repeated changes coalesce into a single pending notification.
type Watcher struct {
	C chan struct{}

	path     string
	poll     time.Duration
	debounce time.Duration
	stop     chan struct{}
	now      func() time.Time // injectable for tests
}

// New creates a Watcher for path. Zero poll/debounce use the defaults.
func New(path string, poll, debounce time.Duration) *Watcher {
	if poll <= 0 {
		poll = DefaultPoll
	}
	if debounce < 0 {
		debounce = DefaultDebounce
	}
	return &Watcher{
		C:        make(chan struct{}, 1),
		path:     path,
		poll:     poll,
		debounce: debounce,
		stop:     make(chan struct{}),
		now:      time.Now,
	}
}

// Start begins polling in a background goroutine.
func (w *Watcher) Start() {
	go w.loop()
}

// Stop ends polling. Safe to call once.
func (w *Watcher) Stop() {
	close(w.stop)
}

func (w *Watcher) loop() {
	last := statSig(w.path)
	var pending *sig
	var pendingSince time.Time

	ticker := time.NewTicker(w.poll)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			cur := statSig(w.path)
			if pending != nil {
				if !cur.equal(*pending) {
					// Still changing; reset the debounce window.
					p := cur
					pending = &p
					pendingSince = w.now()
					continue
				}
				if w.now().Sub(pendingSince) >= w.debounce {
					last = cur
					pending = nil
					w.emit()
				}
				continue
			}
			if !cur.equal(last) {
				p := cur
				pending = &p
				pendingSince = w.now()
			}
		}
	}
}

func (w *Watcher) emit() {
	select {
	case w.C <- struct{}{}:
	default: // a notification is already pending; coalesce.
	}
}
