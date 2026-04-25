package provider

import (
	"fmt"
	"kerbecs/config"
	"kerbecs/pkg/middleware"
)

// Defaults applied when neither global nor route limits are set.
const (
	defaultMaxRequestBytes  int64 = 100 << 20 // 100 MiB
	defaultMaxResponseBytes int64 = 100 << 20 // 100 MiB
)

// Static is a Provider backed by the parsed config file. Routes are resolved
// once at construction; hot reload is deferred to a later phase.
type Static struct {
	routes []Route
}

func NewStatic(f *config.File, mws *middleware.Registry) (*Static, error) {
	globalLimits := Limits{
		MaxRequestBytes:  nonZeroInt64(f.Gateway.Limits.MaxRequestBytes.Bytes(), defaultMaxRequestBytes),
		MaxResponseBytes: nonZeroInt64(f.Gateway.Limits.MaxResponseBytes.Bytes(), defaultMaxResponseBytes),
	}

	upstreams := make(map[string]*Upstream, len(f.Upstreams))
	for name, u := range f.Upstreams {
		if len(u.Instances) == 0 {
			return nil, fmt.Errorf("upstream %q: at least one instance is required", name)
		}
		instances := append([]string(nil), u.Instances...)
		lb, err := newLoadBalancer(u.LoadBalancer, instances)
		if err != nil {
			return nil, fmt.Errorf("upstream %q: %w", name, err)
		}
		upstreams[name] = &Upstream{
			Name:         u.Name,
			Version:      u.Version,
			Instances:    instances,
			LoadBalancer: u.LoadBalancer,
			HealthCheck:  convertHealthCheck(u.HealthCheck),
			Timeouts: Timeouts{
				Dial:           u.Timeouts.Dial.AsDuration(),
				ResponseHeader: u.Timeouts.ResponseHeader.AsDuration(),
				Overall:        u.Timeouts.Overall.AsDuration(),
				Idle:           u.Timeouts.Idle.AsDuration(),
			},
			lb: lb,
		}
	}

	routes := make([]Route, 0, len(f.Routes))
	for i, r := range f.Routes {
		if r.Upstream == "" {
			return nil, fmt.Errorf("route %q (index %d): upstream is required", r.Name, i)
		}
		up, ok := upstreams[r.Upstream]
		if !ok {
			return nil, fmt.Errorf("route %q: references unknown upstream %q", r.Name, r.Upstream)
		}
		env, err := resolveEnvelope(r.Envelope)
		if err != nil {
			return nil, fmt.Errorf("route %q: %w", r.Name, err)
		}
		if r.Match.Path == "" {
			return nil, fmt.Errorf("route %q: match.path is required", r.Name)
		}
		chain, err := mws.Chain(r.Middlewares)
		if err != nil {
			return nil, fmt.Errorf("route %q: %w", r.Name, err)
		}
		routes = append(routes, Route{
			Name:        r.Name,
			Match:       RouteMatch{Path: r.Match.Path, Methods: append([]string(nil), r.Match.Methods...), Host: r.Match.Host},
			Upstream:    up,
			Rewrite:     convertRewrite(r.Rewrite),
			Envelope:    env,
			Limits:      mergeLimits(globalLimits, r.Limits),
			Middlewares: chain,
		})
	}
	return &Static{routes: routes}, nil
}

func mergeLimits(global Limits, override *config.Limits) Limits {
	out := global
	if override == nil {
		return out
	}
	if v := override.MaxRequestBytes.Bytes(); v > 0 {
		out.MaxRequestBytes = v
	}
	if v := override.MaxResponseBytes.Bytes(); v > 0 {
		out.MaxResponseBytes = v
	}
	return out
}

func nonZeroInt64(v, fallback int64) int64 {
	if v <= 0 {
		return fallback
	}
	return v
}

func (s *Static) Name() string     { return "static" }
func (s *Static) Routes() []Route  { return s.routes }

func convertHealthCheck(hc *config.HealthCheck) *HealthCheck {
	if hc == nil {
		return nil
	}
	return &HealthCheck{
		Path:     hc.Path,
		Interval: hc.Interval.AsDuration(),
		Timeout:  hc.Timeout.AsDuration(),
	}
}

func convertRewrite(r *config.Rewrite) *Rewrite {
	if r == nil {
		return nil
	}
	return &Rewrite{StripPrefix: r.StripPrefix, ReplacePrefix: r.ReplacePrefix}
}

// resolveEnvelope maps config envelope names to runtime modes.
// Phase 1 only supports the two built-ins; custom templates land later.
func resolveEnvelope(name string) (EnvelopeMode, error) {
	switch name {
	case "", "default":
		return EnvelopeDefault, nil
	case "passthrough":
		return EnvelopePassthrough, nil
	default:
		return 0, fmt.Errorf("unknown envelope %q (phase 1 supports only 'default' and 'passthrough')", name)
	}
}
