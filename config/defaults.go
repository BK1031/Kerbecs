package config

// ApplyDefaults fills in unset fields on a parsed File with the values the
// gateway needs in order to boot. It returns a list of human-readable warnings
// for defaults that represent security-relevant fallbacks (e.g. admin
// credentials), so the caller can log them at warn level.
func ApplyDefaults(f *File) []string {
	var warnings []string

	if f.Gateway.Env == "" {
		f.Gateway.Env = "PROD"
	}
	if f.Listeners.Gateway.Port == "" {
		f.Listeners.Gateway.Port = "10310"
	}
	if f.Listeners.Admin.Port == "" {
		f.Listeners.Admin.Port = "10300"
	}
	if f.Listeners.Admin.Auth.Type == "" {
		f.Listeners.Admin.Auth.Type = "basic"
	}
	if f.Listeners.Admin.Auth.Username == "" {
		f.Listeners.Admin.Auth.Username = "admin"
		warnings = append(warnings, "admin username not set in config; defaulting to \"admin\"")
	}
	if f.Listeners.Admin.Auth.Password == "" {
		f.Listeners.Admin.Auth.Password = "admin"
		warnings = append(warnings, "admin password not set in config; defaulting to \"admin\" — DO NOT USE IN PRODUCTION")
	}

	return warnings
}
