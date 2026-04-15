package engine

import (
	"github.com/smackface/wh-lethality/internal/profiles"
)

// AttachedUnit represents a bodyguard unit with an optional attached character.
// In WH40K 10th edition, a CHARACTER with the Leader ability joins a unit,
// merging their model(s) in and granting their rules to the whole unit.
type AttachedUnit struct {
	Bodyguard *profiles.UnitProfile
	Character *profiles.UnitProfile // nil = no character attached
}

// Resolve returns a single merged UnitProfile representing the combined unit
// as it exists on the table for this simulation:
//   - Character's model groups are appended to the bodyguard's groups
//   - Character's rules are merged into the unit's rule list (deduped)
//   - Keywords are merged (deduped)
//
// The source profiles are never modified.
func (a *AttachedUnit) Resolve() profiles.UnitProfile {
	if a.Character == nil {
		return *a.Bodyguard
	}

	merged := profiles.UnitProfile{
		ID:    a.Bodyguard.ID + "+" + a.Character.ID,
		Label: a.Bodyguard.Label + " + " + a.Character.Label,
	}

	// Merge model groups (bodyguard first, then character models)
	merged.Groups = append(merged.Groups, a.Bodyguard.Groups...)
	merged.Groups = append(merged.Groups, a.Character.Groups...)

	// Merge rules (deduplicated)
	merged.Rules = mergeStrings(a.Bodyguard.Rules, a.Character.Rules)

	// Merge keywords (deduplicated)
	merged.Keywords = mergeStrings(a.Bodyguard.Keywords, a.Character.Keywords)

	return merged
}

// ApplyDetachmentRules returns a copy of the unit with additional rules added
// from a detachment ability (or stratagem). Source is not modified.
func ApplyDetachmentRules(unit profiles.UnitProfile, additionalRules []string) profiles.UnitProfile {
	copy := unit
	copy.Rules = mergeStrings(unit.Rules, additionalRules)
	return copy
}

// mergeStrings combines two string slices, deduplicating entries.
func mergeStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, s := range append(a, b...) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
