package rules

import (
	"fmt"
	"sort"
	"strings"
)

// Phase represents the combat phase a rule fires in.
type Phase string

const (
	PhaseHit    Phase = "hit"
	PhaseWound  Phase = "wound"
	PhaseSave   Phase = "save"
	PhaseDamage Phase = "damage"
)

// RuleScope defines where a rule may be applied.
type RuleScope string

const (
	ScopeWeapon RuleScope = "weapon" // weapon-intrinsic only
	ScopeUnit   RuleScope = "unit"   // unit/detachment level only
	ScopeBoth   RuleScope = "both"   // may appear at either level
)

// ParamSchema describes a single parameter required by a parametric rule.
type ParamSchema struct {
	Name        string // e.g. "X", "Keyword"
	Type        string // "int" | "keyword"
	Description string
}

// RuleDefinition fully describes one rule in the game.
// The UI only surfaces rules present in the Registry, preventing orphan rules
// that the engine cannot resolve.
type RuleDefinition struct {
	Name         string
	DisplayName  string // UI label — may contain placeholder e.g. "Sustained Hits (X)"
	Phases       []Phase
	Scope        RuleScope
	IsParametric bool
	Params       []ParamSchema
	Description  string
	Implemented  bool // false = known but not yet simulated; shown greyed out in UI
}

// Registry is the authoritative dictionary of all rules the application knows about.
// Any rule string used in a profile MUST have a matching entry here (or match
// a parametric base like "Sustained Hits" or "Anti").
var Registry = map[string]RuleDefinition{

	// ── Fully Implemented ─────────────────────────────────────────────────────

	DevastatingWounds: {
		Name:        DevastatingWounds,
		DisplayName: "Devastating Wounds",
		Phases:      []Phase{PhaseWound},
		Scope:       ScopeBoth,
		Description: "Critical wounds (normally on 6+; lowered by Anti-[Keyword]) inflict a Mortal Wound instead of a normal wound. Armour and invulnerable saves cannot be used — FNP still applies.",
		Implemented: true,
	},
	LethalHits: {
		Name:        LethalHits,
		DisplayName: "Lethal Hits",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeBoth,
		Description: "Unmodified To Hit rolls of 6 automatically wound the target — no To Wound roll is made. The wound proceeds to the save phase normally. If the weapon also has Sustained Hits X, the natural 6 auto-wounds AND generates X bonus hits (which still need wound rolls).",
		Implemented: true,
	},
	Torrent: {
		Name:        Torrent,
		DisplayName: "Torrent",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeWeapon,
		Description: "This weapon automatically hits its target. No To Hit roll is made.",
		Implemented: true,
	},
	SustainedHitsPrefix: {
		Name:         SustainedHitsPrefix,
		DisplayName:  "Sustained Hits (X)",
		Phases:       []Phase{PhaseHit},
		Scope:        ScopeBoth,
		IsParametric: true,
		Params: []ParamSchema{
			{Name: "X", Type: "int", Description: "Number of bonus hits generated on a natural 6"},
		},
		Description: "Unmodified To Hit rolls of 6 generate X additional hits that still need To Wound rolls. If combined with Lethal Hits, the natural 6 auto-wounds AND generates X bonus hits.",
		Implemented: true,
	},
	Stealth: {
		Name:        Stealth,
		DisplayName: "Stealth",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeUnit,
		Description: "Attackers targeting this unit suffer -1 to their To Hit rolls (hit threshold raised by 1).",
		Implemented: true,
	},

	// Anti-[Keyword] is stored as "Anti-KEYWORD THRESHOLD" (e.g. "Anti-INFANTRY 4").
	// The registry entry uses the base name "Anti" for display/lookup purposes.
	AntiPrefix: {
		Name:         AntiPrefix,
		DisplayName:  "Anti-[Keyword] (X+)",
		Phases:       []Phase{PhaseWound},
		Scope:        ScopeBoth,
		IsParametric: true,
		Params: []ParamSchema{
			{Name: "Keyword", Type: "keyword", Description: "The unit keyword this rule targets (e.g. INFANTRY)"},
			{Name: "X", Type: "int", Description: "Minimum unmodified wound roll for a critical wound (2–6)"},
		},
		Description: "Unmodified To Wound rolls of X+ are critical wounds against units with the specified keyword. Critical wounds trigger Devastating Wounds (if present) even below 6. Without Devastating Wounds, a critical wound is still a normal wound.",
		Implemented: true,
	},

	Blast: {
		Name:        Blast,
		DisplayName: "Blast",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeWeapon,
		Description: "For each 5 models in the target unit, this weapon gains 1 bonus attack.",
		Implemented: true,
	},

	Hazardous: {
		Name:        Hazardous,
		DisplayName: "Hazardous",
		Phases:      []Phase{PhaseDamage},
		Scope:       ScopeWeapon,
		Description: "After this weapon fires, roll 1D6 for each model using it. On a 1, that model suffers 3 Mortal Wounds. The attacker's Feel No Pain (if any) can be used to save each of those mortal wounds.",
		Implemented: true,
	},

	IgnoresCover: {
		Name:        IgnoresCover,
		DisplayName: "Ignores Cover",
		Phases:      []Phase{PhaseSave},
		Scope:       ScopeBoth,
		Description: "This weapon's attacks do not suffer the -1 AP penalty against defenders benefiting from Cover. Only relevant when 'Defender in Cover' is enabled in the simulation configuration.",
		Implemented: true,
	},

	// ── Phase Ordering (not damage-relevant) ─────────────────────────────────

	FightsFirst: {
		Name:        FightsFirst,
		DisplayName: "Fights First",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeUnit,
		Description: "This unit fights before other units in the Fight phase, even if they did not charge. Phase-ordering rule — does not affect damage output calculation.",
		Implemented: false,
	},

	// ── Stubs (saved to profiles; not yet simulated) ──────────────────────────

	"Melta X": {
		Name:         "Melta X",
		DisplayName:  "Melta (X)",
		Phases:       []Phase{PhaseDamage},
		Scope:        ScopeWeapon,
		IsParametric: true,
		Params:       []ParamSchema{{Name: "X", Type: "int", Description: "Bonus damage at half range"}},
		Description:  "Add X to the damage characteristic of each successful attack if the target is within half this weapon's range.",
		Implemented:  false,
	},
	"Indirect Fire": {
		Name:        "Indirect Fire",
		DisplayName: "Indirect Fire",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeWeapon,
		Description: "This weapon can target units not visible to the bearer, but all hit rolls suffer -1.",
		Implemented: false,
	},
	"Precision": {
		Name:        "Precision",
		DisplayName: "Precision",
		Phases:      []Phase{PhaseWound},
		Scope:       ScopeWeapon,
		Description: "Unmodified wound rolls of 6 can target CHARACTER models even if they are not the closest model.",
		Implemented: false,
	},
	"Twin-linked": {
		Name:        "Twin-linked",
		DisplayName: "Twin-linked",
		Phases:      []Phase{PhaseWound},
		Scope:       ScopeWeapon,
		Description: "Re-roll wound rolls of 1.",
		Implemented: false,
	},
	"Rapid Fire X": {
		Name:         "Rapid Fire X",
		DisplayName:  "Rapid Fire (X)",
		Phases:       []Phase{PhaseHit},
		Scope:        ScopeWeapon,
		IsParametric: true,
		Params:       []ParamSchema{{Name: "X", Type: "int", Description: "Bonus attacks within half range"}},
		Description:  "Gain X additional attacks if the target is within half this weapon's range.",
		Implemented:  false,
	},
}

// Get returns the RuleDefinition for a given rule name string (including parametric
// variants like "Sustained Hits 2" and "Anti-INFANTRY 4").
func Get(name string) (RuleDefinition, bool) {
	// Exact match first
	if def, ok := Registry[name]; ok {
		return def, true
	}
	// Sustained Hits X — match by prefix
	var x int
	if n, _ := fmt.Sscanf(name, "Sustained Hits %d", &x); n == 1 {
		return Registry[SustainedHitsPrefix], true
	}
	// Anti-[KEYWORD] X — match by "Anti-" prefix
	if strings.HasPrefix(name, "Anti-") {
		if _, _, ok := ParseAntiRule(name); ok {
			return Registry[AntiPrefix], true
		}
	}
	return RuleDefinition{}, false
}

// AllSorted returns all rule definitions sorted alphabetically by DisplayName.
func AllSorted() []RuleDefinition {
	defs := make([]RuleDefinition, 0, len(Registry))
	for _, d := range Registry {
		defs = append(defs, d)
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].DisplayName < defs[j].DisplayName
	})
	return defs
}

// ImplementedSorted returns only implemented rules, sorted by DisplayName.
func ImplementedSorted() []RuleDefinition {
	var defs []RuleDefinition
	for _, d := range Registry {
		if d.Implemented {
			defs = append(defs, d)
		}
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].DisplayName < defs[j].DisplayName
	})
	return defs
}
