package provider

import "time"

// Provider is a source of routes. A StaticProvider reads from a config file;
// future providers (Rincon, Docker, Kubernetes) will read from elsewhere.
//
// Routes returns a snapshot of the current routes. Callers must not mutate
// the returned slice.
type Provider interface {
	Name() string
	Routes() []Route
}

// Route is the runtime shape of a routing rule. Upstream is a resolved pointer
// to avoid name lookups in the hot path.
type Route struct {
	Name        string
	Match       RouteMatch
	Upstream    *Upstream
	Rewrite     *Rewrite
	Envelope    EnvelopeMode
	Limits      Limits
	Middlewares []string
}

// Limits are the fully-resolved byte caps for a route (global default merged
// with any per-route override).
type Limits struct {
	MaxRequestBytes  int64
	MaxResponseBytes int64
}

// RouteMatch carries unparsed match parameters; the actual matcher is built
// by the router.
type RouteMatch struct {
	Path    string
	Methods []string
	Host    string
}

type EnvelopeMode int

const (
	EnvelopeDefault EnvelopeMode = iota
	EnvelopePassthrough
)

type Rewrite struct {
	StripPrefix   string
	ReplacePrefix string
}

type Upstream struct {
	Name         string
	Version      string
	Instances    []string
	LoadBalancer string
	HealthCheck  *HealthCheck
	Timeouts     Timeouts

	lb LoadBalancer
}

func (u *Upstream) FormattedNameWithVersion() string {
	return u.Name + ":v" + u.Version
}

// Pick returns an instance URL chosen by the upstream's load balancer.
// Callers can assume the result is non-empty — NewStatic rejects upstreams
// with empty instance lists at config-load time.
func (u *Upstream) Pick() string {
	return u.lb.Pick()
}

type HealthCheck struct {
	Path     string
	Interval time.Duration
	Timeout  time.Duration
}

type Timeouts struct {
	Dial           time.Duration
	ResponseHeader time.Duration
	Overall        time.Duration
	Idle           time.Duration
}
