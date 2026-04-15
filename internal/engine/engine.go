package engine

import (
	"math"
	"sort"
	"sync"

	"github.com/smackface/wh-lethality/internal/dice"
	"github.com/smackface/wh-lethality/internal/profiles"
)

// SimConfig describes a single simulation run request.
type SimConfig struct {
	Attacker   profiles.UnitProfile
	Defender   profiles.UnitProfile
	Phase      string // "shooting" or "melee"
	Iterations int
}

// RunSimulation dispatches Iterations concurrent combat resolutions and returns
// aggregated statistics. Each goroutine gets its own freshly seeded dice.Roller.
func RunSimulation(cfg SimConfig) SimStats {
	if cfg.Iterations <= 0 {
		cfg.Iterations = 10_000
	}

	results := make([]CombatResult, cfg.Iterations)
	var wg sync.WaitGroup
	wg.Add(cfg.Iterations)

	for i := 0; i < cfg.Iterations; i++ {
		go func(idx int) {
			defer wg.Done()
			roller := dice.New()
			results[idx] = resolveUnitAttack(cfg.Attacker, cfg.Defender, cfg.Phase, roller)
		}(i)
	}

	wg.Wait()
	return aggregate(results, cfg.Defender.PrimaryStats())
}

// aggregate computes SimStats from raw results.
func aggregate(results []CombatResult, defStats profiles.ModelStats) SimStats {
	n := len(results)
	if n == 0 {
		return SimStats{}
	}

	stats := SimStats{
		Iterations: n,
		DamageDist: make(map[int]int),
	}

	var (
		sumHits, sumWounds, sumMortals, sumUnsaved, sumDamage, sumKills float64
		killCount                                                        int
	)
	damages := make([]int, n)

	for i, r := range results {
		sumHits += float64(r.Hits)
		sumWounds += float64(r.Wounds)
		sumMortals += float64(r.MortalWounds)
		sumUnsaved += float64(r.UnsavedWounds)
		sumDamage += float64(r.TotalDamage)
		sumKills += float64(r.DefenderKills)
		if r.DefenderKills >= 1 {
			killCount++
		}
		stats.DamageDist[r.TotalDamage]++
		damages[i] = r.TotalDamage
	}

	fN := float64(n)
	stats.MeanHits = sumHits / fN
	stats.MeanWounds = sumWounds / fN
	stats.MeanMortalWounds = sumMortals / fN
	stats.MeanUnsavedWounds = sumUnsaved / fN
	stats.MeanDamage = sumDamage / fN
	stats.MeanKills = sumKills / fN
	stats.KillProbability = float64(killCount) / fN

	mean := stats.MeanDamage
	var variance float64
	for _, d := range damages {
		diff := float64(d) - mean
		variance += diff * diff
	}
	stats.DamageStdDev = math.Sqrt(variance / fN)

	sort.Ints(damages)
	stats.Damage50th = damages[int(math.Floor(float64(n)*0.50))]
	stats.Damage75th = damages[int(math.Floor(float64(n)*0.75))]
	stats.Damage95th = damages[int(math.Floor(float64(n)*0.95))]

	return stats
}
