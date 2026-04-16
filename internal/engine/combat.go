package engine

import (
	"github.com/smackface/wh-lethality/internal/dice"
	"github.com/smackface/wh-lethality/internal/profiles"
	"github.com/smackface/wh-lethality/internal/rules"
)

// resolveWeaponAttack runs one complete attack sequence for a single weapon
// carried by a single model, returning the full combat result including any
// Hazardous self-wounds the attacking model suffers.
//
// Parameters:
//   - weapon:           the weapon being fired/swung
//   - effectiveRules:   merged weapon rules + unit rules
//   - attackerStats:    the attacking model's stats (for Hazardous FNP)
//   - defStats:         the defending unit's primary stats (T, saves, FNP, W)
//   - defRules:         the defending unit's own rules (e.g. Stealth)
//   - defKeywords:      the defending unit's keywords (for Anti-[Keyword])
//   - defModelCount:    total defender model count (for Blast bonus attacks)
//   - defInCover:       whether the defender benefits from Cover
//   - roller:           the goroutine-local dice roller
func resolveWeaponAttack(
	weapon profiles.WeaponProfile,
	effectiveRules []string,
	attackerStats profiles.ModelStats,
	defStats profiles.ModelStats,
	defRules []string,
	defKeywords []string,
	defModelCount int,
	defInCover bool,
	withinHalfRange bool,
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

	isTorrent := false
	for _, r := range effectiveRules {
		if r == rules.Torrent {
			isTorrent = true
			break
		}
	}

	// Indirect Fire: -1 to hit (raise hit threshold by 1), like Stealth but from attacker side
	for _, r := range effectiveRules {
		if r == rules.IndirectFire {
			hitThreshold++
			break
		}
	}

	// Blast: +1 attack per 5 models in defender
	attacks := weapon.Attacks
	if weapon.HasRule(rules.Blast) && defModelCount > 0 {
		attacks += defModelCount / 5
	}

	// Rapid Fire X: +X attacks within half range
	if withinHalfRange {
		for _, r := range effectiveRules {
			if x, ok := rules.ParseRapidFire(r); ok {
				attacks += x
				break
			}
		}
	}

	totalHits := 0  // hits that need wound rolls
	autoWounds := 0 // hits from Lethal Hits that skip wound rolls

	for i := 0; i < attacks; i++ {
		roll := roller.D6()
		outcome := rules.ApplyHitRules(rules.HitContext{
			Roll:         roll,
			HitThreshold: hitThreshold,
			IsTorrent:    isTorrent,
		}, effectiveRules)

		if outcome.IsAutoWound {
			autoWounds++
			// BonusHits from Sustained Hits still need wound rolls even when the
			// original hit is an auto-wound via Lethal Hits.
			totalHits += outcome.BonusHits
		} else if outcome.IsHit {
			totalHits++
			totalHits += outcome.BonusHits
		}
	}

	result.Hits = totalHits + autoWounds

	// ─── Wound Phase ──────────────────────────────────────────────────────────

	woundThreshold := rules.WoundThreshold(weapon.Strength, defStats.Toughness)

	// Anti-[Keyword]: compute critical wound threshold for this weapon vs this defender.
	// Devastating Wounds (if present) trigger on any critical wound, regardless of
	// whether it was a natural 6 or a lower roll enabled by Anti-[Keyword].
	critThreshold := rules.CriticalWoundThreshold(effectiveRules, defKeywords)

	// Twin-linked: re-roll wound rolls of 1
	hasTwinLinked := false
	for _, r := range effectiveRules {
		if r == rules.TwinLinked {
			hasTwinLinked = true
			break
		}
	}

	normalWounds := autoWounds // Lethal Hit auto-wounds skip wound roll, go straight to saves
	mortalWounds := 0

	for i := 0; i < totalHits; i++ {
		roll := roller.D6()
		// Twin-linked: re-roll on a 1 (re-rolled result stands)
		if roll == 1 && hasTwinLinked {
			roll = roller.D6()
		}
		outcome := rules.ApplyWoundRules(rules.WoundContext{
			Roll:                   roll,
			WoundThreshold:         woundThreshold,
			CriticalWoundThreshold: critThreshold,
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

	// Cover: defender gets +1 to armour save (reduces threshold by 1) unless attacker
	// has Ignores Cover. Cover only applies to ranged weapons.
	if defInCover && weapon.Type != profiles.WeaponMelee {
		hasIgnoresCover := false
		for _, r := range effectiveRules {
			if r == rules.IgnoresCover {
				hasIgnoresCover = true
				break
			}
		}
		if !hasIgnoresCover {
			effectiveSave-- // +1 to save roll → lower threshold for the defender
			if effectiveSave < 1 {
				effectiveSave = 1
			}
		}
	}

	unsavedWounds := 0
	for i := 0; i < normalWounds; i++ {
		if roller.D6() < effectiveSave {
			unsavedWounds++
		}
	}

	result.UnsavedWounds = unsavedWounds

	// ─── Damage & Feel No Pain ────────────────────────────────────────────────

	// Melta X: +X to damage per successful attack within half range
	effectiveDamage := weapon.Damage
	if withinHalfRange {
		for _, r := range effectiveRules {
			if x, ok := rules.ParseMelta(r); ok {
				effectiveDamage += x
				break
			}
		}
	}

	fnp := defStats.FeelNoPain
	totalDamage := 0

	applyDamageInstances := func(sources int) {
		for i := 0; i < sources; i++ {
			for d := 0; d < effectiveDamage; d++ {
				if fnp > 0 && roller.D6() >= fnp {
					continue // FNP passed — this point of damage negated
				}
				totalDamage++
			}
		}
	}
	applyMortalInstances := func(count int) {
		for i := 0; i < count; i++ {
			if fnp > 0 && roller.D6() >= fnp {
				continue
			}
			totalDamage++
		}
	}

	applyDamageInstances(unsavedWounds)
	applyMortalInstances(mortalWounds)

	result.TotalDamage = totalDamage
	if defStats.Wounds > 0 {
		result.DefenderKills = totalDamage / defStats.Wounds
	}

	// ─── Hazardous Self-Wounds ────────────────────────────────────────────────
	// Roll 1D6. On a 1, the bearer suffers 3 Mortal Wounds.
	// The attacker's own FNP (if any) applies to each of those MWs.

	if weapon.HasRule(rules.Hazardous) {
		if roller.D6() == 1 {
			attackerFNP := attackerStats.FeelNoPain
			selfMW := 0
			for d := 0; d < 3; d++ {
				if attackerFNP > 0 && roller.D6() >= attackerFNP {
					continue // attacker FNP saves this MW
				}
				selfMW++
			}
			result.HazardousSelfMW = selfMW
		}
	}

	return result
}

// resolveUnitAttack runs all weapon attacks from a fully resolved UnitProfile
// against a defender for a given combat phase.
func resolveUnitAttack(
	attacker profiles.UnitProfile,
	defender profiles.UnitProfile,
	phase string,
	defModelCount int,
	defInCover bool,
	withinHalfRange bool,
	roller *dice.Roller,
) CombatResult {
	defStats := defender.PrimaryStats()
	combined := CombatResult{}

	for _, group := range attacker.Groups {
		normalCount := group.NormalCount()

		for i := 0; i < normalCount; i++ {
			for _, w := range group.Weapons {
				if !w.UsableInPhase(phase) {
					continue
				}
				effective := combineRules(w.Rules, attacker.Rules)
				r := resolveWeaponAttack(
					w, effective,
					group.Stats, defStats,
					defender.Rules, defender.Keywords,
					defModelCount, defInCover, withinHalfRange,
					roller,
				)
				combined = addResults(combined, r)
			}
		}

		// Special model (sergeant / champion) with their own loadout
		if group.HasSpecialWeapon && len(group.SpecialWeapons) > 0 {
			for _, w := range group.SpecialWeapons {
				if !w.UsableInPhase(phase) {
					continue
				}
				effective := combineRules(w.Rules, attacker.Rules)
				r := resolveWeaponAttack(
					w, effective,
					group.Stats, defStats,
					defender.Rules, defender.Keywords,
					defModelCount, defInCover, withinHalfRange,
					roller,
				)
				combined = addResults(combined, r)
			}
		}
	}

	return combined
}

// addResults sums two CombatResults together.
func addResults(a, b CombatResult) CombatResult {
	return CombatResult{
		Hits:            a.Hits + b.Hits,
		Wounds:          a.Wounds + b.Wounds,
		MortalWounds:    a.MortalWounds + b.MortalWounds,
		UnsavedWounds:   a.UnsavedWounds + b.UnsavedWounds,
		TotalDamage:     a.TotalDamage + b.TotalDamage,
		DefenderKills:   a.DefenderKills + b.DefenderKills,
		HazardousSelfMW: a.HazardousSelfMW + b.HazardousSelfMW,
	}
}

// combineRules returns a new slice containing all elements of a and b without deduplication.
// (attachment.go's mergeStrings deduplicates; for weapon+unit rule merging we want to preserve all)
func combineRules(a, b []string) []string {
	out := make([]string, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}
