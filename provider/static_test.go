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

func TestNewStatic_RejectsEmptyInstances(t *testing.T) {
	c := baseConfig()
	u := c.Upstreams["users"]
	u.Instances = nil
	c.Upstreams["users"] = u
	if _, err := NewStatic(c); err == nil {
		t.Error("expected error for empty instances list")
	}
}

func TestNewStatic_RejectsUnknownLoadBalancer(t *testing.T) {
	c := baseConfig()
	u := c.Upstreams["users"]
	u.LoadBalancer = "weighted"
	c.Upstreams["users"] = u
	if _, err := NewStatic(c); err == nil {
		t.Error("expected error for unknown load balancer")
	}
}

func TestNewStatic_TimeoutsResolution(t *testing.T) {
	c := baseConfig()
	c.Gateway.Timeouts = config.Timeouts{
		Dial:    config.Duration(5 * time.Second),
		Headers: config.Duration(30 * time.Second),
		Overall: config.Duration(15 * time.Second),
		Idle:    config.Duration(90 * time.Second),
	}
	u := c.Upstreams["users"]
	u.Timeouts = config.Timeouts{
		Headers: config.Duration(60 * time.Second), // override headers
	}
	c.Upstreams["users"] = u
	c.Routes[0].Timeouts = &config.RouteTimeouts{
		Overall: config.Duration(45 * time.Second), // override overall per-route
	}

	s, err := NewStatic(c)
	if err != nil {
		t.Fatal(err)
	}
	r := s.Routes()[0]

	if got := r.Upstream.Timeouts.Dial; got != 5*time.Second {
		t.Errorf("dial: got %v, want 5s (inherited from global)", got)
	}
	if got := r.Upstream.Timeouts.Headers; got != 60*time.Second {
		t.Errorf("headers: got %v, want 60s (upstream override)", got)
	}
	if got := r.Upstream.Timeouts.Idle; got != 90*time.Second {
		t.Errorf("idle: got %v, want 90s (inherited from global)", got)
	}
	if got := r.OverallTimeout; got != 45*time.Second {
		t.Errorf("overall: got %v, want 45s (route override)", got)
	}
}

func TestNewStatic_OverallFallsThroughToUpstream(t *testing.T) {
	c := baseConfig()
	u := c.Upstreams["users"]
	u.Timeouts = config.Timeouts{Overall: config.Duration(20 * time.Second)}
	c.Upstreams["users"] = u
	// no per-route override

	s, err := NewStatic(c)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Routes()[0].OverallTimeout; got != 20*time.Second {
		t.Errorf("overall should inherit upstream value: got %v, want 20s", got)
	}
}

func TestNewStatic_RoundRobinPicksAcrossInstances(t *testing.T) {
	c := baseConfig()
	u := c.Upstreams["users"]
	u.Instances = []string{"http://a:80", "http://b:80", "http://c:80"}
	c.Upstreams["users"] = u
	s, err := NewStatic(c)
	if err != nil {
		t.Fatal(err)
	}
	up := s.Routes()[0].Upstream
	got := []string{up.Pick(), up.Pick(), up.Pick(), up.Pick()}
	want := []string{"http://a:80", "http://b:80", "http://c:80", "http://a:80"}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("pick %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
