package engine

import (
	"github.com/smackface/wh-lethality/internal/dice"
	"github.com/smackface/wh-lethality/internal/profiles"
	"github.com/smackface/wh-lethality/internal/rules"
)

// resolveWeaponAttack runs one complete attack sequence for a single weapon
// against a defender's stats. effectiveRules is the merged set of weapon
// rules + unit-level rules.
func resolveWeaponAttack(
	weapon profiles.WeaponProfile,
	effectiveRules []string,
	defStats profiles.ModelStats,
	defRules []string,
	roller *dice.Roller,
) CombatResult {
	result := CombatResult{}

	// ─── Hit Phase ────────────────────────────────────────────────────────────

	hitThreshold := weapon.BalSkill

	// Stealth: attacker suffers -1 to hit (raise threshold by 1)
	for _, r := range defRules {
		if r == rules.Stealth {
			hitThreshold++
			break
		}
	}

	isTorrent := weapon.HasRule(rules.Torrent)
	// Torrent may also come from unit-level rules
	if !isTorrent {
		for _, r := range effectiveRules {
			if r == rules.Torrent {
				isTorrent = true
				break
			}
		}
	}

	totalHits := 0
	autoWounds := 0

	for i := 0; i < weapon.FiringRate; i++ {
		roll := roller.D6()
		outcome := rules.ApplyHitRules(rules.HitContext{
			Roll:         roll,
			HitThreshold: hitThreshold,
			IsTorrent:    isTorrent,
		}, effectiveRules)

		if outcome.IsAutoWound {
			autoWounds++
		}
		if outcome.IsHit {
			totalHits++
		}
		totalHits += outcome.BonusHits
	}

	result.Hits = totalHits + autoWounds

	// ─── Wound Phase ──────────────────────────────────────────────────────────

	woundThreshold := rules.WoundThreshold(weapon.Strength, defStats.Toughness)
	normalWounds := autoWounds // Lethal Hits skip wound roll, go straight to saves
	mortalWounds := 0

	for i := 0; i < totalHits; i++ {
		roll := roller.D6()
		outcome := rules.ApplyWoundRules(rules.WoundContext{
			Roll:           roll,
			WoundThreshold: woundThreshold,
		}, effectiveRules)

		if outcome.IsMortalWound {
			mortalWounds++
		} else if outcome.IsWound {
			normalWounds++
		}
	}

	result.Wounds = normalWounds
	result.MortalWounds = mortalWounds

	// ─── Save Phase ───────────────────────────────────────────────────────────

	effectiveSave := rules.EffectiveSave(defStats.ArmorSave, defStats.InvulSave, weapon.AP)
	unsavedWounds := 0

	for i := 0; i < normalWounds; i++ {
		if roller.D6() < effectiveSave {
			unsavedWounds++
		}
	}

	result.UnsavedWounds = unsavedWounds

	// ─── Damage & Feel No Pain ────────────────────────────────────────────────

	totalDamage := 0
	fnp := defStats.FeelNoPain

	applyDamage := func(sources int) {
		for i := 0; i < sources; i++ {
			for d := 0; d < weapon.Damage; d++ {
				if fnp > 0 && roller.D6() >= fnp {
					continue // FNP passed — point of damage negated
				}
				totalDamage++
			}
		}
	}

	applyDamage(unsavedWounds)
	applyDamage(mortalWounds)

	result.TotalDamage = totalDamage
	if defStats.Wounds > 0 {
		result.DefenderKills = totalDamage / defStats.Wounds
	}

	return result
}

// resolveUnitAttack runs all weapon attacks from a fully resolved UnitProfile
// against a defender for a given combat phase. It iterates every model group
// and each model's weapons, merging unit-level rules into effective weapon rules.
func resolveUnitAttack(
	attacker profiles.UnitProfile,
	defender profiles.UnitProfile,
	phase string,
	roller *dice.Roller,
) CombatResult {
	defStats := defender.PrimaryStats()
	combined := CombatResult{}

	for _, group := range attacker.Groups {
		// Normal models (all if no special weapon, Count-1 if there is one)
		normalCount := group.NormalCount()
		for i := 0; i < normalCount; i++ {
			for _, w := range group.Weapons {
				if !w.UsableInPhase(phase) {
					continue
				}
				effective := mergeStrings(w.Rules, attacker.Rules)
				r := resolveWeaponAttack(w, effective, defStats, defender.Rules, roller)
				combined = addResults(combined, r)
			}
		}

		// Special model (sergeant, champion) with their own weapon loadout
		if group.HasSpecialWeapon && len(group.SpecialWeapons) > 0 {
			for _, w := range group.SpecialWeapons {
				if !w.UsableInPhase(phase) {
					continue
				}
				effective := mergeStrings(w.Rules, attacker.Rules)
				r := resolveWeaponAttack(w, effective, defStats, defender.Rules, roller)
				combined = addResults(combined, r)
			}
		}
	}

	return combined
}

// addResults sums two CombatResults together.
func addResults(a, b CombatResult) CombatResult {
	return CombatResult{
		Hits:          a.Hits + b.Hits,
		Wounds:        a.Wounds + b.Wounds,
		MortalWounds:  a.MortalWounds + b.MortalWounds,
		UnsavedWounds: a.UnsavedWounds + b.UnsavedWounds,
		TotalDamage:   a.TotalDamage + b.TotalDamage,
		DefenderKills: a.DefenderKills + b.DefenderKills,
	}
}
