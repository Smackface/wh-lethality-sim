// Package dice provides a dice roller seeded independently per instance.
// Each goroutine should create its own Roller so simulations don't share RNG state.
package dice

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	mathrand "math/rand"
)

// Roller is a self-contained dice roller. Safe for use within a single goroutine.
// Do NOT share a Roller across goroutines — create one per goroutine.
type Roller struct {
	rng *mathrand.Rand
}

// New creates a Roller seeded from crypto/rand for high-quality entropy.
func New() *Roller {
	return &Roller{rng: mathrand.New(mathrand.NewSource(cryptoSeed()))}
}

// NewSeeded creates a Roller with an explicit seed (useful for reproducible tests).
func NewSeeded(seed int64) *Roller {
	return &Roller{rng: mathrand.New(mathrand.NewSource(seed))}
}

// D6 rolls a single six-sided die, returning a value in [1, 6].
func (r *Roller) D6() int {
	return int(r.rng.Int63n(6)) + 1
}

// D3 rolls a single three-sided die, returning a value in [1, 3].
func (r *Roller) D3() int {
	return int(r.rng.Int63n(3)) + 1
}

// RollD6s rolls n six-sided dice and returns all results.
func (r *Roller) RollD6s(n int) []int {
	results := make([]int, n)
	for i := range results {
		results[i] = r.D6()
	}
	return results
}

// cryptoSeed reads 8 bytes from crypto/rand and returns them as an int64.
func cryptoSeed() int64 {
	var b [8]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		// Extremely rare fallback — just use a constant with some spread
		return int64(0x7EADBEEFCAFEBABE)
	}
	return int64(binary.LittleEndian.Uint64(b[:]))
}
