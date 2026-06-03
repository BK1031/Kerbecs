package admin

import (
	"net/http"

	"kerbecs/config"
	"kerbecs/router"

	"github.com/gin-gonic/gin"
)

// RouterFunc returns the live router. It's a func (not a *Router) so the admin
// endpoints always read the current generation after a hot reload.
type RouterFunc func() *router.Router

type resolveResponse struct {
	Matched       bool     `json:"matched"`
	Route         string   `json:"route,omitempty"`
	Upstream      string   `json:"upstream,omitempty"`
	URL           string   `json:"url,omitempty"`
	Instances     []string `json:"instances,omitempty"`
	RewrittenPath string   `json:"rewritten_path,omitempty"`
}

// Resolve answers "which upstream serves this request?" for the given path
// (and optional method/host). It runs the live router's matcher and applies the
// route's rewrite, so callers get back the upstream URL and the path to send —
// the same decision the gateway would make. Replaces external service-registry
// route matching for in-cluster callers.
//
//	GET /admin-gw/resolve?path=/api/core/entity/123&method=GET
//	-> { matched, route, upstream, url, instances, rewritten_path }
func Resolve(current RouterFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "path query parameter is required"})
			return
		}
		method := c.DefaultQuery("method", http.MethodGet)
		host := c.Query("host")

		rt := current()
		if rt == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "router not ready"})
			return
		}

		m := rt.Find(method, host, path)
		if m == nil || m.Route == nil {
			c.JSON(http.StatusNotFound, resolveResponse{Matched: false})
			return
		}

		resp := resolveResponse{
			Matched:       true,
			Route:         m.Route.Name,
			RewrittenPath: router.RewritePath(path, m.Route.Rewrite),
		}
		if up := m.Route.Upstream; up != nil {
			resp.Upstream = up.Name
			resp.Instances = up.Instances
			resp.URL = up.Pick()
		}
		c.JSON(http.StatusOK, resp)
	}
}

type routeInfo struct {
	Name          string   `json:"name"`
	Path          string   `json:"path"`
	Methods       []string `json:"methods,omitempty"`
	Host          string   `json:"host,omitempty"`
	Upstream      string   `json:"upstream,omitempty"`
	StripPrefix   string   `json:"strip_prefix,omitempty"`
	ReplacePrefix string   `json:"replace_prefix,omitempty"`
}

// Routes lists the live route table in precedence order — the source of truth
// callers can cache to resolve locally between refreshes.
//
//	GET /admin-gw/routes -> { routes: [...] }
func Routes(current RouterFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		rt := current()
		if rt == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "router not ready"})
			return
		}
		out := make([]routeInfo, 0)
		for _, r := range rt.Routes() {
			info := routeInfo{
				Name:    r.Name,
				Path:    r.Match.Path,
				Methods: r.Match.Methods,
				Host:    r.Match.Host,
			}
			if r.Upstream != nil {
				info.Upstream = r.Upstream.Name
			}
			if r.Rewrite != nil {
				info.StripPrefix = r.Rewrite.StripPrefix
				info.ReplacePrefix = r.Rewrite.ReplacePrefix
			}
			out = append(out, info)
		}
		c.JSON(http.StatusOK, gin.H{"routes": out})
	}
}

type upstreamInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version,omitempty"`
	Instances    []string `json:"instances"`
	LoadBalancer string   `json:"load_balancer,omitempty"`
}

// Upstreams lists the distinct upstreams referenced by the route table, with
// their instance pools — a service-registry view of the gateway.
//
//	GET /admin-gw/upstreams -> { upstreams: [...] }
func Upstreams(current RouterFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		rt := current()
		if rt == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "router not ready"})
			return
		}
		out := make([]upstreamInfo, 0)
		for _, up := range rt.Upstreams() {
			out = append(out, upstreamInfo{
				Name:         up.Name,
				Version:      up.Version,
				Instances:    up.Instances,
				LoadBalancer: up.LoadBalancer,
			})
		}
		c.JSON(http.StatusOK, gin.H{"upstreams": out})
	}
}

// Info reports gateway identity and live table sizes — a quick operational
// snapshot for dashboards and health checks.
//
//	GET /admin-gw/info -> { name, version, env, routes, upstreams }
func Info(env string, current RouterFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		routes, upstreams := 0, 0
		if rt := current(); rt != nil {
			routes = len(rt.Routes())
			upstreams = len(rt.Upstreams())
		}
		c.JSON(http.StatusOK, gin.H{
			"name":      config.Name,
			"version":   config.Version,
			"env":       env,
			"routes":    routes,
			"upstreams": upstreams,
		})
	}
}
