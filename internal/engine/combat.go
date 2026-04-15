// Package engine implements the WH40K 10th edition combat resolution loop.
// Each call to ResolveCombat runs one complete attack sequence (all phases)
// for a given attacker+weapon vs defender pairing using a provided dice roller.
package engine

import (
	"github.com/smackface/wh-lethality/internal/dice"
	"github.com/smackface/wh-lethality/internal/profiles"
	"github.com/smackface/wh-lethality/internal/rules"
)

// ResolveCombat runs one complete attack sequence and returns the raw outcome.
// It is safe to call concurrently from multiple goroutines as long as each
// goroutine supplies its own distinct *dice.Roller.
func ResolveCombat(
	attacker profiles.UnitProfile,
	weapon profiles.WeaponProfile,
	defender profiles.UnitProfile,
	roller *dice.Roller,
) CombatResult {
	result := CombatResult{}

	// ─── Phase 1: Hit Rolls ───────────────────────────────────────────────────

	hitThreshold := weapon.BalSkill

	// Stealth on the defender imposes -1 to hit (raises the threshold by 1)
	if defender.HasRule(rules.Stealth) {
		hitThreshold++
	}

	isTorrent := weapon.HasRule(rules.Torrent)
	totalHits := 0
	autoWounds := 0 // From Lethal Hits — skip wound roll, go straight to saves

	for i := 0; i < weapon.FiringRate; i++ {
		roll := roller.D6()
		outcome := rules.ApplyHitRules(rules.HitContext{
			Roll:         roll,
			HitThreshold: hitThreshold,
			IsTorrent:    isTorrent,
		}, weapon.Rules)

		if outcome.IsAutoWound {
			autoWounds++
		}
		if outcome.IsHit {
			totalHits++
		}
		// Bonus hits (Sustained Hits) are rolled as normal hits; generate them here
		totalHits += outcome.BonusHits
	}

	result.Hits = totalHits + autoWounds

	// ─── Phase 2: Wound Rolls ─────────────────────────────────────────────────

	woundThreshold := rules.WoundThreshold(weapon.Strength, defender.Toughness)
	normalWounds := 0
	mortalWounds := 0

	// Auto-wounds from Lethal Hits skip the wound roll and go straight to saves
	normalWounds += autoWounds

	for i := 0; i < totalHits; i++ {
		roll := roller.D6()
		outcome := rules.ApplyWoundRules(rules.WoundContext{
			Roll:           roll,
			WoundThreshold: woundThreshold,
		}, weapon.Rules)

		if outcome.IsMortalWound {
			mortalWounds++
		} else if outcome.IsWound {
			normalWounds++
		}
	}

	result.Wounds = normalWounds
	result.MortalWounds = mortalWounds

	// ─── Phase 3: Save Rolls (normal wounds only) ─────────────────────────────

	effectiveSave := rules.EffectiveSave(defender.ArmorSave, defender.InvulSave, weapon.AP)
	unsavedWounds := 0

	for i := 0; i < normalWounds; i++ {
		roll := roller.D6()
		// Defender needs to roll >= effectiveSave to pass
		if roll < effectiveSave {
			unsavedWounds++
		}
	}

	result.UnsavedWounds = unsavedWounds

	// ─── Phase 4: Damage & Feel No Pain ───────────────────────────────────────
	//
	// Mortal wounds bypass saves but are still subject to Feel No Pain.
	// Each point of damage gets its own FNP roll.

	totalDamage := 0
	fnpThreshold := defender.FeelNoPain // 0 = no FNP

	applyDamage := func(sources int) {
		for i := 0; i < sources; i++ {
			damagePerWound := weapon.Damage
			for d := 0; d < damagePerWound; d++ {
				if fnpThreshold > 0 {
					if roller.D6() >= fnpThreshold {
						continue // FNP save passed — this point of damage negated
					}
				}
				totalDamage++
			}
		}
	}

	applyDamage(unsavedWounds)
	applyDamage(mortalWounds) // Mortal wounds deal weapon.Damage each

	result.TotalDamage = totalDamage

	// Calculate kills: how many full defender models were eliminated
	// For now, a single-model simulation — kill = damage >= defender wounds
	if defender.Wounds > 0 {
		result.DefenderKills = totalDamage / defender.Wounds
	}

	return result
}
