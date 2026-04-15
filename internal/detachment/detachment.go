// Package detachment defines detachment abilities and stratagems.
// Both are applied to units at runtime — they add rules (and in future, stat
// modifiers) to the resolved AttachedUnit before it enters the simulation.
package detachment

// DetachmentAbility is a passive ability granted by a detachment to qualifying units.
type DetachmentAbility struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	GrantsRules []string `json:"grants_rules"` // rule keywords added to the target unit
	// TODO: stat modifiers (e.g. +1 to armor save)
	// TODO: keyword restriction (e.g. "only applies to CORE units")
}

// Stratagem is a CP-costed ability that can be activated in a specific phase.
type Stratagem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	CPCost      int      `json:"cp_cost"`
	Phase       string   `json:"phase"`       // "shooting", "melee", "any"
	Description string   `json:"description"`
	GrantsRules []string `json:"grants_rules"`
}

// Detachment is a named collection of abilities and stratagems from a faction codex.
type Detachment struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Faction     string              `json:"faction"`
	Abilities   []DetachmentAbility `json:"abilities"`
	Stratagems  []Stratagem         `json:"stratagems"`
}

// ── Example Detachments ──────────────────────────────────────────────────────

// GladiusTaskForce is the baseline Space Marine detachment from the 10th ed core rules.
var GladiusTaskForce = Detachment{
	ID:      "gladius-task-force",
	Name:    "Gladius Task Force",
	Faction: "ADEPTUS ASTARTES",
	Abilities: []DetachmentAbility{
		{
			ID:          "oath-of-moment",
			Name:        "Oath of Moment",
			Description: "At the start of your Command phase, select one enemy unit. Until the start of your next Command phase, each time a model in your army with this ability makes an attack that targets that unit, you can re-roll the Hit roll.",
			// TODO: implement re-roll hit logic
		},
	},
	Stratagems: []Stratagem{
		{
			ID:          "only-in-death",
			Name:        "Only In Death Does Duty End",
			CPCost:      1,
			Phase:       "any",
			Description: "Use this Stratagem when a model with this ability would be destroyed. That model is not removed from play — it can shoot or fight one more time, then it is removed.",
			// TODO: post-death activation
		},
		{
			ID:          "transhuman-physiology",
			Name:        "Transhuman Physiology",
			CPCost:      1,
			Phase:       "any",
			Description: "Use when an enemy targets an ADEPTUS ASTARTES INFANTRY unit. Until the end of the phase, each time an attack targets that unit, an unmodified Wound roll of 1–3 always fails.",
			// TODO: wound roll lower-bound modifier
		},
	},
}

// WaaghBand is a basic Ork detachment.
var WaaghBand = Detachment{
	ID:      "waagh-band",
	Name:    "Waaagh! Band",
	Faction: "ORK",
	Abilities: []DetachmentAbility{
		{
			ID:          "waaagh",
			Name:        "WAAAGH!",
			Description: "At the start of the Fight phase, if any friendly ORK CORE units are within Engagement Range of an enemy unit, add 1 to the Attack characteristic of all friendly ORK CORE models until the end of the phase.",
			// TODO: attack bonus modifier
		},
	},
	Stratagems: []Stratagem{},
}

// Registry maps detachment IDs to their definitions.
var Registry = map[string]*Detachment{
	GladiusTaskForce.ID: &GladiusTaskForce,
	WaaghBand.ID:        &WaaghBand,
}

// Get returns a detachment by ID.
func Get(id string) (*Detachment, bool) {
	d, ok := Registry[id]
	return d, ok
}

// All returns all registered detachments.
func All() []*Detachment {
	out := make([]*Detachment, 0, len(Registry))
	for _, d := range Registry {
		out = append(out, d)
	}
	return out
}
