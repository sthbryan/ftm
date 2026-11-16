package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/sthbryan/ftm/internal/version"
)

//go:embed static/*
var staticFiles embed.FS

const (
	tagTunnels   = "Tunnels"
	tagEvents    = "Events"
	tagProviders = "Providers"
	tagSettings  = "Settings"
	tagApp       = "App"
)

// newAPI configures huma for ftm: OpenAPI 3.1 at /openapi.json, no hosted
// docs page (it would load scripts from a CDN), and response bodies exactly
// as declared, without huma's $schema links.
func newAPI(mux *http.ServeMux) huma.API {
	installErrors()

	cfg := huma.DefaultConfig("Foundry Tunnel Manager", version.Version)
	cfg.Info.Description = "Local API of Foundry Tunnel Manager. Browser pages served by ftm need no " +
		"credentials; any other client sends the token from the runtime file (ftm.json) as " +
		"`Authorization: Bearer <token>` or `" + TokenHeader + ": <token>`."
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	cfg.CreateHooks = nil
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"runtimeToken": {Type: "http", Scheme: "bearer", Description: "The token from the runtime file."},
		"runtimeTokenHeader": {
			Type: "apiKey", In: "header", Name: TokenHeader,
			Description: "The token from the runtime file.",
		},
	}
	cfg.Security = []map[string][]string{{"runtimeToken": {}}, {"runtimeTokenHeader": {}}, {}}

	return humago.New(mux, cfg)
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	api := newAPI(mux)

	s.registerTunnels(api)
	s.registerEvents(api)
	s.registerProviders(api)
	s.registerSettings(api)
	s.registerApp(api)

	mux.HandleFunc("/ws/events", s.handleWebSocket)
	mux.Handle("/", staticHandler())

	return s.guard(jsonMisses(mux))
}

// jsonMisses answers an /api path that matches no route, or matches one only
// with another method, with a JSON error instead of the dashboard page.
func jsonMisses(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isGuardedPath(r.URL.Path) || isRoute(mux, r) {
			mux.ServeHTTP(w, r)
			return
		}

		var allow []string
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			probe := r.Clone(r.Context())
			probe.Method = method
			if isRoute(mux, probe) {
				allow = append(allow, method)
			}
		}
		if len(allow) > 0 {
			w.Header().Set("Allow", strings.Join(allow, ", "))
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		writeError(w, http.StatusNotFound, "not_found", "not found")
	})
}

// isRoute reports whether r reaches an API route rather than the dashboard
// fallback.
func isRoute(mux *http.ServeMux, r *http.Request) bool {
	_, pattern := mux.Handler(r)
	return pattern != "" && pattern != "/"
}

// staticHandler serves the built dashboard: web/dist when ftm runs from the
// repository, the embedded copy otherwise. Paths without a file extension get
// index.html so client-side routes reload.
func staticHandler() http.Handler {
	var staticFS fs.FS
	if _, err := os.Stat(filepath.Join("web", "dist")); err == nil {
		staticFS = os.DirFS(filepath.Join("web", "dist"))
	} else {
		staticFS, _ = fs.Sub(staticFiles, "static")
	}
	fileServer := http.FileServer(http.FS(staticFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !strings.Contains(r.URL.Path, ".") {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

// jsonContent documents a JSON response body of v's type.
func jsonContent(api huma.API, v any, name string) map[string]*huma.MediaType {
	return map[string]*huma.MediaType{
		"application/json": {Schema: api.OpenAPI().Components.Schemas.Schema(typeOf(v), true, name)},
	}
}
