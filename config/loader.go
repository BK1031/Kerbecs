package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultFilePath is used when KERBECS_CONFIG is unset.
const DefaultFilePath = "kerbecs.yaml"

// FilePath returns the path to the config file, honoring KERBECS_CONFIG.
func FilePath() string {
	if p := os.Getenv("KERBECS_CONFIG"); p != "" {
		return p
	}
	return DefaultFilePath
}

// LoadFile reads and parses the kerbecs config file, expanding ${VAR} and
// ${VAR:default} references using the process environment.
func LoadFile(path string) (*File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	expanded := expandEnv(raw)
	var f File
	if err := yaml.Unmarshal(expanded, &f); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &f, nil
}

// expandEnv replaces ${VAR} and ${VAR:default} in the input with the value of
// VAR from the environment, or the default if VAR is unset or empty.
func expandEnv(input []byte) []byte {
	return []byte(os.Expand(string(input), func(key string) string {
		name, def, hasDefault := strings.Cut(key, ":")
		if val, ok := os.LookupEnv(name); ok && val != "" {
			return val
		}
		if hasDefault {
			return def
		}
		return ""
	}))
}
