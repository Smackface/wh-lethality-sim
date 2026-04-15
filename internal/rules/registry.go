package rules

import (
	"fmt"
	"sort"
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
	Name        string // e.g. "X"
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
// Any rule string used in a profile MUST have a matching entry here.
var Registry = map[string]RuleDefinition{

	// ── Implemented ──────────────────────────────────────────────────────────

	DevastatingWounds: {
		Name:        DevastatingWounds,
		DisplayName: "Devastating Wounds",
		Phases:      []Phase{PhaseWound},
		Scope:       ScopeWeapon,
		Description: "Unmodified To Wound rolls of 6 inflict a Mortal Wound instead of a normal wound. Armour and invulnerable saves cannot be used against Mortal Wounds.",
		Implemented: true,
	},
	LethalHits: {
		Name:        LethalHits,
		DisplayName: "Lethal Hits",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeBoth,
		Description: "Unmodified To Hit rolls of 6 automatically wound the target. No To Wound roll is made; the wound proceeds to the save phase normally.",
		Implemented: true,
	},
	Torrent: {
		Name:        Torrent,
		DisplayName: "Torrent",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeWeapon,
		Description: "This weapon automatically hits its target. No To Hit roll is required.",
		Implemented: true,
	},
	SustainedHitsPrefix: {
		Name:         SustainedHitsPrefix,
		DisplayName:  "Sustained Hits (X)",
		Phases:       []Phase{PhaseHit},
		Scope:        ScopeBoth,
		IsParametric: true,
		Params:       []ParamSchema{{Name: "X", Type: "int", Description: "Bonus hits generated on an unmodified 6 to hit"}},
		Description:  "Unmodified To Hit rolls of 6 generate X additional hits that are resolved against the target.",
		Implemented:  true,
	},
	Stealth: {
		Name:        Stealth,
		DisplayName: "Stealth",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeUnit,
		Description: "Each time an attack targets this unit, subtract 1 from the To Hit roll.",
		Implemented: true,
	},

	// ── Not Yet Implemented ───────────────────────────────────────────────────
	// Shown in the UI (so users know they exist) but flagged as unimplemented
	// so the engine can warn rather than silently ignore them.

	"Twin-linked": {
		Name:        "Twin-linked",
		DisplayName: "Twin-linked",
		Phases:      []Phase{PhaseWound},
		Scope:       ScopeWeapon,
		Description: "You may re-roll wound rolls for attacks made with this weapon.",
		Implemented: false,
	},
	"Hazardous": {
		Name:        "Hazardous",
		DisplayName: "Hazardous",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeWeapon,
		Description: "After firing, roll one D6. On a 1 the bearer suffers 1 Mortal Wound.",
		Implemented: false,
	},
	"Lance": {
		Name:        "Lance",
		DisplayName: "Lance",
		Phases:      []Phase{PhaseWound},
		Scope:       ScopeWeapon,
		Description: "+1 to wound rolls against Vehicle and Monster keyword targets.",
		Implemented: false,
	},
	"Ignores Cover": {
		Name:        "Ignores Cover",
		DisplayName: "Ignores Cover",
		Phases:      []Phase{PhaseSave},
		Scope:       ScopeWeapon,
		Description: "The target cannot use the Benefit of Cover against attacks from this weapon.",
		Implemented: false,
	},
	"Precision": {
		Name:        "Precision",
		DisplayName: "Precision",
		Phases:      []Phase{PhaseWound},
		Scope:       ScopeWeapon,
		Description: "Unmodified Wound rolls of 6 may target a Character in the unit, regardless of targeting priority.",
		Implemented: false,
	},
	"Anti-[Keyword] X+": {
		Name:         "Anti-[Keyword] X+",
		DisplayName:  "Anti-[Keyword] (X+)",
		Phases:       []Phase{PhaseWound},
		Scope:        ScopeWeapon,
		IsParametric: true,
		Params: []ParamSchema{
			{Name: "Keyword", Type: "keyword", Description: "Target keyword (e.g. INFANTRY, FLY, CHAOS)"},
			{Name: "X", Type: "int", Description: "Wound roll threshold for auto-wound"},
		},
		Description: "Unmodified Wound rolls of X+ are always successful against targets with the specified keyword.",
		Implemented: false,
	},
	"Melta X": {
		Name:         "Melta X",
		DisplayName:  "Melta (X)",
		Phases:       []Phase{PhaseDamage},
		Scope:        ScopeWeapon,
		IsParametric: true,
		Params:       []ParamSchema{{Name: "X", Type: "int", Description: "Bonus damage at half range"}},
		Description:  "If the target is within half this weapon's range, add X to the damage of each successful attack.",
		Implemented:  false,
	},
	"Blast": {
		Name:        "Blast",
		DisplayName: "Blast",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeWeapon,
		Description: "Add 1 attack per 5 models in the target unit. Minimum 3 attacks if the target unit has 5+ models.",
		Implemented: false,
	},
	"Rapid Fire X": {
		Name:         "Rapid Fire X",
		DisplayName:  "Rapid Fire (X)",
		Phases:       []Phase{PhaseHit},
		Scope:        ScopeWeapon,
		IsParametric: true,
		Params:       []ParamSchema{{Name: "X", Type: "int", Description: "Bonus attacks within half range"}},
		Description:  "This weapon generates X additional attacks if the target is within half the weapon's range.",
		Implemented:  false,
	},
	"Fights First": {
		Name:        "Fights First",
		DisplayName: "Fights First",
		Phases:      []Phase{PhaseHit},
		Scope:       ScopeUnit,
		Description: "This unit fights before other units in the Fight phase, even if it did not charge.",
		Implemented: false,
	},
}

// Get returns the RuleDefinition for a name, handling parametric variants
// like "Sustained Hits 2" matching the "Sustained Hits" base entry.
func Get(name string) (RuleDefinition, bool) {
	if def, ok := Registry[name]; ok {
		return def, true
	}
	// Sustained Hits X
	var x int
	if n, _ := fmt.Sscanf(name, "Sustained Hits %d", &x); n == 1 {
		def := Registry[SustainedHitsPrefix]
		return def, true
	}
	return RuleDefinition{}, false
}

// AllSorted returns all rule definitions sorted alphabetically by name.
func AllSorted() []RuleDefinition {
	defs := make([]RuleDefinition, 0, len(Registry))
	for _, d := range Registry {
		defs = append(defs, d)
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})
	return defs
}

// ImplementedSorted returns only implemented rules, sorted by name.
func ImplementedSorted() []RuleDefinition {
	var defs []RuleDefinition
	for _, d := range Registry {
		if d.Implemented {
			defs = append(defs, d)
		}
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})
	return defs
}
