package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// requestIDConfig is the typed config for the request_id middleware.
//
//   header:          response header name (default: X-Request-Id)
//   trust_incoming:  if true, an incoming request that already has the header
//                    set keeps that value; if false, the gateway always
//                    generates a fresh ID.
type requestIDConfig struct {
	Type          string `yaml:"type"`
	Header        string `yaml:"header"`
	TrustIncoming bool   `yaml:"trust_incoming"`
}

type requestID struct {
	header        string
	trustIncoming bool
}

func newRequestID(decode func(any) error) (Middleware, error) {
	var c requestIDConfig
	if err := decode(&c); err != nil {
		return nil, err
	}
	if c.Header == "" {
		c.Header = "X-Request-Id"
	}
	return &requestID{header: c.Header, trustIncoming: c.TrustIncoming}, nil
}

func (m *requestID) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var id string
		if m.trustIncoming {
			id = c.GetHeader(m.header)
		}
		if id == "" {
			if v, _ := uuid.NewV7(); v != uuid.Nil {
				id = v.String()
			}
		}
		c.Set("Request-ID", id)
		c.Request.Header.Set(m.header, id)
		c.Writer.Header().Set(m.header, id)
		c.Next()
	}
}
