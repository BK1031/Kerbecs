package config

const (
	Name    = "Kerbecs"
	Version = "3.1.1"
)

func FormattedNameWithVersion() string {
	return Name + ":v" + Version
}
