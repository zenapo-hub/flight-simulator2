package env

import (
	"math"

	"flight-simulator2/internal/geometry/vector"
)

// Wind represents a constant wind vector in the environment.
// The wind is specified in meters per second in the east (Wx) and north (Wy) directions.
type Wind struct {
	// Wx is the eastward component of the wind in m/s (positive = east, negative = west)
	Wx float64
	// Wy is the northward component of the wind in m/s (positive = north, negative = south)
	Wy float64
}

// Apply applies wind as a constant ground drift.
// We modify position directly (ground track), without changing the aircraft's own velocity.
func (w Wind) Apply(dt float64, pos vector.Vec3, vel vector.Vec3) (vector.Vec3, vector.Vec3, string) {
	// Wind affects ground track but not the aircraft's airspeed
	drift := vector.Vec3{X: w.Wx * dt, Y: w.Wy * dt}
	return pos.Add(drift), vel, ""
}

// Calm returns a Wind with zero velocity (no wind).
func Calm() Wind {
	return Wind{Wx: 0, Wy: 0}
}

// FromSpeedAndDir creates a Wind from a speed (m/s) and direction (degrees).
// Direction is in degrees clockwise from north (0° = north, 90° = east).
func FromSpeedAndDir(speed, directionDeg float64) Wind {
	// Convert direction from degrees to radians
	rad := (90 - directionDeg) * math.Pi / 180 // Convert to math angle (0° = east, 90° = north)
	return Wind{
		Wx: speed * math.Cos(rad),
		Wy: speed * math.Sin(rad),
	}
}
