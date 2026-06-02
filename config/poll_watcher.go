package config

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// defaultPollInterval is used when WatchModePoll is selected without an
// explicit watch_interval.
const defaultPollInterval = 5 * time.Second

// PollWatcher invokes onChange when the config file at path changes, detected
// by stat-ing the file on a fixed interval rather than subscribing to
// filesystem events. It's the portable fallback for environments where
// inotify/kqueue events aren't delivered for the config path (some network
// and container volume mounts).
type PollWatcher struct {
	path     string
	interval time.Duration
	onChange func()
}

// NewPollWatcher returns a PollWatcher. A non-positive interval falls back to
// defaultPollInterval. onChange runs synchronously on the watcher's goroutine
// and should not block for long.
func NewPollWatcher(path string, interval time.Duration, onChange func()) *PollWatcher {
	if interval <= 0 {
		interval = defaultPollInterval
	}
	return &PollWatcher{path: path, interval: interval, onChange: onChange}
}

// Start blocks until ctx is canceled, firing onChange whenever the file's
// fingerprint changes between polls.
func (w *PollWatcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	last, haveLast := fingerprint(w.path)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			cur, ok := fingerprint(w.path)
			if !ok {
				// Stat failed — likely a transient state mid-swap (e.g. a
				// Kubernetes ConfigMap ..data rename). Keep the previous
				// fingerprint so the next successful poll still detects the
				// change.
				continue
			}
			if !haveLast {
				last, haveLast = cur, true
				continue
			}
			if !cur.equal(last) {
				last = cur
				w.onChange()
			}
		}
	}
}

// fileFingerprint identifies a version of the config file. The resolved
// symlink target is included because a Kubernetes ConfigMap update swaps the
// ..data symlink to a freshly named directory — that target path changes even
// in the rare case where size and mtime collide.
type fileFingerprint struct {
	resolved string
	size     int64
	modTime  time.Time
}

func (a fileFingerprint) equal(b fileFingerprint) bool {
	return a.resolved == b.resolved && a.size == b.size && a.modTime.Equal(b.modTime)
}

// fingerprint stats path (following symlinks) and returns its fingerprint.
// The bool is false when the file can't be stat'd.
func fingerprint(path string) (fileFingerprint, bool) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Not a symlink (or broken mid-swap) — fall back to the raw path so a
		// plain regular file still fingerprints on size + mtime.
		resolved = path
	}
	info, err := os.Stat(path)
	if err != nil {
		return fileFingerprint{}, false
	}
	return fileFingerprint{resolved: resolved, size: info.Size(), modTime: info.ModTime()}, true
}
