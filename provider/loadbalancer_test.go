package provider

import (
	"sync"
	"testing"
)

func TestRoundRobin_CyclesThroughInstances(t *testing.T) {
	lb, err := newLoadBalancer("round_robin", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "c", "a", "b", "c", "a"}
	for i, w := range want {
		if got := lb.Pick(); got != w {
			t.Errorf("pick %d: got %q, want %q", i, got, w)
		}
	}
}

func TestRoundRobin_Concurrent(t *testing.T) {
	lb, _ := newLoadBalancer("round_robin", []string{"a", "b", "c", "d"})
	counts := make(map[string]int, 4)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pick := lb.Pick()
			mu.Lock()
			counts[pick]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	// With atomic round-robin, each instance should get exactly 1000/4 = 250
	// picks across concurrent callers.
	for _, inst := range []string{"a", "b", "c", "d"} {
		if counts[inst] != 250 {
			t.Errorf("instance %q: got %d picks, want 250", inst, counts[inst])
		}
	}
}

func TestRandom_PicksFromPool(t *testing.T) {
	pool := []string{"a", "b", "c"}
	seen := map[string]bool{}
	lb, _ := newLoadBalancer("random", pool)
	for i := 0; i < 200; i++ {
		seen[lb.Pick()] = true
	}
	for _, inst := range pool {
		if !seen[inst] {
			t.Errorf("instance %q never picked across 200 calls", inst)
		}
	}
}

func TestEmptyStrategy_DefaultsToRoundRobin(t *testing.T) {
	lb, err := newLoadBalancer("", []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if lb.Pick() != "a" || lb.Pick() != "b" || lb.Pick() != "a" {
		t.Error("empty strategy should behave as round_robin")
	}
}

func TestUnknownStrategy_Errors(t *testing.T) {
	if _, err := newLoadBalancer("weighted", []string{"a"}); err == nil {
		t.Error("expected error for unknown strategy")
	}
}
