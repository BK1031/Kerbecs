package gateway

import (
	"kerbecs/provider"
	"kerbecs/router"
	"net"
	"net/http"
	"time"
)

// Default timeout values applied when neither global, upstream, nor route
// config sets one. These are conservative — they favor protecting the
// gateway over allowing very slow upstreams.
const (
	defaultDialTimeout    = 5 * time.Second
	defaultHeadersTimeout = 30 * time.Second
	defaultIdleConnTimeout = 90 * time.Second
)

// buildTransportCache returns a map from upstream name to a dedicated
// *http.Transport, so each upstream gets its own connection pool and timeout
// profile.
func buildTransportCache(rt *router.Router) map[string]*http.Transport {
	out := map[string]*http.Transport{}
	for _, up := range rt.Upstreams() {
		out[up.Name] = buildTransport(up.Timeouts)
	}
	return out
}

func buildTransport(t provider.Timeouts) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   nonZero(t.Dial, defaultDialTimeout),
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		DialContext:           dialer.DialContext,
		ResponseHeaderTimeout: nonZero(t.Headers, defaultHeadersTimeout),
		IdleConnTimeout:       nonZero(t.Idle, defaultIdleConnTimeout),
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		ForceAttemptHTTP2:     true,
	}
}

func nonZero(v, fallback time.Duration) time.Duration {
	if v <= 0 {
		return fallback
	}
	return v
}
