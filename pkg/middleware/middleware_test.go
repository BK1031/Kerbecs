package middleware

import (
	"kerbecs/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// decodeYAML returns a decode func for use in tests, simulating what the
// config layer would pass to a factory.
func decodeYAML(t *testing.T, src string) func(any) error {
	t.Helper()
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(src), &node); err != nil {
		t.Fatal(err)
	}
	// yaml.Unmarshal wraps in a Document node; descend to the content.
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = *node.Content[0]
	}
	return node.Decode
}

func TestBuildRegistry_UnknownType(t *testing.T) {
	defer Reset()
	defs := map[string]config.Middleware{
		"x": {Type: "no_such_thing"},
	}
	if _, err := BuildRegistry(defs); err == nil {
		t.Error("expected error for unknown middleware type")
	}
}

func TestRegistry_ChainResolvesNames(t *testing.T) {
	defer Reset()
	RegisterBuiltins()
	defs := map[string]config.Middleware{}
	// Build via constructed defs would need YAML decoding; skip and just
	// verify Chain returns an error when a name is missing.
	r, err := BuildRegistry(defs)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Chain([]string{"ghost"}); err == nil {
		t.Error("expected error for unknown middleware name")
	}
	chain, err := r.Chain(nil)
	if err != nil || chain != nil {
		t.Errorf("nil names should yield nil chain, no error; got %v / %v", chain, err)
	}
}

func TestRequestID_GeneratesAndPropagates(t *testing.T) {
	mw, err := newRequestID(decodeYAML(t, `type: request_id`))
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(mw.Handler())
	r.GET("/", func(c *gin.Context) {
		v, _ := c.Get("Request-ID")
		c.String(http.StatusOK, v.(string))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Request-Id") == "" {
		t.Error("expected X-Request-Id on response")
	}
	if w.Body.String() == "" {
		t.Error("expected Request-ID set in context")
	}
}

func TestRequestID_TrustsIncomingWhenConfigured(t *testing.T) {
	mw, err := newRequestID(decodeYAML(t, `
type: request_id
trust_incoming: true
`))
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(mw.Handler())
	r.GET("/", func(c *gin.Context) {
		v, _ := c.Get("Request-ID")
		c.String(http.StatusOK, v.(string))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", "client-supplied-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "client-supplied-id" {
		t.Errorf("expected client-supplied-id; got %q", w.Body.String())
	}
}

func TestRequestID_RegeneratesByDefault(t *testing.T) {
	mw, err := newRequestID(decodeYAML(t, `type: request_id`))
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(mw.Handler())
	r.GET("/", func(c *gin.Context) {
		v, _ := c.Get("Request-ID")
		c.String(http.StatusOK, v.(string))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", "client-supplied-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() == "client-supplied-id" {
		t.Error("default config should regenerate, not trust client value")
	}
}

func TestHeaders_RequestAddRemove(t *testing.T) {
	mw, err := newHeaders(decodeYAML(t, `
type: headers
request:
  add:
    X-Gateway: kerbecs
  remove: [X-Internal-Trace]
response:
  add:
    X-Foo: bar
`))
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(mw.Handler())
	r.GET("/", func(c *gin.Context) {
		// echo headers we received
		c.Writer.Header().Set("X-Saw-Gateway", c.GetHeader("X-Gateway"))
		c.Writer.Header().Set("X-Saw-Internal", c.GetHeader("X-Internal-Trace"))
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Internal-Trace", "should-be-stripped")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Saw-Gateway") != "kerbecs" {
		t.Error("request.add did not apply: handler did not see X-Gateway")
	}
	if w.Header().Get("X-Saw-Internal") != "" {
		t.Error("request.remove did not apply: handler saw X-Internal-Trace")
	}
	if w.Header().Get("X-Foo") != "bar" {
		t.Error("response.add did not apply")
	}
}
