package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"kerbecs/config"
	"kerbecs/provider"
	"kerbecs/router"

	"github.com/gin-gonic/gin"
)

func testRouter(t *testing.T) *router.Router {
	t.Helper()
	f := &config.File{
		Upstreams: map[string]config.Upstream{
			"core": {Name: "core", Instances: []string{"http://core:9999"}},
		},
		Routes: []config.Route{
			{
				Name:     "core-internal",
				Match:    config.RouteMatch{Path: "/api/core/*"},
				Upstream: "core",
				Rewrite:  &config.Rewrite{StripPrefix: "/api"},
			},
		},
	}
	static, err := provider.NewStatic(f)
	if err != nil {
		t.Fatalf("NewStatic: %v", err)
	}
	rt, err := router.New(static)
	if err != nil {
		t.Fatalf("router.New: %v", err)
	}
	return rt
}

func newEngine(rt *router.Router) *gin.Engine {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	rf := RouterFunc(func() *router.Router { return rt })
	e.GET("/admin-gw/resolve", Resolve(rf))
	e.GET("/admin-gw/routes", Routes(rf))
	e.GET("/admin-gw/upstreams", Upstreams(rf))
	return e
}

func TestResolve_Match(t *testing.T) {
	e := newEngine(testRouter(t))

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin-gw/resolve?path=/api/core/entity/123", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", w.Code, w.Body.String())
	}
	var resp resolveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Matched || resp.Upstream != "core" {
		t.Errorf("got %+v, want matched core", resp)
	}
	if resp.URL != "http://core:9999" {
		t.Errorf("url = %q, want http://core:9999", resp.URL)
	}
	if resp.RewrittenPath != "/core/entity/123" {
		t.Errorf("rewritten_path = %q, want /core/entity/123", resp.RewrittenPath)
	}
}

func TestResolve_NoMatch(t *testing.T) {
	e := newEngine(testRouter(t))

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin-gw/resolve?path=/nope", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	var resp resolveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Matched {
		t.Error("expected matched=false")
	}
}

func TestResolve_MissingPath(t *testing.T) {
	e := newEngine(testRouter(t))

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin-gw/resolve", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestRoutesAndUpstreams(t *testing.T) {
	e := newEngine(testRouter(t))

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin-gw/routes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("routes status = %d", w.Code)
	}
	var routes struct {
		Routes []routeInfo `json:"routes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &routes); err != nil {
		t.Fatal(err)
	}
	if len(routes.Routes) != 1 || routes.Routes[0].Upstream != "core" || routes.Routes[0].StripPrefix != "/api" {
		t.Errorf("unexpected routes: %+v", routes.Routes)
	}

	w = httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin-gw/upstreams", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("upstreams status = %d", w.Code)
	}
	var ups struct {
		Upstreams []upstreamInfo `json:"upstreams"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &ups); err != nil {
		t.Fatal(err)
	}
	if len(ups.Upstreams) != 1 || ups.Upstreams[0].Name != "core" || len(ups.Upstreams[0].Instances) != 1 {
		t.Errorf("unexpected upstreams: %+v", ups.Upstreams)
	}
}
