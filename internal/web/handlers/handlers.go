// Package handlers contains all HTTP request handlers for the wh-lethality web UI.
package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

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

func (h *H) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// ─── Pages ────────────────────────────────────────────────────────────────────

func (h *H) Index(w http.ResponseWriter, r *http.Request) {
	units, _ := h.store.List()
	h.render(w, "index.html", map[string]any{
		"Units":       units,
		"Detachments": detachment.All(),
	})
}

func (h *H) ListUnits(w http.ResponseWriter, r *http.Request) {
	units, _ := h.store.List()
	h.render(w, "units.html", map[string]any{"Units": units})
}

func (h *H) NewUnitForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, "unit_form.html", map[string]any{
		"Unit":      &profiles.UnitProfile{},
		"Rules":     rules.AllSorted(),
		"GroupsJSON": "[]",
	})
}

func (h *H) GetUnit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	unit, err := h.store.Load(id)
	if err != nil {
		http.Error(w, "unit not found", http.StatusNotFound)
		return
	}
	groupsJSON, _ := json.MarshalIndent(unit.Groups, "", "  ")
	h.render(w, "unit_form.html", map[string]any{
		"Unit":       unit,
		"Rules":      rules.AllSorted(),
		"GroupsJSON": string(groupsJSON),
	})
}

func (h *H) CreateUnit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Parse keywords from comma-separated string
	kwRaw := r.FormValue("keywords")
	var keywords []string
	for _, kw := range strings.Split(kwRaw, ",") {
		kw = strings.TrimSpace(kw)
		if kw != "" {
			keywords = append(keywords, kw)
		}
	}

	// Parse model groups from JSON textarea
	groupsRaw := r.FormValue("groups_json")
	var groups []profiles.ModelGroup
	if groupsRaw != "" {
		if err := json.Unmarshal([]byte(groupsRaw), &groups); err != nil {
			http.Error(w, "invalid groups JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Assemble unit-level rules, including parametric variants
	ruleList := r.Form["rules"] // non-parametric checkboxes

	// Sustained Hits and other single-int parametric rules
	for _, baseName := range r.Form["rule_param_base"] {
		baseName = strings.TrimSpace(baseName)
		if baseName == "" {
			continue
		}
		safeKey := toSafeID(baseName)
		valStr := strings.TrimSpace(r.FormValue("rule_param_val_" + safeKey))
		if valStr == "" {
			valStr = "1"
		}
		ruleList = append(ruleList, baseName+" "+valStr)
	}

	// Anti-[Keyword] (X+)
	if r.FormValue("rule_anti_enabled") == "1" {
		kw := strings.ToUpper(strings.TrimSpace(r.FormValue("rule_anti_keyword")))
		thresh := strings.TrimSpace(r.FormValue("rule_anti_threshold"))
		if kw != "" && thresh != "" {
			ruleList = append(ruleList, "Anti-"+kw+" "+thresh)
		}
	}

	unit := &profiles.UnitProfile{
		ID:       r.FormValue("id"),
		Label:    r.FormValue("label"),
		Keywords: keywords,
		Rules:    ruleList,
		Groups:   groups,
	}

	if err := h.store.Save(unit); err != nil {
		http.Error(w, "save error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", "/units")
	w.WriteHeader(http.StatusCreated)
}

func (h *H) DeleteUnit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.Delete(id); err != nil {
		http.Error(w, "delete error", http.StatusInternalServerError)
		return
	}
	// HTMX: return empty 200 so hx-swap="outerHTML" removes the card
	w.WriteHeader(http.StatusOK)
}

// ─── HTMX Partials ────────────────────────────────────────────────────────────

func (h *H) UnitDetails(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		// attacker_id or defender_id comes via hx-include as form value
		if err := r.ParseForm(); err == nil {
			id = r.FormValue("attacker_id")
			if id == "" {
				id = r.FormValue("defender_id")
			}
		}
	}
	if id == "" {
		w.Write([]byte(""))
		return
	}
	unit, err := h.store.Load(id)
	if err != nil {
		w.Write([]byte(`<p class="text-red-400 text-xs">Unit not found</p>`))
		return
	}
	h.render(w, "unit_details.html", map[string]any{
		"Unit": unit,
		"Side": r.URL.Query().Get("side"),
	})
}

// ─── Simulation ───────────────────────────────────────────────────────────────

// DamageBar is pre-computed bar chart data for the template.
type DamageBar struct {
	Damage   int
	Count    int
	PctStr   string
	WidthPct float64
}

// SimView is the view model passed to sim_results.html.
type SimView struct {
	AttackerLabel string
	DefenderLabel string
	Phase         string
	Iterations    int

	// Pre-formatted strings for display
	MeanHitsStr            string
	MeanWoundsStr          string
	MeanMortalsStr         string
	HasMortals             bool
	MeanDamageStr          string
	KillPctStr             string
	KillPctClass           string // tailwind colour class
	MeanKillsStr           string
	StdDevStr              string
	HasHazardous           bool
	MeanHazardousSelfMWStr string

	// Percentiles
	Damage50th int
	Damage75th int
	Damage95th int

	// Bar chart
	Bars []DamageBar
}

func buildSimView(attacker, defender profiles.UnitProfile, phase string, stats engine.SimStats) SimView {
	kp := stats.KillProbability * 100
	kpClass := "text-red-400"
	if kp >= 50 {
		kpClass = "text-green-400"
	} else if kp >= 25 {
		kpClass = "text-yellow-400"
	}

	// Build sorted bar chart data
	type kv struct{ k, v int }
	pairs := make([]kv, 0, len(stats.DamageDist))
	maxCount := 0
	for k, v := range stats.DamageDist {
		pairs = append(pairs, kv{k, v})
		if v > maxCount {
			maxCount = v
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].k < pairs[j].k })

	bars := make([]DamageBar, len(pairs))
	for i, p := range pairs {
		pct := float64(p.v) / float64(stats.Iterations) * 100
		width := 0.0
		if maxCount > 0 {
			width = float64(p.v) / float64(maxCount) * 100
		}
		bars[i] = DamageBar{
			Damage:   p.k,
			Count:    p.v,
			PctStr:   fmt.Sprintf("%.1f", pct),
			WidthPct: math.Round(width*10) / 10,
		}
	}

	return SimView{
		AttackerLabel:          attacker.Label,
		DefenderLabel:          defender.Label,
		Phase:                  phase,
		Iterations:             stats.Iterations,
		MeanHitsStr:            fmt.Sprintf("%.2f", stats.MeanHits),
		MeanWoundsStr:          fmt.Sprintf("%.2f", stats.MeanWounds+stats.MeanMortalWounds),
		MeanMortalsStr:         fmt.Sprintf("%.2f", stats.MeanMortalWounds),
		HasMortals:             stats.MeanMortalWounds > 0.001,
		MeanDamageStr:          fmt.Sprintf("%.2f", stats.MeanDamage),
		KillPctStr:             fmt.Sprintf("%.1f", kp),
		KillPctClass:           kpClass,
		MeanKillsStr:           fmt.Sprintf("%.3f", stats.MeanKills),
		StdDevStr:              fmt.Sprintf("%.2f", stats.DamageStdDev),
		HasHazardous:           stats.MeanHazardousSelfMW > 0.001,
		MeanHazardousSelfMWStr: fmt.Sprintf("%.2f", stats.MeanHazardousSelfMW),
		Damage50th:             stats.Damage50th,
		Damage75th:             stats.Damage75th,
		Damage95th:             stats.Damage95th,
		Bars:                   bars,
	}
}

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

	if attackerID == "" || defenderID == "" {
		http.Error(w, "attacker and defender required", http.StatusBadRequest)
		return
	}

	attacker, err := h.store.Load(attackerID)
	if err != nil {
		http.Error(w, "attacker not found: "+err.Error(), http.StatusBadRequest)
		return
	}
	defender, err := h.store.Load(defenderID)
	if err != nil {
		http.Error(w, "defender not found: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Character attachment
	au := &engine.AttachedUnit{Bodyguard: attacker}
	if characterID != "" {
		if char, err := h.store.Load(characterID); err == nil {
			au.Character = char
		}
	}
	resolvedAttacker := au.Resolve()

	// Apply detachment abilities
	for _, aid := range r.Form["detachment_abilities"] {
		if aid == "" {
			continue
		}
		for _, det := range detachment.All() {
			for _, ability := range det.Abilities {
				if ability.ID == aid {
					resolvedAttacker = engine.ApplyDetachmentRules(resolvedAttacker, ability.GrantsRules)
				}
			}
		}
	}

	defInCover := r.FormValue("defender_in_cover") == "1"

	stats := engine.RunSimulation(engine.SimConfig{
		Attacker:        resolvedAttacker,
		Defender:        *defender,
		Phase:           phase,
		Iterations:      iterations,
		DefenderInCover: defInCover,
	})

	view := buildSimView(resolvedAttacker, *defender, phase, stats)
	h.render(w, "sim_results.html", view)
}

// toSafeID converts a rule name to a safe HTML id/name token (spaces → hyphens, lowercase).
func toSafeID(name string) string {
	return strings.ToLower(strings.NewReplacer(" ", "-", "[", "", "]", "", "(", "", ")", "", "/", "-").Replace(name))
}

// ─── API ──────────────────────────────────────────────────────────────────────

func (h *H) ListRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules.AllSorted())
}
