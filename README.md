# wh-lethality

**Warhammer 40K 10th Edition Monte Carlo Combat Simulator**

A Go-powered simulation engine and HTMX web UI that calculates the probability distribution of damage output for any unit matchup in Warhammer 40K 10th edition. Runs thousands of dice iterations per second, handles weapon special rules, character attachment, and detachment abilities.

---

## Features

- **Monte Carlo simulation** — 1,000 to 100,000 dice rolls per run (configurable)
- **Full attack sequence** — hit rolls → wound rolls → save rolls → damage resolution, all per the 10th edition core rules
- **Special rules implemented:** Devastating Wounds, Lethal Hits, Torrent (auto-hit), Sustained Hits X, Stealth (+1 to save)
- **Character attachment** — merge a CHARACTER unit at runtime (rule deduplication handled automatically)
- **Detachment abilities** — grant bonus rules to attacking units (Gladius Task Force, Waaagh! Band stubs included)
- **Unit library** — create, edit, and delete unit profiles stored as plain JSON files
- **Phase selection** — simulate Shooting or Melee (Fight) phase separately
- **Results include:** mean hits/wounds/damage, kill probability, damage std dev, 50th/75th/95th percentiles, and a full damage distribution bar chart
- **HTMX-powered** — zero page reloads, no JavaScript framework

---

## Requirements

- **Go 1.22+** (uses `http.ServeMux` path patterns introduced in 1.22)
- No CGO, no third-party databases, no Docker required

```bash
go version  # should be 1.22 or higher
```

---

## Quick Start

```bash
# 1. Clone
git clone https://github.com/Smackface/wh-lethality-sim.git
cd wh-lethality-sim

# 2. Run (no build step needed)
go run cmd/sim/main.go

# 3. Open
open http://localhost:8080
```

Three example units are included and ready to simulate:
- `intercessors` — Space Marine Intercessors (10-man squad, Bolt Rifles + sergeant Power Fist)
- `lieutenant` — Space Marine Lieutenant (CHARACTER, Lethal Hits ability)
- `ork-boys` — Ork Boyz (20-man mob, Choppas + Rokkit Launcha special)

---

## Command-Line Flags

```
go run cmd/sim/main.go [flags]

  -addr string     Listen address (default ":8080")
  -data string     Path to unit JSON data directory (default "data/units")
  -tmpl string     Path to templates directory (default "web/templates")
  -static string   Path to static assets directory (default "web/static")
```

---

## How It Works

### The Simulation Engine

Each run resolves the full 40K attack sequence **N times in parallel** using goroutines:

```
For each iteration:
  For each model group in the attacker:
    For each weapon the group carries:
      1. Generate attacks (fixed or dice, e.g. "D6")
      2. Hit rolls — apply Ballistic/Weapon Skill, Torrent (auto-hit), Sustained Hits X
      3. Wound rolls — compare Strength vs Toughness (standard 40K table)
                     — apply Lethal Hits (auto-wound on crit), Devastating Wounds (bypass save)
      4. Save rolls  — defender Armor Save, AP modifier, Invulnerable Save, Stealth
      5. Feel No Pain — roll FNP for each damage that gets through
      6. Accumulate total damage and wounds against defender's total wounds pool
```

After N iterations, the engine aggregates:
- Mean/std dev for hits, wounds, mortal wounds, unsaved wounds, damage
- Percentile distribution (50th, 75th, 95th)
- Full damage frequency histogram
- Kill probability (fraction of runs where total damage ≥ defender's total wound pool)
- Mean kills (mean damage ÷ total wounds)

### Unit Profiles

Units are stored as JSON files in `data/units/`. Each file is one `UnitProfile`:

```json
{
  "id": "intercessors",
  "label": "Intercessors (10)",
  "keywords": ["INFANTRY", "ADEPTUS ASTARTES", "TACTICUS"],
  "rules": [],
  "groups": [
    {
      "name": "Intercessors",
      "count": 9,
      "stats": {
        "ballistic_skill": 3,
        "weapon_skill": 3,
        "toughness": 4,
        "armor_save": 3,
        "wounds": 2
      },
      "weapons": [
        {
          "name": "Bolt Rifle",
          "type": "ranged",
          "firing_rate": 2,
          "bal_skill": 3,
          "strength": 4,
          "ap": 1,
          "damage": 1
        }
      ]
    },
    {
      "name": "Sergeant",
      "count": 1,
      "has_special_weapon": true,
      "stats": { ... },
      "weapons": [
        { "name": "Bolt Rifle", ... },
        { "name": "Power Fist", "type": "melee", ... }
      ]
    }
  ]
}
```

**Key fields:**
| Field | Description |
|---|---|
| `count` | Number of models in this group |
| `has_special_weapon` | If `true`, this group is split off (typically a sergeant/champion) |
| `type` | `"ranged"`, `"melee"`, or `"pistol"` |
| `firing_rate` | Number of attacks this weapon makes per model |
| `rules` | Array of rule names from the registry (e.g. `"Lethal Hits"`, `"Devastating Wounds"`) |
| `ap` | Armour Penetration value (subtracted from armour save, e.g. AP1 = save becomes X+1) |
| `damage` | Can be a fixed integer or a dice string like `"D3"`, `"D6"`, `"2D3"` |

### Rule Registry

All supported rules live in `internal/rules/`. Adding a new rule:

1. Add the keyword name constant to `keywords.go`
2. Add a `RuleEntry` to the registry in `registry.go` (with `Implemented: true/false`)
3. Wire the logic into `ApplyHitRules` or `ApplyWoundRules` in `registry.go`

Currently implemented rules:
| Rule | Phase | Effect |
|---|---|---|
| `Lethal Hits` | Hit | Critical hits (6) auto-wound |
| `Devastating Wounds` | Wound | Critical wounds bypass armour/invuln, deal damage as mortal wounds |
| `Torrent` | Hit | All attacks auto-hit (no hit roll) |
| `Sustained Hits X` | Hit | Each critical hit generates X additional hits |
| `Stealth` | Save | Defender gets +1 to their armour save |

Stubs (saved to profiles, not yet simulated): Twin-linked, Anti-[Keyword], Melta X, Blast, Indirect Fire, Hazardous, Lance, Precision, Rapid Fire X.

### Character Attachment

Characters can be attached to a bodyguard unit at runtime. The merged `AttachedUnit.Resolve()` call:
- Combines both units' `Groups` into a single profile
- Deduplicates rules (union of both rule sets)
- The merged unit attacks as one unit in the simulation

---

## Web UI

| Route | Description |
|---|---|
| `GET /` | Simulation form — pick attacker, defender, phase, iterations, optional character + detachment ability |
| `GET /units` | Unit library — all saved profiles as cards |
| `GET /units/new` | Create a new unit |
| `GET /units/{id}` | Edit an existing unit |
| `POST /simulate` | Runs the simulation, returns HTMX partial (results inline) |
| `DELETE /units/{id}` | Delete a unit (HTMX, no page reload) |
| `GET /api/rules` | JSON list of all rules in the registry |

---

## Project Structure

```
wh-lethality-sim/
├── cmd/sim/main.go              # Entry point — starts HTTP server
├── data/units/                  # Unit profile JSON files
│   ├── intercessors.json
│   ├── lieutenant.json
│   └── ork-boys.json
├── internal/
│   ├── dice/dice.go             # Dice roller (crypto/rand seeded, goroutine-safe)
│   ├── profiles/
│   │   ├── model.go             # UnitProfile, ModelGroup, DefenderStats types
│   │   ├── weapon.go            # WeaponProfile type
│   │   └── loader.go            # JSON file store (Load/Save/List/Delete)
│   ├── rules/
│   │   ├── keywords.go          # Rule name constants + scan helpers
│   │   └── registry.go          # Rule registry, ApplyHitRules, ApplyWoundRules, WoundThreshold
│   ├── engine/
│   │   ├── combat.go            # resolveWeaponAttack, resolveUnitAttack
│   │   ├── engine.go            # RunSimulation (goroutine-parallel dispatch)
│   │   ├── attachment.go        # AttachedUnit.Resolve() — character merging
│   │   └── result.go            # SimStats, SimConfig types
│   ├── detachment/detachment.go # Detachment abilities and stratagems
│   ├── report/report.go         # CLI-friendly text report formatter
│   └── web/
│       ├── server.go            # HTTP mux setup, template parsing, FuncMap
│       └── handlers/handlers.go # All request handlers + SimView builder
└── web/
    ├── static/                  # Static assets (CSS overrides, etc.)
    └── templates/
        ├── layout.html          # Shared header/footer defines
        ├── index.html           # Simulation page
        ├── units.html           # Unit library
        ├── unit_form.html       # Create/edit unit form
        └── partials/
            ├── sim_results.html # HTMX swap target — results display
            └── unit_details.html# HTMX swap target — unit stat summary
```

---

## Adding Units

### Via the Web UI
1. Go to `/units/new`
2. Fill in the label, keywords, rules, and paste your model groups JSON
3. Save — the unit appears in the library and simulation dropdown immediately

### Via JSON directly
Drop a `.json` file into `data/units/` following the schema above. The server picks it up on the next request (no restart needed).

---

## Roadmap

- [ ] Dynamic model group builder (replace JSON textarea with form fields)
- [ ] More rules: Twin-linked, Anti-[Keyword], Melta, Blast, Rapid Fire, Indirect Fire
- [ ] Multi-phase simulation (Shooting + Melee in one run)
- [ ] Army-vs-army simulation (multiple units per side)
- [ ] Save/compare simulation results
- [ ] Full faction unit library (Astartes, Orks, Necrons, Tyranids...)
- [ ] Export results to CSV

---

## License

MIT
