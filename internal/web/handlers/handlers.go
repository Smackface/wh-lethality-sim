// Package handlers contains all HTTP request handlers for the wh-lethality web UI.
package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/smackface/wh-lethality/internal/detachment"
	"github.com/smackface/wh-lethality/internal/engine"
	"github.com/smackface/wh-lethality/internal/profiles"
	"github.com/smackface/wh-lethality/internal/rules"
)

// H holds dependencies shared across all handlers.
type H struct {
	store *profiles.Store
	tmpl  *template.Template
}

// New creates a handler set.
func New(store *profiles.Store, tmpl *template.Template) *H {
	return &H{store: store, tmpl: tmpl}
}

// render executes a template by name, writing to w. On error, writes 500.
func (h *H) render(w http.ResponseWriter, name string, data any) {
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// ─── Pages ────────────────────────────────────────────────────────────────────

// Index serves the main simulation page.
func (h *H) Index(w http.ResponseWriter, r *http.Request) {
	units, _ := h.store.List()
	h.render(w, "index.html", map[string]any{
		"Units":       units,
		"Detachments": detachment.All(),
	})
}

// ListUnits serves the unit library page.
func (h *H) ListUnits(w http.ResponseWriter, r *http.Request) {
	units, _ := h.store.List()
	h.render(w, "units.html", map[string]any{"Units": units})
}

// NewUnitForm serves the blank unit creation form.
func (h *H) NewUnitForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, "unit_form.html", map[string]any{
		"Unit":  &profiles.UnitProfile{},
		"Rules": rules.AllSorted(),
	})
}

// GetUnit serves the edit form for an existing unit.
func (h *H) GetUnit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	unit, err := h.store.Load(id)
	if err != nil {
		http.Error(w, "unit not found", http.StatusNotFound)
		return
	}
	h.render(w, "unit_form.html", map[string]any{
		"Unit":  unit,
		"Rules": rules.AllSorted(),
	})
}

// CreateUnit handles POST /units — saves a new unit from form data.
func (h *H) CreateUnit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// Simple single-group unit from form for now
	// TODO: multi-group form builder
	unit := &profiles.UnitProfile{
		ID:    r.FormValue("id"),
		Label: r.FormValue("label"),
		Rules: r.Form["rules"],
	}
	if err := h.store.Save(unit); err != nil {
		http.Error(w, "save error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// HTMX redirect to unit list
	w.Header().Set("HX-Redirect", "/units")
}

// DeleteUnit handles DELETE /units/{id}.
func (h *H) DeleteUnit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.Delete(id); err != nil {
		http.Error(w, "delete error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ─── HTMX Partials ────────────────────────────────────────────────────────────

// UnitDetails returns an HTML fragment with unit stats for the attacker/defender panel.
// Query params: id=<unit_id>&side=attacker|defender
func (h *H) UnitDetails(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.Write([]byte(""))
		return
	}
	unit, err := h.store.Load(id)
	if err != nil {
		http.Error(w, "unit not found", http.StatusNotFound)
		return
	}
	side := r.URL.Query().Get("side")
	h.render(w, "unit_details.html", map[string]any{
		"Unit": unit,
		"Side": side,
	})
}

// ─── Simulation ───────────────────────────────────────────────────────────────

// RunSim handles POST /simulate.
// Reads form fields: attacker_id, character_id, defender_id, phase, iterations,
// detachment_abilities[] (optional).
func (h *H) RunSim(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	attackerID := r.FormValue("attacker_id")
	characterID := r.FormValue("character_id")
	defenderID := r.FormValue("defender_id")
	phase := r.FormValue("phase")
	if phase == "" {
		phase = "shooting"
	}
	iterations := 10_000
	if s := r.FormValue("iterations"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100_000 {
			iterations = n
		}
	}

	attacker, err := h.store.Load(attackerID)
	if err != nil {
		http.Error(w, "attacker not found: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Character attachment
	attached := &engine.AttachedUnit{Bodyguard: attacker}
	if characterID != "" {
		char, err := h.store.Load(characterID)
		if err == nil {
			attached.Character = char
		}
	}
	resolvedAttacker := attached.Resolve()

	// Apply detachment abilities
	if abilityIDs := r.Form["detachment_abilities"]; len(abilityIDs) > 0 {
		for _, aid := range abilityIDs {
			for _, det := range detachment.All() {
				for _, ability := range det.Abilities {
					if ability.ID == aid {
						resolvedAttacker = engine.ApplyDetachmentRules(resolvedAttacker, ability.GrantsRules)
					}
				}
			}
		}
	}

	defender, err := h.store.Load(defenderID)
	if err != nil {
		http.Error(w, "defender not found: "+err.Error(), http.StatusBadRequest)
		return
	}

	stats := engine.RunSimulation(engine.SimConfig{
		Attacker:   resolvedAttacker,
		Defender:   *defender,
		Phase:      phase,
		Iterations: iterations,
	})

	h.render(w, "sim_results.html", map[string]any{
		"Attacker": resolvedAttacker,
		"Defender": defender,
		"Phase":    phase,
		"Stats":    stats,
	})
}

// ─── API ──────────────────────────────────────────────────────────────────────

// ListRules returns all rules as JSON for the UI dropdown.
func (h *H) ListRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules.AllSorted())
}

// Ensure fmt is used
var _ = fmt.Sprintf
