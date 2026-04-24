package router

import (
	"kerbecs/provider"
	"testing"
)

type fakeProvider struct {
	name   string
	routes []provider.Route
}

func (f *fakeProvider) Name() string              { return f.name }
func (f *fakeProvider) Routes() []provider.Route { return f.routes }

func up(name string) *provider.Upstream {
	return &provider.Upstream{Name: name, Version: "1.0.0", Instances: []string{"http://" + name}}
}

func TestRouter_PathMatching(t *testing.T) {
	fp := &fakeProvider{name: "static", routes: []provider.Route{
		{Name: "users", Match: provider.RouteMatch{Path: "/users/*"}, Upstream: up("users")},
		{Name: "billing-exact", Match: provider.RouteMatch{Path: "exact:/billing"}, Upstream: up("billing")},
		{Name: "orders-regex", Match: provider.RouteMatch{Path: `regex:^/orders/\d+$`}, Upstream: up("orders")},
		{Name: "root", Match: provider.RouteMatch{Path: "/health"}, Upstream: up("health")},
	}}
	r, err := New(fp)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		path string
		want string // route name; "" = no match
	}{
		{"/users", "users"},
		{"/users/123", "users"},
		{"/users/abc/def", "users"},
		{"/billing", "billing-exact"},
		{"/billing/foo", ""},
		{"/orders/42", "orders-regex"},
		{"/orders/abc", ""},
		{"/health", "root"},
		{"/health/sub", ""},
		{"/nope", ""},
	}
	for _, c := range cases {
		m := r.Find("GET", "", c.path)
		got := ""
		if m != nil {
			got = m.Route.Name
		}
		if got != c.want {
			t.Errorf("path %q: want %q, got %q", c.path, c.want, got)
		}
	}
}

func TestRouter_MethodFiltering(t *testing.T) {
	fp := &fakeProvider{name: "static", routes: []provider.Route{
		{Name: "only-get", Match: provider.RouteMatch{Path: "/x", Methods: []string{"GET"}}, Upstream: up("x")},
	}}
	r, _ := New(fp)

	if r.Find("GET", "", "/x") == nil {
		t.Error("GET /x should match")
	}
	if r.Find("POST", "", "/x") != nil {
		t.Error("POST /x should not match")
	}
}

func TestRouter_HostFiltering(t *testing.T) {
	fp := &fakeProvider{name: "static", routes: []provider.Route{
		{Name: "api-only", Match: provider.RouteMatch{Path: "/x", Host: "api.example.com"}, Upstream: up("x")},
	}}
	r, _ := New(fp)

	if r.Find("GET", "api.example.com", "/x") == nil {
		t.Error("correct host should match")
	}
	if r.Find("GET", "other.example.com", "/x") != nil {
		t.Error("wrong host should not match")
	}
}

func TestRouter_FirstMatchWins(t *testing.T) {
	fp1 := &fakeProvider{name: "primary", routes: []provider.Route{
		{Name: "first", Match: provider.RouteMatch{Path: "/users/*"}, Upstream: up("a")},
	}}
	fp2 := &fakeProvider{name: "secondary", routes: []provider.Route{
		{Name: "second", Match: provider.RouteMatch{Path: "/users/*"}, Upstream: up("b")},
	}}
	r, _ := New(fp1, fp2)
	m := r.Find("GET", "", "/users/1")
	if m == nil || m.Route.Name != "first" {
		t.Errorf("want 'first' to win, got %+v", m)
	}
}

func TestRewritePath(t *testing.T) {
	cases := []struct {
		path string
		rw   *provider.Rewrite
		want string
	}{
		{"/users/123", nil, "/users/123"},
		{"/users/123", &provider.Rewrite{StripPrefix: "/users"}, "/123"},
		{"/users", &provider.Rewrite{StripPrefix: "/users"}, "/"},
		{"/users/123", &provider.Rewrite{ReplacePrefix: "/v1"}, "/v1/users/123"},
		{"/users/123", &provider.Rewrite{StripPrefix: "/other"}, "/users/123"}, // no match, unchanged
	}
	for _, c := range cases {
		got := RewritePath(c.path, c.rw)
		if got != c.want {
			t.Errorf("RewritePath(%q, %+v) = %q, want %q", c.path, c.rw, got, c.want)
		}
	}
}

func TestRouter_RegexCompileError(t *testing.T) {
	fp := &fakeProvider{name: "static", routes: []provider.Route{
		{Name: "bad", Match: provider.RouteMatch{Path: "regex:[invalid"}, Upstream: up("x")},
	}}
	if _, err := New(fp); err == nil {
		t.Error("expected compile error for invalid regex")
	}
}
