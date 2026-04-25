package middleware

import "github.com/gin-gonic/gin"

// headersConfig is the typed config for the headers middleware.
//
//   request:   add/remove headers on the inbound request before proxy
//   response:  add/remove headers on the outbound response after proxy
//
// 'add' overwrites existing headers with the given value (Set semantics, not
// Add). 'remove' is a list of header names to delete.
type headersConfig struct {
	Type     string             `yaml:"type"`
	Request  headersDirection   `yaml:"request"`
	Response headersDirection   `yaml:"response"`
}

type headersDirection struct {
	Add    map[string]string `yaml:"add"`
	Remove []string          `yaml:"remove"`
}

type headersMW struct {
	requestAdd     map[string]string
	requestRemove  []string
	responseAdd    map[string]string
	responseRemove []string
}

func newHeaders(decode func(any) error) (Middleware, error) {
	var c headersConfig
	if err := decode(&c); err != nil {
		return nil, err
	}
	return &headersMW{
		requestAdd:     c.Request.Add,
		requestRemove:  c.Request.Remove,
		responseAdd:    c.Response.Add,
		responseRemove: c.Response.Remove,
	}, nil
}

func (m *headersMW) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, name := range m.requestRemove {
			c.Request.Header.Del(name)
		}
		for name, value := range m.requestAdd {
			c.Request.Header.Set(name, value)
		}

		c.Next()

		for _, name := range m.responseRemove {
			c.Writer.Header().Del(name)
		}
		for name, value := range m.responseAdd {
			c.Writer.Header().Set(name, value)
		}
	}
}
