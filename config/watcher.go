package config

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounce is how long to wait after the last filesystem event before firing
// the callback. Editors and atomic-write tools fire several events for a
// single logical save (write + rename + chmod), so we coalesce them.
const debounce = 100 * time.Millisecond

// FileWatcher invokes a callback when the config file changes, until its
// context is canceled. Both the fsnotify-backed Watcher and the interval-based
// PollWatcher implement it.
type FileWatcher interface {
	Start(ctx context.Context) error
}

// NewConfigWatcher builds the FileWatcher selected by the static provider
// config — fsnotify file events (WatchModeFile, the default) or interval
// polling (WatchModePoll). Mode and interval defaults are applied by
// ApplyDefaults, so cfg is expected to be normalized here.
func NewConfigWatcher(cfg StaticProviderConfig, path string, onChange func()) FileWatcher {
	if cfg.WatchMode == WatchModePoll {
		return NewPollWatcher(path, cfg.WatchInterval.AsDuration(), onChange)
	}
	return NewWatcher(path, onChange)
}

// Watcher invokes onChange whenever the config file at path is modified.
//
// We watch the parent directory rather than the file itself so editor
// save-via-rename and Kubernetes ConfigMap symlink swaps both fire correctly
// — fsnotify on a single file misses both of those patterns.
type Watcher struct {
	path     string
	onChange func()
}

// NewWatcher returns a Watcher that fires onChange when path is modified.
// onChange runs synchronously on the watcher's goroutine; it should not
// block for long.
func NewWatcher(path string, onChange func()) *Watcher {
	return &Watcher{path: path, onChange: onChange}
}

// Start blocks until ctx is canceled. Errors during startup are returned;
// errors during operation are logged via the callback returning normally
// (caller decides how to handle parse failures).
func (w *Watcher) Start(ctx context.Context) error {
	dir := filepath.Dir(w.path)
	target, err := filepath.Abs(w.path)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	defer fsw.Close()

	if err := fsw.Add(dir); err != nil {
		return fmt.Errorf("watch %s: %w", dir, err)
	}

	var timer *time.Timer
	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-fsw.Events:
			if !ok {
				return nil
			}
			eventPath, _ := filepath.Abs(event.Name)
			// Care about events on our exact file, plus K8s ConfigMap
			// re-mounts (the symlinked '..data' directory swap).
			if eventPath != target && filepath.Base(event.Name) != "..data" {
				continue
			}
			if timer == nil {
				timer = time.AfterFunc(debounce, w.onChange)
			} else {
				timer.Reset(debounce)
			}

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("fsnotify error: %w", err)
		}
	}
}
