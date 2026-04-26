package gateway

import (
	"kerbecs/router"
	"net/http"
	"sync/atomic"
)

// LiveState is the atomically-swappable runtime data the proxy handler reads
// per request: the route table and the per-upstream HTTP transport cache.
// Both pieces must be updated together so a request never sees a route from
// generation N alongside transports from generation N-1.
type LiveState struct {
	Router     *router.Router
	Transports map[string]*http.Transport
}

// BuildState constructs a LiveState from a router. Transport cache is built
// from the unique upstreams in the router's current route table.
func BuildState(rt *router.Router) *LiveState {
	transports := map[string]*http.Transport{}
	for _, up := range rt.Upstreams() {
		transports[up.Name] = buildTransport(up.Timeouts)
	}
	return &LiveState{Router: rt, Transports: transports}
}

// StatePointer is the atomic pointer the listener reads from. main owns the
// pointer; the watcher updates it on config reload.
type StatePointer = atomic.Pointer[LiveState]
