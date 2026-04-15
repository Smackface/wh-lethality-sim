// Package profiles defines unit and weapon data structures.
// Profiles are loaded from JSON files and combined at runtime by the engine.
package profiles

// ModelStats holds the defensive statline for a single model.
type ModelStats struct {
	Toughness  int `json:"toughness"`
	ArmorSave  int `json:"armor_save"`            // e.g. 3 for 3+
	InvulSave  int `json:"invul_save,omitempty"`   // e.g. 4 for 4++; 0 = none
	FeelNoPain int `json:"feel_no_pain,omitempty"` // e.g. 5 for 5+++ FNP; 0 = none
	Wounds     int `json:"wounds"`
}

// ModelGroup is a set of models within a unit that share the same stats.
//
// If HasSpecialWeapon is true, one model from this group (e.g. a sergeant or
// champion) carries SpecialWeapons instead of the standard Weapons. The engine
// splits that model off automatically: (Count-1) models use Weapons, 1 uses
// SpecialWeapons.
type ModelGroup struct {
	Name             string          `json:"name"`
	Count            int             `json:"count"`
	Stats            ModelStats      `json:"stats"`
	Weapons          []WeaponProfile `json:"weapons"`
	HasSpecialWeapon bool            `json:"has_special_weapon,omitempty"`
	SpecialWeapons   []WeaponProfile `json:"special_weapons,omitempty"`
}

// NormalCount returns the number of models using the standard Weapons loadout.
func (g *ModelGroup) NormalCount() int {
	if g.HasSpecialWeapon && g.Count > 0 {
		return g.Count - 1
	}
	return g.Count
}

// UnitProfile is the top-level definition for an independent unit.
// It is loaded from JSON and combined with other profiles at runtime.
type UnitProfile struct {
	ID       string       `json:"id"`
	Label    string       `json:"label"`
	Groups   []ModelGroup `json:"groups"`
	Rules    []string     `json:"rules"`    // unit-level rules (apply to every weapon attack)
	Keywords []string     `json:"keywords"` // e.g. ["INFANTRY", "ADEPTUS ASTARTES"]
}

// HasRule returns true if this unit carries the named rule.
func (u *UnitProfile) HasRule(name string) bool {
	for _, r := range u.Rules {
		if r == name {
			return true
		}
	}
	return false
}

// HasKeyword returns true if the unit has the given keyword.
func (u *UnitProfile) HasKeyword(kw string) bool {
	for _, k := range u.Keywords {
		if k == kw {
			return true
		}
	}
	return false
}

// TotalModels returns the sum of all model counts across every group.
// Used by the Blast rule (extra attack per 5 models in target unit).
func (u *UnitProfile) TotalModels() int {
	total := 0
	for _, g := range u.Groups {
		total += g.Count
	}
	return total
}

// PrimaryStats returns the ModelStats of the first group.
// Used for defender-side checks (T, saves). For homogeneous units this is exact;
// heterogeneous units are a future TODO.
func (u *UnitProfile) PrimaryStats() ModelStats {
	if len(u.Groups) > 0 {
		return u.Groups[0].Stats
	}
	return ModelStats{}
}
