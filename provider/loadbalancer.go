package provider

import (
	"fmt"
	"math/rand/v2"
	"sync/atomic"
)

// LoadBalancer picks one instance URL from an upstream's pool. Implementations
// must be safe for concurrent callers — Pick is invoked from every proxied
// request.
type LoadBalancer interface {
	Pick() string
}

// newLoadBalancer builds a LoadBalancer for the given strategy and instance
// list. Empty strategy defaults to round-robin.
func newLoadBalancer(strategy string, instances []string) (LoadBalancer, error) {
	switch strategy {
	case "", "round_robin":
		return &roundRobin{instances: instances}, nil
	case "random":
		return &randomLB{instances: instances}, nil
	default:
		return nil, fmt.Errorf("unknown load_balancer strategy %q (supported: round_robin, random)", strategy)
	}
}

type roundRobin struct {
	instances []string
	idx       atomic.Uint64
}

func (r *roundRobin) Pick() string {
	n := r.idx.Add(1) - 1
	return r.instances[n%uint64(len(r.instances))]
}

type randomLB struct {
	instances []string
}

func (r *randomLB) Pick() string {
	return r.instances[rand.IntN(len(r.instances))]
}
