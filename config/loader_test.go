package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kerbecs.yaml")
	yaml := `
gateway:
  name: test-gw
  version: 9.9.9
  env: ${TEST_ENV:DEV}

listeners:
  gateway: { port: "${TEST_PORT:10310}" }
  admin:
    port: "10300"
    auth: { type: basic, username: ${TEST_USER}, password: ${TEST_PASS:fallback} }

providers:
  static: { watch: true }

upstreams:
  users:
    name: users
    version: 1.0.0
    instances: [http://users:8080]
    timeouts:
      dial: 2s
      response_header: 5s
      overall: 30s
      idle: 90s

routes:
  - name: users-api
    match: { path: /users/*, methods: [GET, POST] }
    upstream: users
    envelope: default
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TEST_USER", "admin")
	// TEST_PORT and TEST_ENV deliberately unset to exercise defaults.
	// TEST_PASS deliberately unset to fall through to "fallback".

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}

	if cfg.Gateway.Env != "DEV" {
		t.Errorf("env: want DEV, got %q", cfg.Gateway.Env)
	}
	if cfg.Listeners.Gateway.Port != "10310" {
		t.Errorf("gateway port: want 10310, got %q", cfg.Listeners.Gateway.Port)
	}
	if cfg.Listeners.Admin.Auth.Username != "admin" {
		t.Errorf("admin user: want admin, got %q", cfg.Listeners.Admin.Auth.Username)
	}
	if cfg.Listeners.Admin.Auth.Password != "fallback" {
		t.Errorf("admin pass: want fallback, got %q", cfg.Listeners.Admin.Auth.Password)
	}

	u, ok := cfg.Upstreams["users"]
	if !ok {
		t.Fatal("upstream 'users' missing")
	}
	if got := u.Timeouts.Dial.AsDuration(); got != 2*time.Second {
		t.Errorf("dial timeout: want 2s, got %v", got)
	}
	if got := u.Timeouts.Overall.AsDuration(); got != 30*time.Second {
		t.Errorf("overall timeout: want 30s, got %v", got)
	}

	if len(cfg.Routes) != 1 || cfg.Routes[0].Upstream != "users" {
		t.Errorf("routes: %+v", cfg.Routes)
	}
	if cfg.Routes[0].Match.Path != "/users/*" {
		t.Errorf("route path: %q", cfg.Routes[0].Match.Path)
	}
}

func TestSize_UnmarshalYAML(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{`100MB`, 100 << 20},
		{`500KB`, 500 << 10},
		{`1GiB`, 1 << 30},
		{`1024`, 1024},
		{`1.5MB`, int64(1.5 * float64(1<<20))},
		{`10 M`, 10 << 20},
		{`2 B`, 2},
	}
	for _, c := range cases {
		dir := t.TempDir()
		path := filepath.Join(dir, "s.yaml")
		if err := os.WriteFile(path, []byte("size: "+c.in), 0o600); err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Size Size `yaml:"size"`
		}
		raw, _ := os.ReadFile(path)
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if doc.Size.Bytes() != c.want {
			t.Errorf("%q: got %d, want %d", c.in, doc.Size.Bytes(), c.want)
		}
	}

	// Invalid units should fail.
	var doc struct {
		Size Size `yaml:"size"`
	}
	if err := yaml.Unmarshal([]byte("size: 10QQ"), &doc); err == nil {
		t.Errorf("expected error for invalid unit")
	}
}

func TestExpandEnv(t *testing.T) {
	t.Setenv("FOO", "bar")

	cases := []struct {
		in, want string
	}{
		{"${FOO}", "bar"},
		{"${MISSING}", ""},
		{"${MISSING:default}", "default"},
		{"${FOO:ignored}", "bar"},
		{"plain text", "plain text"},
		{"${FOO}/${MISSING:x}", "bar/x"},
	}
	for _, c := range cases {
		got := string(expandEnv([]byte(c.in)))
		if got != c.want {
			t.Errorf("expandEnv(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
