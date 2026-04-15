package engine

// CombatResult holds the raw outcome of a single combat simulation iteration.
type CombatResult struct {
	Hits            int // Total hits (incl. bonus hits from Sustained Hits, auto-wounds from Lethal Hits)
	Wounds          int // Normal wounds that proceeded to save phase
	MortalWounds    int // Mortal wounds from Devastating Wounds (bypass armor/invul saves)
	UnsavedWounds   int // Normal wounds that failed their save
	TotalDamage     int // Total damage inflicted on defender after all saves and FNP
	DefenderKills   int // Defender models killed (TotalDamage / defender.Wounds)
	HazardousSelfMW int // Mortal wounds the attacker suffers from Hazardous weapons (after own FNP)
}

// SimStats holds aggregated statistics across all simulation iterations.
type SimStats struct {
	Iterations int

	// Per-phase means
	MeanHits            float64
	MeanWounds          float64
	MeanMortalWounds    float64
	MeanUnsavedWounds   float64
	MeanDamage          float64
	MeanKills           float64
	MeanHazardousSelfMW float64 // mean attacker self-wounds from Hazardous

	// Kill probability (P(kills >= 1))
	KillProbability float64

	// Damage distribution (key = damage dealt, value = iteration count)
	DamageDist map[int]int

	// Standard deviation of damage
	DamageStdDev float64

	// Percentiles
	Damage50th int // Median
	Damage75th int
	Damage95th int
}
