// Package report formats simulation statistics for human-readable output (CLI use).
package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/smackface/wh-lethality/internal/engine"
	"github.com/smackface/wh-lethality/internal/profiles"
)

// Print writes a full statistical report to w.
func Print(w io.Writer, attacker profiles.UnitProfile, defender profiles.UnitProfile, phase string, stats engine.SimStats) {
	sep := strings.Repeat("─", 60)

	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "  %s  vs  %s  [%s]\n", attacker.Label, defender.Label, phase)
	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "  Simulations:      %d\n\n", stats.Iterations)

	fmt.Fprintln(w, "  ── Phase Averages ──────────────────────────────────────")
	fmt.Fprintf(w, "  Mean Hits:          %6.2f\n", stats.MeanHits)
	fmt.Fprintf(w, "  Mean Wounds:        %6.2f  (incl. %5.2f mortal)\n",
		stats.MeanWounds+stats.MeanMortalWounds, stats.MeanMortalWounds)
	fmt.Fprintf(w, "  Mean Unsaved:       %6.2f\n\n", stats.MeanUnsavedWounds)

	fmt.Fprintln(w, "  ── Damage Output ───────────────────────────────────────")
	fmt.Fprintf(w, "  Mean Damage:        %6.2f\n", stats.MeanDamage)
	fmt.Fprintf(w, "  Std Dev:            %6.2f\n", stats.DamageStdDev)
	fmt.Fprintf(w, "  Median (50th %%):   %6d\n", stats.Damage50th)
	fmt.Fprintf(w, "  75th Percentile:   %6d\n", stats.Damage75th)
	fmt.Fprintf(w, "  95th Percentile:   %6d\n\n", stats.Damage95th)

	fmt.Fprintln(w, "  ── Kill Statistics ─────────────────────────────────────")
	fmt.Fprintf(w, "  Mean Kills:         %6.3f\n", stats.MeanKills)
	fmt.Fprintf(w, "  Kill Probability:   %6.1f%%\n\n", stats.KillProbability*100)

	fmt.Fprintln(w, "  ── Damage Distribution ─────────────────────────────────")
	printDist(w, stats.DamageDist, stats.Iterations)
	fmt.Fprintln(w, sep)
}

func printDist(w io.Writer, dist map[int]int, total int) {
	keys := make([]int, 0, len(dist))
	for k := range dist {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	maxCount := 0
	for _, c := range dist {
		if c > maxCount {
			maxCount = c
		}
	}

	barWidth := 30
	for _, dmg := range keys {
		count := dist[dmg]
		pct := float64(count) / float64(total) * 100
		bars := 0
		if maxCount > 0 {
			bars = int(float64(count) / float64(maxCount) * float64(barWidth))
		}
		bar := strings.Repeat("█", bars)
		fmt.Fprintf(w, "  %3d dmg: %-30s %6.1f%% (%d)\n", dmg, bar, pct, count)
	}
}
