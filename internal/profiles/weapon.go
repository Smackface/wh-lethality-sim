package profiles

import "fmt"

// WeaponProfile represents a single weapon's characteristics.
type WeaponProfile struct {
	Name string // e.g. "Sternguard Bolter"

	// Core ballistic stats
	FiringRate int // Number of attacks generated (A)
	BalSkill   int // Ballistic/Weapon Skill — the minimum roll to hit (e.g. 3 for 3+)
	Strength   int // S
	AP         int // Armor Piercing as a positive magnitude (e.g. 1 for AP-1, 2 for AP-2)
	Damage     int // D — flat damage per unsaved wound

	// Special rules carried by this weapon.
	// String-based for flexibility; see rules/keywords.go for defined constants.
	// Examples: "Devastating Wounds", "Lethal Hits", "Torrent", "Sustained Hits 1"
	Rules []string

	// TODO: DamageExpr string    — for variable damage, e.g. "D3", "D6+1", "2D3"
	// TODO: AntiKeyword string   — for Anti-[Keyword] X+ rules
	// TODO: AntiThreshold int    — the wound roll threshold for Anti rules
	// TODO: MeltaBonus int       — Melta X (bonus damage within half-range)
	// TODO: Blast bool           — Minimum 3 shots vs 5+ model units
}

// HasRule returns true if the weapon has the named rule (exact match).
func (w *WeaponProfile) HasRule(name string) bool {
	for _, r := range w.Rules {
		if r == name {
			return true
		}
	}
	return false
}

// SustainedHitsBonus returns the bonus hit count from "Sustained Hits X", or 0.
func (w *WeaponProfile) SustainedHitsBonus() int {
	var bonus int
	for _, r := range w.Rules {
		if _, err := fmt.Sscanf(r, "Sustained Hits %d", &bonus); err == nil {
			return bonus
		}
	}
	return 0
}
