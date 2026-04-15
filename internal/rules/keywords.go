// Package rules defines, interprets, and resolves all WH40K 10th edition special rules
// as they apply to combat. The engine calls into this package to determine how rules
// interact during each phase of attack resolution.
//
// Adding a new rule:
//  1. Add its string constant here.
//  2. If it affects a specific phase, add a branch in the appropriate Apply* function.
//  3. Add documentation explaining when it triggers and what it does.
package rules

import "fmt"

// ─── Weapon Rule Keywords ────────────────────────────────────────────────────

const (
	// DevastatingWounds — To Wound rolls of 6 cause a Mortal Wound instead of
	// a normal wound. The enemy cannot use their armor save or invulnerable save
	// against Mortal Wounds (they may still use Feel No Pain).
	DevastatingWounds = "Devastating Wounds"

	// LethalHits — To Hit rolls of 6 cause an automatic wound without needing
	// a To Wound roll. The wound still proceeds to the save phase normally.
	LethalHits = "Lethal Hits"

	// Torrent — This weapon auto-hits; no To Hit roll is made.
	// All attacks automatically count as hits.
	Torrent = "Torrent"

	// SustainedHitsPrefix — To Hit rolls of 6 generate bonus hits equal to X.
	// Full rule string format: "Sustained Hits X" (e.g. "Sustained Hits 1").
	// Use WeaponProfile.SustainedHitsBonus() to extract X.
	SustainedHitsPrefix = "Sustained Hits"

	// TODO: Melta X — add X to damage if target is within half range.
	// TODO: Anti-[Keyword] X+ — wound roll of X+ is an auto-wound vs target keyword.
	// TODO: Blast — minimum 3 shots if target unit has 5+ models.
	// TODO: Indirect Fire — does not require line of sight; -1 to hit.
	// TODO: Precision — wounds on unmodified 6s can target Character models.
	// TODO: Twin-linked — re-roll wound rolls.
	// TODO: Rapid Fire X — double firing rate if within half range.
)

// ─── Unit/Model Rule Keywords ─────────────────────────────────────────────────

const (
	// Stealth — Attackers targeting this unit suffer -1 to their To Hit rolls.
	// This effectively increases the hit threshold by 1 (e.g. 3+ becomes 4+).
	Stealth = "Stealth"

	// TODO: Cover — same effect as Stealth (+1 to armor saves vs ranged).
	// TODO: Leader — grants special rules to attached unit.
	// TODO: Lone Operative — cannot be targeted unless attacker is within 12".
)

// ─── Phase Contexts ───────────────────────────────────────────────────────────
// These structs are passed to Apply* functions so rules have full context
// about the attack being resolved.

// HitContext holds all state relevant to a single To Hit roll.
type HitContext struct {
	Roll         int  // The raw D6 result
	HitThreshold int  // The roll needed to hit (after any modifiers)
	IsTorrent    bool // Whether this weapon auto-hits
}

// HitOutcome is returned by ApplyHitRules to describe what a single hit roll produced.
type HitOutcome struct {
	IsHit       bool // The roll succeeded as a normal hit
	IsAutoWound bool // The roll triggered Lethal Hits (skip wound roll, counts as wound)
	BonusHits   int  // Extra hits from Sustained Hits X
}

// WoundContext holds all state relevant to a single To Wound roll.
type WoundContext struct {
	Roll          int // The raw D6 result
	WoundThreshold int // The roll needed to wound
}

// WoundOutcome is returned by ApplyWoundRules.
type WoundOutcome struct {
	IsWound       bool // Normal wound that proceeds to save
	IsMortalWound bool // Mortal wound: bypasses armor/invul saves
}

// ─── Phase Resolution ─────────────────────────────────────────────────────────

// ApplyHitRules interprets a single To Hit roll given the weapon's rules.
// Returns a HitOutcome describing what the roll produced.
func ApplyHitRules(ctx HitContext, weaponRules []string) HitOutcome {
	if ctx.IsTorrent {
		return HitOutcome{IsHit: true}
	}

	roll := ctx.Roll

	// Check for Lethal Hits (natural 6 = auto-wound, not a hit)
	if roll == 6 {
		for _, r := range weaponRules {
			if r == LethalHits {
				return HitOutcome{IsAutoWound: true}
			}
		}
	}

	// Check for Sustained Hits (natural 6 = bonus hits)
	bonusHits := 0
	if roll == 6 {
		for _, r := range weaponRules {
			var x int
			if _, err := scanSustainedHits(r, &x); err == nil {
				bonusHits = x
				break
			}
		}
	}

	// Normal hit check
	isHit := roll >= ctx.HitThreshold

	if isHit {
		return HitOutcome{IsHit: true, BonusHits: bonusHits}
	}
	return HitOutcome{BonusHits: bonusHits} // miss, but may still generate bonus hits on 6
}

// ApplyWoundRules interprets a single To Wound roll given the weapon's rules.
func ApplyWoundRules(ctx WoundContext, weaponRules []string) WoundOutcome {
	// Devastating Wounds: natural 6 to wound = mortal wound
	if ctx.Roll == 6 {
		for _, r := range weaponRules {
			if r == DevastatingWounds {
				return WoundOutcome{IsMortalWound: true}
			}
		}
	}

	if ctx.Roll >= ctx.WoundThreshold {
		return WoundOutcome{IsWound: true}
	}
	return WoundOutcome{}
}

// WoundThreshold returns the minimum D6 roll needed to wound based on S vs T.
//
//	S ≥ 2×T  →  2+
//	S  >  T  →  3+
//	S ==  T  →  4+
//	S  <  T  →  5+
//	S ≤ T/2  →  6+  (integer division, rounds down)
func WoundThreshold(strength, toughness int) int {
	switch {
	case strength >= toughness*2:
		return 2
	case strength > toughness:
		return 3
	case strength == toughness:
		return 4
	case strength > toughness/2: // strength < toughness but not halved
		return 5
	default: // strength <= toughness/2
		return 6
	}
}

// scanSustainedHits is a helper to extract X from "Sustained Hits X".
func scanSustainedHits(rule string, out *int) (int, error) {
	n, err := fmt.Sscanf(rule, "Sustained Hits %d", out)
	return n, err
}

// EffectiveSave returns the save threshold the defender must meet or beat after AP.
// If the model has an invulnerable save, the better of the two is returned.
// Returns 7 if the save is impossible (worse than 6+).
func EffectiveSave(armorSave, invulSave, ap int) int {
	// AP degrades the armor save (AP-1 means 3+ becomes 4+)
	modifiedArmor := armorSave + ap
	effective := modifiedArmor

	// Invulnerable save is unaffected by AP; use it if better
	if invulSave > 0 && invulSave < effective {
		effective = invulSave
	}

	return effective
}
