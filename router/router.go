package router

import (
	"fmt"
	"kerbecs/provider"
	"regexp"
	"strings"
)

// Router holds compiled routes from one or more providers and finds the first
// match for an incoming request.
type Router struct {
	routes []compiled
}

// Match is returned by Find; it exposes the matched route and carries the
// compiled rewrite helper so callers don't recompute the transform.
type Match struct {
	Route *provider.Route
}

type compiled struct {
	route *provider.Route
	match func(method, host, path string) bool
}

// New builds a Router from the routes produced by the given providers, in
// order. Earlier providers take precedence on conflict (first match wins).
func New(providers ...provider.Provider) (*Router, error) {
	var compiledRoutes []compiled
	for _, p := range providers {
		for i, r := range p.Routes() {
			r := r
			m, err := compileMatch(r.Match)
			if err != nil {
				return nil, fmt.Errorf("provider %q route %q (index %d): %w", p.Name(), r.Name, i, err)
			}
			compiledRoutes = append(compiledRoutes, compiled{route: &r, match: m})
		}
	}
	return &Router{routes: compiledRoutes}, nil
}

// Upstreams returns each distinct upstream referenced by at least one route,
// preserving first-occurrence order across providers.
func (r *Router) Upstreams() []*provider.Upstream {
	seen := map[string]bool{}
	var out []*provider.Upstream
	for _, c := range r.routes {
		if c.route.Upstream == nil {
			continue
		}
		name := c.route.Upstream.Name
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, c.route.Upstream)
	}
	return out
}

// Find returns the first matching route, or nil if none matches.
func (r *Router) Find(method, host, path string) *Match {
	for _, c := range r.routes {
		if c.match(method, host, path) {
			return &Match{Route: c.route}
		}
	}
	return nil
}

// RewritePath applies the route's rewrite (if any) to the incoming path.
func RewritePath(path string, rw *provider.Rewrite) string {
	if rw == nil {
		return path
	}
	if rw.StripPrefix != "" && strings.HasPrefix(path, rw.StripPrefix) {
		stripped := strings.TrimPrefix(path, rw.StripPrefix)
		if stripped == "" {
			return "/"
		}
		if !strings.HasPrefix(stripped, "/") {
			stripped = "/" + stripped
		}
		return stripped
	}
	if rw.ReplacePrefix != "" {
		// replace_prefix prepends; it does not strip.
		if !strings.HasPrefix(rw.ReplacePrefix, "/") {
			return "/" + rw.ReplacePrefix + path
		}
		return rw.ReplacePrefix + path
	}
	return path
}

// compileMatch turns a RouteMatch into a predicate over (method, host, path).
//
// Path syntax:
//   - "exact:/foo"       exact match
//   - "regex:<pattern>"  regex match
//   - "/foo/*"           prefix match (trailing /* stripped)
//   - "/foo"             exact match (no special suffix)
func compileMatch(m provider.RouteMatch) (func(string, string, string) bool, error) {
	pathFn, err := compilePath(m.Path)
	if err != nil {
		return nil, err
	}

	methodSet := make(map[string]struct{}, len(m.Methods))
	for _, method := range m.Methods {
		methodSet[strings.ToUpper(method)] = struct{}{}
	}
	host := m.Host

	return func(method, reqHost, reqPath string) bool {
		if len(methodSet) > 0 {
			if _, ok := methodSet[strings.ToUpper(method)]; !ok {
				return false
			}
		}
		if host != "" && host != reqHost {
			return false
		}
		return pathFn(reqPath)
	}, nil
}

func compilePath(pattern string) (func(string) bool, error) {
	switch {
	case strings.HasPrefix(pattern, "regex:"):
		re, err := regexp.Compile(pattern[len("regex:"):])
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		return func(p string) bool { return re.MatchString(p) }, nil

	case strings.HasPrefix(pattern, "exact:"):
		exact := pattern[len("exact:"):]
		return func(p string) bool { return p == exact }, nil

	case strings.HasSuffix(pattern, "/*"):
		prefix := strings.TrimSuffix(pattern, "/*")
		return func(p string) bool {
			return p == prefix || strings.HasPrefix(p, prefix+"/")
		}, nil

	default:
		exact := pattern
		return func(p string) bool { return p == exact }, nil
	}
}
