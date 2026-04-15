// Package web provides the HTTP server and HTMX-driven UI for wh-lethality.
package web

import (
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/smackface/wh-lethality/internal/profiles"
	"github.com/smackface/wh-lethality/internal/rules"
	"github.com/smackface/wh-lethality/internal/web/handlers"
)

// New creates the HTTP server, parses all templates from tmplFS, and registers routes.
// tmplFS should be rooted at the templates directory (e.g. os.DirFS("web/templates")).
// staticFS should be rooted at the static directory (e.g. os.DirFS("web/static")).
func New(store *profiles.Store, tmplFS fs.FS, staticFS fs.FS) (http.Handler, error) {
	funcMap := template.FuncMap{
		// join combines a string slice with a separator
		"join": strings.Join,
		// hasRule checks if a unit profile has a given rule (used in unit_form.html)
		"hasRule": func(u *profiles.UnitProfile, name string) bool {
			if u == nil {
				return false
			}
			return u.HasRule(name)
		},
		// allRules exposes the registry to templates
		"allRules": rules.AllSorted,
	}

	tmpl, err := template.New("root").Funcs(funcMap).ParseFS(
		tmplFS,
		"*.html",
		"partials/*.html",
	)
	if err != nil {
		return nil, err
	}

	h := handlers.New(store, tmpl)
	mux := http.NewServeMux()

	// Static assets
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Pages
	mux.HandleFunc("GET /{$}", h.Index) // exact root only
	mux.HandleFunc("GET /units", h.ListUnits)
	mux.HandleFunc("GET /units/new", h.NewUnitForm)
	mux.HandleFunc("POST /units", h.CreateUnit)
	mux.HandleFunc("GET /units/{id}", h.GetUnit)
	mux.HandleFunc("DELETE /units/{id}", h.DeleteUnit)

	// HTMX partials
	mux.HandleFunc("GET /partials/unit-details", h.UnitDetails)

	// Simulation
	mux.HandleFunc("POST /simulate", h.RunSim)

	// Rules API
	mux.HandleFunc("GET /api/rules", h.ListRules)

	return mux, nil
}
