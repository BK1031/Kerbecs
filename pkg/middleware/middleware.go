// Package middleware defines the runtime model for Kerbecs's pluggable
// request middlewares: an interface, a factory contract, and a registry that
// turns config-defined middleware blocks into runnable gin handlers.
//
// Each middleware "type" (e.g. request_id, headers) registers a Factory at
// startup via Register. Factories receive a decoder bound to the middleware's
// own YAML node so they can decode into a typed config without fishing
// through map[string]any.
package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// Middleware is something with a single gin handler attached. The Handler is
// what gets installed in a route or listener chain.
type Middleware interface {
	Handler() gin.HandlerFunc
}

// Factory builds a Middleware instance from its config. The decode function
// is a bound method on config.Middleware that decodes the full YAML node
// (including type-specific fields) into the factory's typed config struct.
type Factory func(decode func(any) error) (Middleware, error)

var registry = map[string]Factory{}

// Register installs a Factory under typeName. Calling twice for the same name
// panics — middleware types are expected to be unique and known at compile
// time.
func Register(typeName string, factory Factory) {
	if _, exists := registry[typeName]; exists {
		panic("middleware type already registered: " + typeName)
	}
	registry[typeName] = factory
}

// Build constructs a Middleware for the given typeName, returning an error if
// the type is unknown or the factory fails to decode its config.
func Build(typeName string, decode func(any) error) (Middleware, error) {
	factory, ok := registry[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown middleware type %q", typeName)
	}
	return factory(decode)
}

// Reset clears the registry. Test-only helper.
func Reset() {
	registry = map[string]Factory{}
}
