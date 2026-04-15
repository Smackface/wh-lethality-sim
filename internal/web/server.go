// Package web provides the HTTP server and HTMX-driven UI for wh-lethality.
package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/smackface/wh-lethality/internal/profiles"
	"github.com/smackface/wh-lethality/internal/rules"
	"github.com/smackface/wh-lethality/internal/web/handlers"
)

// safeID converts a rule name to a CSS/HTML-safe id token (spaces→hyphens, lowercase).
func safeID(name string) string {
	r := strings.NewReplacer(" ", "-", "[", "", "]", "", "(", "", ")", "", "/", "-")
	return strings.ToLower(r.Replace(name))
}

// existingRuleVal returns the current parameter values for a parametric rule on a unit.
// Returns a 2-element slice: [value0, value1] (empty strings if not found).
//   - "Sustained Hits 2"  → ["2", ""]
//   - "Anti-INFANTRY 4"   → ["INFANTRY", "4"]
func existingRuleVal(u *profiles.UnitProfile, baseName string) []string {
	empty := []string{"", ""}
	if u == nil {
		return empty
	}
	for _, r := range u.Rules {
		switch baseName {
		case rules.SustainedHitsPrefix:
			if strings.HasPrefix(r, rules.SustainedHitsPrefix+" ") {
				val := strings.TrimPrefix(r, rules.SustainedHitsPrefix+" ")
				return []string{val, ""}
			}
		case rules.AntiPrefix:
			if kw, thresh, ok := rules.ParseAntiRule(r); ok {
				return []string{kw, fmt.Sprint(thresh)}
			}
		}
	}
	return empty
}

// New creates the HTTP server, parses all templates from tmplFS, and registers routes.
// tmplFS should be rooted at the templates directory (e.g. os.DirFS("web/templates")).
func New(store *profiles.Store, tmplFS fs.FS, staticFS fs.FS) (http.Handler, error) {
	funcMap := template.FuncMap{
		"join": strings.Join,
		"hasRule": func(u *profiles.UnitProfile, name string) bool {
			if u == nil {
				return false
			}
			return u.HasRule(name)
		},
		// safeId converts a rule name to a CSS/HTML-safe id token
		"safeId": safeID,
		// existingRuleVal returns parameter values for a rule already on the unit
		"existingRuleVal": existingRuleVal,
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
	mux.HandleFunc("GET /{$}", h.Index)
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
