package config

const (
	Name    = "Kerbecs"
	Version = "3.2.0"
)

func FormattedNameWithVersion() string {
	return Name + ":v" + Version
}
