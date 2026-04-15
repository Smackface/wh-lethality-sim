// Package report formats simulation statistics for human-readable output.
package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/smackface/wh-lethality/internal/engine"
	"github.com/smackface/wh-lethality/internal/profiles"
)

// Print writes a full statistical report to w for the given matchup and stats.
func Print(
	w io.Writer,
	attacker profiles.UnitProfile,
	weapon profiles.WeaponProfile,
	defender profiles.UnitProfile,
	stats engine.SimStats,
) {
	sep := strings.Repeat("─", 60)

	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "  MATCHUP: %s [%s]  vs  %s\n", attacker.Label, weapon.Name, defender.Label)
	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "  Simulations:      %d\n", stats.Iterations)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "  ── Phase Averages ──────────────────────────────────────")
	fmt.Fprintf(w, "  Mean Hits:          %6.2f\n", stats.MeanHits)
	fmt.Fprintf(w, "  Mean Wounds:        %6.2f  (incl. %5.2f mortal)\n",
		stats.MeanWounds+stats.MeanMortalWounds, stats.MeanMortalWounds)
	fmt.Fprintf(w, "  Mean Unsaved:       %6.2f\n", stats.MeanUnsavedWounds)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "  ── Damage Output ───────────────────────────────────────")
	fmt.Fprintf(w, "  Mean Damage:        %6.2f\n", stats.MeanDamage)
	fmt.Fprintf(w, "  Std Dev:            %6.2f\n", stats.DamageStdDev)
	fmt.Fprintf(w, "  Median (50th %%):   %6d\n", stats.Damage50th)
	fmt.Fprintf(w, "  75th Percentile:   %6d\n", stats.Damage75th)
	fmt.Fprintf(w, "  95th Percentile:   %6d\n", stats.Damage95th)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "  ── Kill Statistics ─────────────────────────────────────")
	fmt.Fprintf(w, "  Mean Kills:         %6.3f\n", stats.MeanKills)
	fmt.Fprintf(w, "  Kill Probability:   %6.1f%%\n", stats.KillProbability*100)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "  ── Damage Distribution ─────────────────────────────────")
	printDist(w, stats.DamageDist, stats.Iterations)
	fmt.Fprintln(w, sep)
}

// printDist prints a compact ASCII bar chart of the damage distribution.
func printDist(w io.Writer, dist map[int]int, total int) {
	if len(dist) == 0 {
		return
	}

	// Sort keys
	keys := make([]int, 0, len(dist))
	for k := range dist {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Find max count for scaling
	maxCount := 0
	for _, c := range dist {
		if c > maxCount {
			maxCount = c
		}
	}

	barWidth := 30 // max bar width in chars
	for _, dmg := range keys {
		count := dist[dmg]
		pct := float64(count) / float64(total) * 100
		bars := int(float64(count) / float64(maxCount) * float64(barWidth))
		bar := strings.Repeat("█", bars)
		fmt.Fprintf(w, "  %3d dmg: %-30s %6.1f%% (%d)\n", dmg, bar, pct, count)
	}
}
