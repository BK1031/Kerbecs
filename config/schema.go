package config

import (
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
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Env     string `yaml:"env"`
}

type ListenersSection struct {
	Gateway GatewayListener `yaml:"gateway"`
	Admin   AdminListener   `yaml:"admin"`
}

type GatewayListener struct {
	Port string     `yaml:"port"`
	CORS *CORSConfig `yaml:"cors,omitempty"`
}

type AdminListener struct {
	Port string    `yaml:"port"`
	Auth AdminAuth `yaml:"auth"`
}

type AdminAuth struct {
	Type     string `yaml:"type"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type CORSConfig struct {
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods,omitempty"`
	AllowedHeaders   []string `yaml:"allowed_headers,omitempty"`
	AllowCredentials bool     `yaml:"allow_credentials,omitempty"`
	MaxAge           Duration `yaml:"max_age,omitempty"`
}

type ProvidersSection struct {
	Static StaticProviderConfig `yaml:"static"`
	// Rincon and other providers added in later phases.
}

type StaticProviderConfig struct {
	Watch bool `yaml:"watch"`
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
	Dial           Duration `yaml:"dial"`
	ResponseHeader Duration `yaml:"response_header"`
	Overall        Duration `yaml:"overall"`
	Idle           Duration `yaml:"idle"`
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
	Name        string     `yaml:"name"`
	Match       RouteMatch `yaml:"match"`
	Upstream    string     `yaml:"upstream"`
	Rewrite     *Rewrite   `yaml:"rewrite,omitempty"`
	Envelope    string     `yaml:"envelope,omitempty"`
	Middlewares []string   `yaml:"middlewares,omitempty"`
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
	Level            string   `yaml:"level"`
	Format           string   `yaml:"format"`
	AccessLog        bool     `yaml:"access_log"`
	RedactHeaders    []string `yaml:"redact_headers,omitempty"`
	MaxBodyLogBytes  int      `yaml:"max_body_log_bytes,omitempty"`
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
