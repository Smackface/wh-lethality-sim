// cmd/sim is the entry point for the wh-lethality Monte Carlo simulator.
// For now, matchups are defined in code. This will later accept profile files
// or a CLI interface once the profiles module is fleshed out.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/smackface/wh-lethality/internal/engine"
	"github.com/smackface/wh-lethality/internal/profiles"
	"github.com/smackface/wh-lethality/internal/report"
	"github.com/smackface/wh-lethality/internal/rules"
)

const iterations = 10_000

func main() {
	// ─── Define Matchup ───────────────────────────────────────────────────────
	//
	// Example from the brief:
	//   1 Sternguard Veteran (T4, 3+, 2W) with Sternguard Bolter (FR2, S4, AP1, D1, Devastating Wounds)
	//   vs
	//   1 Ork Boy (T4, 5+, 2W)

	sternguard := profiles.UnitProfile{
		Name:      "Sternguard Veteran",
		Toughness: 4,
		ArmorSave: 3,
		Wounds:    2,
	}

	sternguardBolter := profiles.WeaponProfile{
		Name:       "Sternguard Bolter",
		FiringRate: 2,
		BalSkill:   3,
		Strength:   4,
		AP:         1,
		Damage:     1,
		Rules:      []string{rules.DevastatingWounds},
	}

	orkBoy := profiles.UnitProfile{
		Name:      "Ork Boy",
		Toughness: 4,
		ArmorSave: 5,
		Wounds:    2,
		Keywords:  []string{"ORK", "INFANTRY"},
	}

	// ─── Run Simulation ───────────────────────────────────────────────────────

	fmt.Printf("Running %d simulations...\n\n", iterations)
	start := time.Now()
	stats := engine.RunSimulation(sternguard, sternguardBolter, orkBoy, iterations)
	elapsed := time.Since(start)

	report.Print(os.Stdout, sternguard, sternguardBolter, orkBoy, stats)
	fmt.Printf("\n  Completed in %v\n\n", elapsed)

	// ─── Additional Matchup: Torrent + Lethal Hits Example ───────────────────

	flameThrower := profiles.WeaponProfile{
		Name:       "Heavy Flamer",
		FiringRate: 2,
		BalSkill:   0, // irrelevant with Torrent
		Strength:   5,
		AP:         1,
		Damage:     1,
		Rules:      []string{rules.Torrent, rules.LethalHits},
	}

	fmt.Printf("Running %d simulations (Heavy Flamer example)...\n\n", iterations)
	start = time.Now()
	statsFlame := engine.RunSimulation(sternguard, flameThrower, orkBoy, iterations)
	elapsed = time.Since(start)

	report.Print(os.Stdout, sternguard, flameThrower, orkBoy, statsFlame)
	fmt.Printf("\n  Completed in %v\n\n", elapsed)

	// ─── Stealth Defender Example ─────────────────────────────────────────────

	ghostkeel := profiles.UnitProfile{
		Name:      "Ghostkeel (Stealth)",
		Toughness: 7,
		ArmorSave: 3,
		InvulSave: 5,
		Wounds:    8,
		Rules:     []string{rules.Stealth},
		Keywords:  []string{"TAU", "VEHICLE"},
	}

	boltRifle := profiles.WeaponProfile{
		Name:       "Bolt Rifle",
		FiringRate: 2,
		BalSkill:   3,
		Strength:   4,
		AP:         1,
		Damage:     1,
	}

	fmt.Printf("Running %d simulations (vs Stealth target)...\n\n", iterations)
	start = time.Now()
	statsStealth := engine.RunSimulation(sternguard, boltRifle, ghostkeel, iterations)
	elapsed = time.Since(start)

	report.Print(os.Stdout, sternguard, boltRifle, ghostkeel, statsStealth)
	fmt.Printf("\n  Completed in %v\n\n", elapsed)
}
