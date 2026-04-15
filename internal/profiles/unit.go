// Package profiles defines the data structures for unit and weapon profiles.
// TODO(Hunter): This module is intentionally lean for now. We'll explore
// how to load/store profiles (JSON, YAML, a profile registry, etc.) together.
package profiles

// UnitProfile represents the statline and special rules for a single model.
type UnitProfile struct {
	Name string // e.g. "Sternguard Veteran"

	// Core stats
	Toughness int // T
	ArmorSave int // e.g. 3 for a 3+ save
	Wounds    int // W

	// Optional defensive stats (0 = not present)
	InvulSave  int // e.g. 4 for a 4++ invulnerable save; 0 = none
	FeelNoPain int // e.g. 5 for a 5+++ FNP save; 0 = none

	// Keywords used by Anti-[Keyword] weapon rules
	Keywords []string // e.g. ["INFANTRY", "CHAOS", "HERETIC ASTARTES"]

	// Special rules carried by this model/unit that affect combat resolution.
	// Examples: "Stealth" (attackers suffer -1 to hit)
	// TODO: expand as the rules module grows
	Rules []string
}

// HasRule returns true if the unit has the named rule.
func (u *UnitProfile) HasRule(name string) bool {
	for _, r := range u.Rules {
		if r == name {
			return true
		}
	}
	return false
}

// HasKeyword returns true if the unit has the named keyword (case-sensitive).
func (u *UnitProfile) HasKeyword(kw string) bool {
	for _, k := range u.Keywords {
		if k == kw {
			return true
		}
	}
	return false
}
