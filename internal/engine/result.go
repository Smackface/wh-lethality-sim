package engine

// CombatResult holds the raw outcome of a single combat simulation iteration.
type CombatResult struct {
	Hits          int // Total hits (including bonus hits from Sustained Hits, auto-wounds from Lethal Hits)
	Wounds        int // Normal wounds that proceeded to save phase
	MortalWounds  int // Wounds from Devastating Wounds (bypass armor/invul saves)
	UnsavedWounds int // Normal wounds that failed their save
	TotalDamage   int // Total damage inflicted after all saves and FNP
	DefenderKills int // Number of defender models killed (damage / defender.Wounds, for multi-model later)
}

// SimStats holds aggregated statistics across all simulation iterations.
type SimStats struct {
	Iterations int

	// Per-phase means
	MeanHits          float64
	MeanWounds        float64
	MeanMortalWounds  float64
	MeanUnsavedWounds float64
	MeanDamage        float64
	MeanKills         float64

	// Kill probability (P(kills >= 1))
	KillProbability float64

	// Damage distribution (index = damage dealt, value = how many sims produced that)
	DamageDist map[int]int

	// Standard deviation of damage
	DamageStdDev float64

	// Percentiles
	Damage50th int // Median
	Damage75th int
	Damage95th int
}
