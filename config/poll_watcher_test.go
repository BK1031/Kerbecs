package config

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestPollWatcher_FiresOnChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kerbecs.yaml")
	if err := os.WriteFile(path, []byte("v: 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var fired atomic.Int32
	w := NewPollWatcher(path, 20*time.Millisecond, func() { fired.Add(1) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.Start(ctx) }()

	// Let the watcher capture the initial fingerprint before we change it.
	time.Sleep(50 * time.Millisecond)

	// Different size and content so the change is detected regardless of mtime
	// granularity on the test filesystem.
	if err := os.WriteFile(path, []byte("v: 123456789\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for fired.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	if fired.Load() == 0 {
		t.Fatal("expected onChange to fire after file write")
	}
}

func TestPollWatcher_IgnoresUnrelatedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kerbecs.yaml")
	other := filepath.Join(dir, "other.txt")
	if err := os.WriteFile(path, []byte("v: 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var fired atomic.Int32
	w := NewPollWatcher(path, 20*time.Millisecond, func() { fired.Add(1) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.Start(ctx) }()

	time.Sleep(50 * time.Millisecond)

	if err := os.WriteFile(other, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)

	if fired.Load() != 0 {
		t.Errorf("expected no fires for unrelated file, got %d", fired.Load())
	}
}

// A ConfigMap update swaps a ..data symlink to a new target directory. The
// fingerprint includes the resolved path, so it should fire even if the new
// file's size and mtime happen to match the old one.
func TestPollWatcher_DetectsSymlinkSwap(t *testing.T) {
	dir := t.TempDir()
	dataA := filepath.Join(dir, "a")
	dataB := filepath.Join(dir, "b")
	for _, d := range []string{dataA, dataB} {
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "kerbecs.yaml"), []byte("v: same\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	link := filepath.Join(dir, "..data")
	if err := os.Symlink(dataA, link); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(link, "kerbecs.yaml")

	var fired atomic.Int32
	w := NewPollWatcher(path, 20*time.Millisecond, func() { fired.Add(1) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.Start(ctx) }()

	time.Sleep(50 * time.Millisecond)

	// Atomically repoint the symlink at the other directory (rename-over).
	tmp := filepath.Join(dir, ".data-tmp")
	if err := os.Symlink(dataB, tmp); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, link); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for fired.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	if fired.Load() == 0 {
		t.Fatal("expected onChange to fire after symlink swap")
	}
}

func TestNewConfigWatcher_SelectsMode(t *testing.T) {
	noop := func() {}
	if _, ok := NewConfigWatcher(StaticProviderConfig{Watch: true, WatchMode: WatchModePoll, WatchInterval: Duration(time.Second)}, "kerbecs.yaml", noop).(*PollWatcher); !ok {
		t.Error("expected *PollWatcher for watch_mode: poll")
	}
	if _, ok := NewConfigWatcher(StaticProviderConfig{Watch: true, WatchMode: WatchModeFile}, "kerbecs.yaml", noop).(*Watcher); !ok {
		t.Error("expected *Watcher for watch_mode: file")
	}
	if _, ok := NewConfigWatcher(StaticProviderConfig{Watch: true}, "kerbecs.yaml", noop).(*Watcher); !ok {
		t.Error("expected *Watcher when watch_mode is unset")
	}
}
