package middleware

import (
	"fmt"
	"kerbecs/config"

	"github.com/gin-gonic/gin"
)

// Registry holds resolved middleware instances keyed by their config name
// (e.g. "req-id", "strip-headers"). Built once at startup from the loaded
// config file.
type Registry struct {
	instances map[string]Middleware
}

// BuildRegistry constructs a Middleware for every entry in defs and returns
// them keyed by name. Unknown types or decode errors fail-fast at config
// load.
func BuildRegistry(defs map[string]config.Middleware) (*Registry, error) {
	instances := make(map[string]Middleware, len(defs))
	for name, def := range defs {
		m, err := Build(def.Type, def.Decode)
		if err != nil {
			return nil, fmt.Errorf("middleware %q: %w", name, err)
		}
		instances[name] = m
	}
	return &Registry{instances: instances}, nil
}

// Get returns the Middleware registered under name.
func (r *Registry) Get(name string) (Middleware, bool) {
	if r == nil {
		return nil, false
	}
	m, ok := r.instances[name]
	return m, ok
}

// Chain resolves a list of middleware names to gin handlers in the order
// given. Returns an error on the first unknown name.
func (r *Registry) Chain(names []string) ([]gin.HandlerFunc, error) {
	if r == nil || len(names) == 0 {
		return nil, nil
	}
	out := make([]gin.HandlerFunc, 0, len(names))
	for _, name := range names {
		m, ok := r.instances[name]
		if !ok {
			return nil, fmt.Errorf("unknown middleware %q", name)
		}
		out = append(out, m.Handler())
	}
	return out, nil
}
