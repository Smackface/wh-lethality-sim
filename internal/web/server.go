// Package web provides the HTTP server and HTMX-driven UI for wh-lethality.
package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/smackface/wh-lethality/internal/profiles"
	"github.com/smackface/wh-lethality/internal/web/handlers"
)

// Server holds all server state and routes.
type Server struct {
	mux   *http.ServeMux
	store *profiles.Store
}

// New creates a Server, parses templates from the provided FS, and registers routes.
func New(store *profiles.Store, tmplFS fs.FS, staticFS fs.FS) (*Server, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"mul": func(a, b float64) float64 { return a * b },
		"pct": func(f float64) string {
			return template.HTMLEscapeString(fmt.Sprintf("%.1f%%", f*100))
		},
	}).ParseFS(tmplFS, "templates/*.html", "templates/partials/*.html")
	if err != nil {
		return nil, err
	}

	h := handlers.New(store, tmpl)

	s := &Server{
		mux:   http.NewServeMux(),
		store: store,
	}

	// Static assets
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Pages
	s.mux.HandleFunc("GET /", h.Index)

	// Unit CRUD
	s.mux.HandleFunc("GET /units", h.ListUnits)
	s.mux.HandleFunc("GET /units/new", h.NewUnitForm)
	s.mux.HandleFunc("POST /units", h.CreateUnit)
	s.mux.HandleFunc("GET /units/{id}", h.GetUnit)
	s.mux.HandleFunc("DELETE /units/{id}", h.DeleteUnit)

	// HTMX partials
	s.mux.HandleFunc("GET /partials/unit-details", h.UnitDetails)

	// Simulation
	s.mux.HandleFunc("POST /simulate", h.RunSim)

	// Rules API (used by UI dropdowns)
	s.mux.HandleFunc("GET /api/rules", h.ListRules)

	return s, nil
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
