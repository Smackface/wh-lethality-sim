// Package rules defines, interprets, and resolves all WH40K 10th edition special rules
// as they apply to combat. The engine calls into this package to determine how rules
// interact during each phase of attack resolution.
//
// Adding a new rule:
//  1. Add its string constant here.
//  2. If it affects a specific phase, add a branch in the appropriate Apply* function.
//  3. Add documentation explaining when it triggers and what it does.
package rules

import (
	"fmt"
	"strconv"
	"strings"
)

// ─── Weapon Rule Keywords ─────────────────────────────────────────────────────

const (
	// DevastatingWounds — Critical wounds (normally on 6, or lower via Anti-[Keyword])
	// inflict a Mortal Wound instead of a normal wound, bypassing armour and
	// invulnerable saves. FNP still applies.
	DevastatingWounds = "Devastating Wounds"

	// LethalHits — Unmodified To Hit rolls of 6 cause an automatic wound without
	// needing a To Wound roll. The wound still proceeds to the save phase normally.
	// Interaction with Sustained Hits X: on a natural 6, the original hit auto-wounds
	// AND X bonus hits are generated — those bonus hits still need wound rolls.
	LethalHits = "Lethal Hits"

	// Torrent — This weapon auto-hits; no To Hit roll is made.
	Torrent = "Torrent"

	// SustainedHitsPrefix — To Hit rolls of 6 generate X bonus hits.
	// Full rule string: "Sustained Hits X" (e.g. "Sustained Hits 2").
	SustainedHitsPrefix = "Sustained Hits"

	// AntiPrefix — Anti-[KEYWORD] (X+): To Wound rolls of X+ are always critical
	// wounds against units with the specified keyword. Critical wounds trigger
	// Devastating Wounds even on sub-6 rolls when this rule applies.
	// Full rule string: "Anti-KEYWORD X" (e.g. "Anti-INFANTRY 4").
	AntiPrefix = "Anti"

	// Blast — For each 5 models in the target unit, this weapon gains 1 bonus attack.
	Blast = "Blast"

	// Hazardous — After firing, roll 1D6. On a 1, the bearer suffers 3 Mortal Wounds.
	// The attacker's FNP (if any) can save each of these mortal wounds.
	Hazardous = "Hazardous"

	// IgnoresCover — Attacker does not suffer the -1 AP penalty against defenders
	// who benefit from Cover.
	IgnoresCover = "Ignores Cover"

	// TwinLinked — Re-roll wound rolls of 1.
	TwinLinked = "Twin-linked"

	// RapidFirePrefix — Within half range, this weapon gains X bonus attacks.
	// Full rule string: "Rapid Fire X" (e.g. "Rapid Fire 2").
	RapidFirePrefix = "Rapid Fire"

	// MeltaPrefix — Within half range, add X to the damage of each successful attack.
	// Full rule string: "Melta X" (e.g. "Melta 2").
	MeltaPrefix = "Melta"

	// IndirectFire — Can target units not visible to the bearer, but -1 to all hit rolls.
	IndirectFire = "Indirect Fire"

	// Precision — Unmodified wound rolls of 6 can be allocated to CHARACTER models
	// within the target unit, bypassing the normal closest-model rule.
	// In the current sim model this is a target-selection rule and does not affect
	// damage output totals.
	Precision = "Precision"
)

// ─── Unit/Model Rule Keywords ─────────────────────────────────────────────────

const (
	// Stealth — Attackers targeting this unit suffer -1 to their To Hit rolls.
	Stealth = "Stealth"

	// FightsFirst — This unit fights before other units in the Fight phase,
	// even if they did not charge. (Phase ordering; not simulated in damage calc.)
	FightsFirst = "Fights First"

	// TODO: Cover — +1 to armor saves vs ranged (handled as sim config flag instead)
	// TODO: Leader — grants special rules to attached unit
	// TODO: Lone Operative — cannot be targeted unless attacker is within 12"
)

// ─── Parsing Helpers ─────────────────────────────────────────────────────────

// ParseAntiRule extracts the keyword and critical wound threshold from a rule
// string of the form "Anti-KEYWORD THRESHOLD" (e.g. "Anti-INFANTRY 4").
// Returns ok=false if the string is not a valid Anti rule.
func ParseAntiRule(name string) (keyword string, threshold int, ok bool) {
	if !strings.HasPrefix(name, "Anti-") {
		return "", 0, false
	}
	rest := name[len("Anti-"):] // e.g. "INFANTRY 4"
	idx := strings.LastIndex(rest, " ")
	if idx < 0 {
		return "", 0, false
	}
	kw := strings.TrimSpace(rest[:idx])
	tStr := strings.TrimSpace(rest[idx+1:])
	t, err := strconv.Atoi(tStr)
	if err != nil || t < 2 || t > 6 {
		return "", 0, false
	}
	return kw, t, true
}

// scanSustainedHits is a helper to extract X from "Sustained Hits X".
func scanSustainedHits(rule string, out *int) (int, error) {
	n, err := fmt.Sscanf(rule, "Sustained Hits %d", out)
	return n, err
}

// ParseRapidFire extracts X from "Rapid Fire X" (e.g. "Rapid Fire 2" → 2, true).
func ParseRapidFire(name string) (int, bool) {
	var x int
	if n, _ := fmt.Sscanf(name, "Rapid Fire %d", &x); n == 1 {
		return x, true
	}
	return 0, false
}

// ParseMelta extracts X from "Melta X" (e.g. "Melta 2" → 2, true).
func ParseMelta(name string) (int, bool) {
	var x int
	if n, _ := fmt.Sscanf(name, "Melta %d", &x); n == 1 {
		return x, true
	}
	return 0, false
}

// ─── Phase Contexts ───────────────────────────────────────────────────────────

// HitContext holds all state relevant to a single To Hit roll.
type HitContext struct {
	Roll         int  // The raw D6 result
	HitThreshold int  // The roll needed to hit (after any modifiers)
	IsTorrent    bool // Whether this weapon auto-hits
}

// HitOutcome is returned by ApplyHitRules to describe what a single hit roll produced.
type HitOutcome struct {
	IsHit       bool // The roll succeeded as a normal hit (requires wound roll)
	IsAutoWound bool // The roll triggered Lethal Hits (skip wound roll, counts as wound)
	BonusHits   int  // Extra hits from Sustained Hits X (always require wound rolls)
}

// WoundContext holds all state relevant to a single To Wound roll.
type WoundContext struct {
	Roll                   int // The raw D6 result
	WoundThreshold         int // The roll needed to wound
	CriticalWoundThreshold int // Roll for a critical wound; defaults to 6 (lowered by Anti-[Keyword])
}

// WoundOutcome is returned by ApplyWoundRules.
type WoundOutcome struct {
	IsWound       bool // Normal wound that proceeds to save
	IsMortalWound bool // Critical wound with Devastating Wounds: bypasses armor/invul saves
}

// ─── Phase Resolution ─────────────────────────────────────────────────────────

// ApplyHitRules interprets a single To Hit roll given the weapon's effective rules.
//
// Critical interactions:
//   - Lethal Hits + Sustained Hits X: a natural 6 produces an auto-wound (Lethal Hits)
//     AND X bonus hits (Sustained Hits). The bonus hits still need wound rolls.
//   - Torrent: bypasses all hit rolls entirely (handled by caller setting IsTorrent).
func ApplyHitRules(ctx HitContext, effectiveRules []string) HitOutcome {
	if ctx.IsTorrent {
		return HitOutcome{IsHit: true}
	}

	roll := ctx.Roll

	// Scan all rules in one pass to avoid repeated iteration
	hasLethalHits := false
	sustainedBonus := 0
	for _, r := range effectiveRules {
		if r == LethalHits {
			hasLethalHits = true
		}
		var x int
		if _, err := scanSustainedHits(r, &x); err == nil && x > sustainedBonus {
			sustainedBonus = x
		}
	}

	if roll == 6 {
		if hasLethalHits {
			// Natural 6 with Lethal Hits: auto-wound AND generate Sustained bonus hits.
			// The auto-wound skips the wound roll. Bonus hits (if any) still need wound rolls.
			return HitOutcome{IsAutoWound: true, BonusHits: sustainedBonus}
		}
		// Sustained Hits only: bonus hits are generated; original hit still needs wound roll.
		isHit := roll >= ctx.HitThreshold
		return HitOutcome{IsHit: isHit, BonusHits: sustainedBonus}
	}

	// Non-6 roll: standard hit check, no bonus hits
	return HitOutcome{IsHit: roll >= ctx.HitThreshold}
}

// ApplyWoundRules interprets a single To Wound roll given the weapon's effective rules.
//
// Critical wound threshold (ctx.CriticalWoundThreshold) is normally 6, but is lowered
// by Anti-[Keyword] rules when the defender has the matching keyword. Devastating Wounds
// trigger on any critical wound, regardless of what lowered the threshold.
func ApplyWoundRules(ctx WoundContext, effectiveRules []string) WoundOutcome {
	critThresh := ctx.CriticalWoundThreshold
	if critThresh == 0 {
		critThresh = 6 // default: critical wound only on natural 6
	}

	// Critical wound (= natural >= critThresh)
	if ctx.Roll >= critThresh {
		for _, r := range effectiveRules {
			if r == DevastatingWounds {
				// Mortal wound — bypasses armour and invulnerable saves
				return WoundOutcome{IsMortalWound: true}
			}
		}
		// Critical wound without Devastating Wounds → still a normal wound
		return WoundOutcome{IsWound: true}
	}

	// Normal wound check
	if ctx.Roll >= ctx.WoundThreshold {
		return WoundOutcome{IsWound: true}
	}

	return WoundOutcome{}
}

// CriticalWoundThreshold returns the minimum roll for a critical wound, given
// the effective rules and the defending unit's keywords.
// Normally this is 6; Anti-[KEYWORD] (X+) lowers it when keywords match.
func CriticalWoundThreshold(effectiveRules []string, defKeywords []string) int {
	lowest := 6
	kwSet := make(map[string]struct{}, len(defKeywords))
	for _, kw := range defKeywords {
		kwSet[kw] = struct{}{}
	}
	for _, r := range effectiveRules {
		if kw, thresh, ok := ParseAntiRule(r); ok {
			if _, has := kwSet[kw]; has && thresh < lowest {
				lowest = thresh
			}
		}
	}
	return lowest
}

// WoundThreshold returns the minimum D6 roll needed to wound based on S vs T.
//
//	S ≥ 2×T  →  2+
//	S  >  T  →  3+
//	S ==  T  →  4+
//	S  <  T  →  5+
//	S ≤ T/2  →  6+
func WoundThreshold(strength, toughness int) int {
	switch {
	case strength >= toughness*2:
		return 2
	case strength > toughness:
		return 3
	case strength == toughness:
		return 4
	case strength > toughness/2:
		return 5
	default:
		return 6
	}
}

// EffectiveSave returns the save threshold the defender must meet or beat after AP.
// If the model has an invulnerable save, the better of the two is returned.
// Returns 7 if the save is impossible (worse than 6+).
func EffectiveSave(armorSave, invulSave, ap int) int {
	modifiedArmor := armorSave + ap
	effective := modifiedArmor

	// Invulnerable save is unaffected by AP; use it if better
	if invulSave > 0 && invulSave < effective {
		effective = invulSave
	}

	return effective
}
