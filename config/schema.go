package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// File is the root of kerbecs.yaml.
type File struct {
	Gateway       GatewaySection        `yaml:"gateway"`
	Listeners     ListenersSection      `yaml:"listeners"`
	Providers     ProvidersSection      `yaml:"providers"`
	Upstreams     map[string]Upstream   `yaml:"upstreams"`
	Middlewares   map[string]Middleware `yaml:"middlewares"`
	Envelopes     map[string]Envelope   `yaml:"envelopes"`
	Routes        []Route               `yaml:"routes"`
	Observability ObservabilitySection  `yaml:"observability"`
}

type GatewaySection struct {
	Name     string   `yaml:"name"`
	Version  string   `yaml:"version"`
	Env      string   `yaml:"env"`
	Limits   Limits   `yaml:"limits"`
	Timeouts Timeouts `yaml:"timeouts"`
}

// Limits caps the size of request and response bodies. At the gateway level
// these are global defaults; per-route limits override.
type Limits struct {
	MaxRequestBytes  Size `yaml:"max_request_bytes,omitempty"`
	MaxResponseBytes Size `yaml:"max_response_bytes,omitempty"`
}

type ListenersSection struct {
	Gateway GatewayListener `yaml:"gateway"`
	Admin   AdminListener   `yaml:"admin"`
}

type GatewayListener struct {
	Port string      `yaml:"port"`
	CORS *CORSConfig `yaml:"cors,omitempty"`
}

type AdminListener struct {
	Port string      `yaml:"port"`
	Auth AdminAuth   `yaml:"auth"`
	CORS *CORSConfig `yaml:"cors,omitempty"`
}

type AdminAuth struct {
	Type     string `yaml:"type"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowAllOrigins  bool     `yaml:"allow_all_origins,omitempty"`
	AllowedOrigins   []string `yaml:"allowed_origins,omitempty"`
	AllowedMethods   []string `yaml:"allowed_methods,omitempty"`
	AllowedHeaders   []string `yaml:"allowed_headers,omitempty"`
	AllowCredentials bool     `yaml:"allow_credentials,omitempty"`
	MaxAge           Duration `yaml:"max_age,omitempty"`
}

type ProvidersSection struct {
	Static StaticProviderConfig `yaml:"static"`
	// Rincon and other providers added in later phases.
}

// Watch reload mechanisms for the static provider.
const (
	// WatchModeFile reloads via filesystem events (fsnotify). Low latency,
	// but depends on inotify/kqueue delivering events for the config path —
	// which not every filesystem does (some network and overlay mounts, and
	// certain container volume drivers, never fire events).
	WatchModeFile = "file"
	// WatchModePoll reloads by stat-ing the config file on a fixed interval.
	// Higher latency and a small constant cost, but works anywhere os.Stat
	// does — the reliable choice when file events aren't delivered.
	WatchModePoll = "poll"
)

type StaticProviderConfig struct {
	// Watch enables hot-reloading the config file. When false the config is
	// read once at startup and a restart is required to pick up changes.
	Watch bool `yaml:"watch"`
	// WatchMode selects the reload mechanism: "file" (default, fsnotify) or
	// "poll". Ignored when Watch is false.
	WatchMode string `yaml:"watch_mode,omitempty"`
	// WatchInterval is the poll period for WatchModePoll. Defaults to 5s.
	// Ignored for WatchModeFile.
	WatchInterval Duration `yaml:"watch_interval,omitempty"`
}

type Upstream struct {
	Name         string       `yaml:"name"`
	Version      string       `yaml:"version"`
	Instances    []string     `yaml:"instances"`
	LoadBalancer string       `yaml:"load_balancer,omitempty"`
	HealthCheck  *HealthCheck `yaml:"health_check,omitempty"`
	Timeouts     Timeouts     `yaml:"timeouts"`
}

type HealthCheck struct {
	Path     string   `yaml:"path"`
	Interval Duration `yaml:"interval"`
	Timeout  Duration `yaml:"timeout"`
}

type Timeouts struct {
	Dial    Duration `yaml:"dial"`
	Headers Duration `yaml:"headers"`
	Overall Duration `yaml:"overall"`
	Idle    Duration `yaml:"idle"`
}

// RouteTimeouts is the subset of Timeouts that can be overridden at the
// route level. dial, headers, and idle are applied via the per-upstream
// Transport so they live on the upstream; only overall is per-request.
type RouteTimeouts struct {
	Overall Duration `yaml:"overall,omitempty"`
}

// Middleware holds a middleware definition. The runtime is introduced in a
// later phase; for now we keep the type loose.
type Middleware struct {
	Type   string         `yaml:"type"`
	Config map[string]any `yaml:",inline"`
}

type Envelope struct {
	Type     string `yaml:"type"`
	Template string `yaml:"template,omitempty"`
}

type Route struct {
	Name        string         `yaml:"name"`
	Match       RouteMatch     `yaml:"match"`
	Upstream    string         `yaml:"upstream"`
	Rewrite     *Rewrite       `yaml:"rewrite,omitempty"`
	Envelope    string         `yaml:"envelope,omitempty"`
	Middlewares []string       `yaml:"middlewares,omitempty"`
	Limits      *Limits        `yaml:"limits,omitempty"`
	Timeouts    *RouteTimeouts `yaml:"timeouts,omitempty"`
}

type RouteMatch struct {
	Path    string   `yaml:"path"`
	Methods []string `yaml:"methods,omitempty"`
	Host    string   `yaml:"host,omitempty"`
}

type Rewrite struct {
	StripPrefix   string `yaml:"strip_prefix,omitempty"`
	ReplacePrefix string `yaml:"replace_prefix,omitempty"`
}

type ObservabilitySection struct {
	Logging LoggingConfig `yaml:"logging"`
	Metrics MetricsConfig `yaml:"metrics"`
	Tracing TracingConfig `yaml:"tracing"`
}

type LoggingConfig struct {
	Level           string   `yaml:"level"`
	Format          string   `yaml:"format"`
	AccessLog       bool     `yaml:"access_log"`
	RedactHeaders   []string `yaml:"redact_headers,omitempty"`
	MaxBodyLogBytes int      `yaml:"max_body_log_bytes,omitempty"`
}

type MetricsConfig struct {
	Prometheus PrometheusConfig `yaml:"prometheus"`
}

type PrometheusConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type TracingConfig struct {
	OTLP OTLPConfig `yaml:"otlp"`
}

type OTLPConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"`
}

// Size is a byte count that YAML-unmarshals from strings like "100MB",
// "500KB", "1GiB", or from raw integer bytes. Binary multipliers (1024)
// are used for all suffixes.
type Size int64

func (s Size) Bytes() int64 { return int64(s) }

var sizeRE = regexp.MustCompile(`^\s*(\d+(?:\.\d+)?)\s*([a-zA-Z]*)\s*$`)

func (s *Size) UnmarshalYAML(value *yaml.Node) error {
	// Try integer first.
	var n int64
	if err := value.Decode(&n); err == nil {
		*s = Size(n)
		return nil
	}
	var raw string
	if err := value.Decode(&raw); err != nil {
		return err
	}
	if raw == "" {
		*s = 0
		return nil
	}
	m := sizeRE.FindStringSubmatch(raw)
	if m == nil {
		return fmt.Errorf("invalid size %q", raw)
	}
	num, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return fmt.Errorf("invalid size %q: %w", raw, err)
	}
	mult, err := sizeMultiplier(m[2])
	if err != nil {
		return fmt.Errorf("invalid size %q: %w", raw, err)
	}
	*s = Size(int64(num * float64(mult)))
	return nil
}

func sizeMultiplier(unit string) (int64, error) {
	switch strings.ToUpper(unit) {
	case "", "B":
		return 1, nil
	case "K", "KB", "KIB":
		return 1 << 10, nil
	case "M", "MB", "MIB":
		return 1 << 20, nil
	case "G", "GB", "GIB":
		return 1 << 30, nil
	default:
		return 0, fmt.Errorf("unknown size unit %q", unit)
	}
}

// Duration wraps time.Duration for YAML unmarshaling of strings like "5s".
type Duration time.Duration

func (d Duration) AsDuration() time.Duration {
	return time.Duration(d)
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	if s == "" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}
