package engine

import (
	"math"
	"sort"
	"sync"

	"github.com/smackface/wh-lethality/internal/dice"
	"github.com/smackface/wh-lethality/internal/profiles"
)

// RunSimulation dispatches `iterations` combat resolutions concurrently using goroutines.
// Each goroutine receives its own freshly-seeded dice.Roller to ensure independent randomness.
// Returns aggregated SimStats across all iterations.
func RunSimulation(
	attacker profiles.UnitProfile,
	weapon profiles.WeaponProfile,
	defender profiles.UnitProfile,
	iterations int,
) SimStats {
	results := make([]CombatResult, iterations)

	var wg sync.WaitGroup
	wg.Add(iterations)

	for i := 0; i < iterations; i++ {
		go func(idx int) {
			defer wg.Done()
			roller := dice.New() // fresh crypto-seeded roller per goroutine
			results[idx] = ResolveCombat(attacker, weapon, defender, roller)
		}(i)
	}

	wg.Wait()
	return aggregate(results, defender)
}

// aggregate computes statistics from a slice of raw combat results.
func aggregate(results []CombatResult, defender profiles.UnitProfile) SimStats {
	n := len(results)
	if n == 0 {
		return SimStats{}
	}

	stats := SimStats{
		Iterations: n,
		DamageDist: make(map[int]int),
	}

	// Accumulate totals
	var (
		totalHits          float64
		totalWounds        float64
		totalMortals       float64
		totalUnsaved       float64
		totalDamage        float64
		totalKills         float64
		killCount          int
	)

	damages := make([]int, n) // for percentile and stddev calculation

	for i, r := range results {
		totalHits += float64(r.Hits)
		totalWounds += float64(r.Wounds)
		totalMortals += float64(r.MortalWounds)
		totalUnsaved += float64(r.UnsavedWounds)
		totalDamage += float64(r.TotalDamage)
		totalKills += float64(r.DefenderKills)

		if r.DefenderKills >= 1 {
			killCount++
		}

		stats.DamageDist[r.TotalDamage]++
		damages[i] = r.TotalDamage
	}

	fN := float64(n)
	stats.MeanHits = totalHits / fN
	stats.MeanWounds = totalWounds / fN
	stats.MeanMortalWounds = totalMortals / fN
	stats.MeanUnsavedWounds = totalUnsaved / fN
	stats.MeanDamage = totalDamage / fN
	stats.MeanKills = totalKills / fN
	stats.KillProbability = float64(killCount) / fN

	// Standard deviation of damage
	mean := stats.MeanDamage
	var variance float64
	for _, d := range damages {
		diff := float64(d) - mean
		variance += diff * diff
	}
	variance /= fN
	stats.DamageStdDev = math.Sqrt(variance)

	// Percentiles (sort damages)
	sort.Ints(damages)
	stats.Damage50th = damages[int(math.Floor(float64(n)*0.50))]
	stats.Damage75th = damages[int(math.Floor(float64(n)*0.75))]
	stats.Damage95th = damages[int(math.Floor(float64(n)*0.95))]

	return stats
}
