package config

const (
	Name    = "Kerbecs"
	Version = "2.2.0"
)

func FormattedNameWithVersion() string {
	return Name + ":v" + Version
}
