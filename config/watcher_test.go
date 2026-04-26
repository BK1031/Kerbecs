package config

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatcher_FiresOnChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kerbecs.yaml")
	if err := os.WriteFile(path, []byte("gateway: { name: a }\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var fired atomic.Int32
	w := NewWatcher(path, func() { fired.Add(1) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.Start(ctx) }()

	// Give the watcher a moment to install the inotify/kqueue handle before
	// we make any changes.
	time.Sleep(50 * time.Millisecond)

	if err := os.WriteFile(path, []byte("gateway: { name: b }\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Wait long enough for debounce + a margin.
	deadline := time.Now().Add(2 * time.Second)
	for fired.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	if fired.Load() == 0 {
		t.Fatal("expected onChange to fire after file write")
	}
}

func TestWatcher_DebouncesBurst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kerbecs.yaml")
	if err := os.WriteFile(path, []byte("v: 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var fired atomic.Int32
	w := NewWatcher(path, func() { fired.Add(1) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.Start(ctx) }()

	time.Sleep(50 * time.Millisecond)

	// Five rapid writes within the debounce window should produce one fire.
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(path, []byte{byte('0' + i), '\n'}, 0o600); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Wait past the debounce window.
	time.Sleep(300 * time.Millisecond)

	got := fired.Load()
	if got != 1 {
		t.Errorf("expected 1 fire after burst, got %d", got)
	}
}

func TestWatcher_IgnoresUnrelatedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kerbecs.yaml")
	other := filepath.Join(dir, "other.txt")
	if err := os.WriteFile(path, []byte("v: 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var fired atomic.Int32
	w := NewWatcher(path, func() { fired.Add(1) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.Start(ctx) }()

	time.Sleep(50 * time.Millisecond)

	// Touch an unrelated file in the same directory.
	if err := os.WriteFile(other, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)

	if fired.Load() != 0 {
		t.Errorf("expected no fires for unrelated file, got %d", fired.Load())
	}
}
