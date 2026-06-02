package config

const (
	Name    = "Kerbecs"
	Version = "3.1.0"
)

func FormattedNameWithVersion() string {
	return Name + ":v" + Version
}
