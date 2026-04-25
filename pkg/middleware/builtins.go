package middleware

// RegisterBuiltins installs every built-in middleware factory into the
// registry. main calls this once before the config is loaded so factories
// are available when BuildRegistry runs.
func RegisterBuiltins() {
	Register("request_id", newRequestID)
	Register("headers", newHeaders)
}
