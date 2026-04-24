package provider

import (
	"kerbecs/config"
	"testing"
	"time"
)

func baseConfig() *config.File {
	return &config.File{
		Upstreams: map[string]config.Upstream{
			"users": {
				Name:      "users",
				Version:   "1.0.0",
				Instances: []string{"http://users:8080"},
				Timeouts:  config.Timeouts{Dial: config.Duration(2 * time.Second)},
			},
		},
		Routes: []config.Route{
			{
				Name:     "users-api",
				Match:    config.RouteMatch{Path: "/users/*", Methods: []string{"GET"}},
				Upstream: "users",
				Envelope: "default",
			},
		},
	}
}

func TestNewStatic_Resolves(t *testing.T) {
	s, err := NewStatic(baseConfig())
	if err != nil {
		t.Fatal(err)
	}
	routes := s.Routes()
	if len(routes) != 1 {
		t.Fatalf("want 1 route, got %d", len(routes))
	}
	r := routes[0]
	if r.Upstream == nil || r.Upstream.Name != "users" {
		t.Errorf("upstream not resolved: %+v", r.Upstream)
	}
	if r.Upstream.Timeouts.Dial != 2*time.Second {
		t.Errorf("dial timeout: %v", r.Upstream.Timeouts.Dial)
	}
	if r.Envelope != EnvelopeDefault {
		t.Errorf("envelope: want default, got %v", r.Envelope)
	}
}

func TestNewStatic_UnknownUpstream(t *testing.T) {
	c := baseConfig()
	c.Routes[0].Upstream = "ghost"
	if _, err := NewStatic(c); err == nil {
		t.Error("expected error for unknown upstream")
	}
}

func TestNewStatic_UnknownEnvelope(t *testing.T) {
	c := baseConfig()
	c.Routes[0].Envelope = "my-custom"
	if _, err := NewStatic(c); err == nil {
		t.Error("expected error for unknown envelope")
	}
}

func TestNewStatic_EnvelopeDefaults(t *testing.T) {
	c := baseConfig()
	c.Routes[0].Envelope = ""
	s, err := NewStatic(c)
	if err != nil {
		t.Fatal(err)
	}
	if s.Routes()[0].Envelope != EnvelopeDefault {
		t.Error("empty envelope should default to EnvelopeDefault")
	}
}

func TestNewStatic_Passthrough(t *testing.T) {
	c := baseConfig()
	c.Routes[0].Envelope = "passthrough"
	s, err := NewStatic(c)
	if err != nil {
		t.Fatal(err)
	}
	if s.Routes()[0].Envelope != EnvelopePassthrough {
		t.Error("envelope: want passthrough")
	}
}

func TestNewStatic_MissingPath(t *testing.T) {
	c := baseConfig()
	c.Routes[0].Match.Path = ""
	if _, err := NewStatic(c); err == nil {
		t.Error("expected error for missing path")
	}
}
