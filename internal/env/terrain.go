package env

import (
	"math"

	"flight-simulator2/internal/geometry/vector"
)

// Terrain implements an environment effect that simulates ground collision detection
// and prevents the aircraft from flying below the terrain plus a safety margin.
type Terrain struct {
	// SafetyMarginM is the minimum allowed altitude above terrain in meters
	SafetyMarginM float64
}

// GroundAltitude calculates the terrain height at a given position.
// This is a simple synthetic terrain function that can be replaced with real elevation data.
// Currently, it creates a wavy terrain pattern for demonstration purposes.
func (t Terrain) GroundAltitude(pos vector.Vec3) float64 {
	// Create a simple wavy terrain pattern
	wave1 := math.Sin(pos.X/1000) * 100
	wave2 := math.Sin((pos.X+pos.Y)/500) * 50
	return wave1 + wave2
}

// Apply enforces terrain collision detection and applies ground effect.
// If the aircraft is below the terrain plus safety margin, it will be moved up
// and its vertical velocity will be set to zero if it was descending.
func (t Terrain) Apply(dt float64, pos vector.Vec3, vel vector.Vec3) (vector.Vec3, vector.Vec3, string) {
	groundAlt := t.GroundAltitude(pos)
	minAllowedAlt := groundAlt + t.SafetyMarginM

	// Check for ground collision
	if pos.Z < minAllowedAlt {
		// Move aircraft to minimum allowed altitude
		pos.Z = minAllowedAlt

		// If descending, stop vertical movement
		if vel.Z < 0 {
			vel.Z = 0
		}

		return pos, vel, "terrain-floor: altitude clipped to safety margin"
	}

	return pos, vel, ""
}

// DefaultTerrain returns a Terrain with a reasonable default safety margin.
func DefaultTerrain() Terrain {
	return Terrain{
		SafetyMarginM: 80, // 80 meters minimum altitude above terrain
	}
}
