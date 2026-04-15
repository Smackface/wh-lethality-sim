package profiles

import "fmt"

// WeaponType indicates which phase a weapon may be used in.
type WeaponType string

const (
	WeaponRanged WeaponType = "ranged"
	WeaponMelee  WeaponType = "melee"
	// Pistol weapons can fire in both the Shooting and Fight phases.
	WeaponPistol WeaponType = "pistol"
)

// WeaponProfile represents a single weapon's full characteristics.
// BalSkill serves as WS for melee weapons — the mechanic is identical (roll d6 >= value).
type WeaponProfile struct {
	Name       string     `json:"name"`
	Type       WeaponType `json:"type"`                 // ranged / melee / pistol
	Attacks  int        `json:"attacks"`           // number of attacks
	BalSkill   int        `json:"bal_skill"`            // BS or WS: minimum roll to hit (e.g. 3 for 3+)
	Strength   int        `json:"strength"`
	AP         int        `json:"ap"`                   // positive magnitude: 1 = AP-1, 2 = AP-2
	Damage     int        `json:"damage"`               // flat damage per unsaved wound
	Rules      []string   `json:"rules,omitempty"`      // weapon-intrinsic rules
}

// HasRule returns true if this weapon carries the named rule.
func (w *WeaponProfile) HasRule(name string) bool {
	for _, r := range w.Rules {
		if r == name {
			return true
		}
	}
	return false
}

// SustainedHitsBonus returns X from "Sustained Hits X", or 0 if not present.
func (w *WeaponProfile) SustainedHitsBonus() int {
	var bonus int
	for _, r := range w.Rules {
		if _, err := fmt.Sscanf(r, "Sustained Hits %d", &bonus); err == nil {
			return bonus
		}
	}
	return 0
}

// UsableInPhase returns true if this weapon type may be used in the given combat phase.
func (w *WeaponProfile) UsableInPhase(phase string) bool {
	switch phase {
	case "shooting":
		return w.Type == WeaponRanged || w.Type == WeaponPistol
	case "melee":
		return w.Type == WeaponMelee || w.Type == WeaponPistol
	default:
		return true // unknown phase: allow everything
	}
}
