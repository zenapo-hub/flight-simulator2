package env

import (
	"flight-simulator2/internal/geometry/vector"
)

// Environment is an interface for applying environmental effects to the aircraft.
// Each implementation can modify the aircraft's velocity or position based on
// environmental factors like wind, terrain, or other atmospheric conditions.
type Environment interface {
	// Apply takes the current position and velocity of the aircraft and returns
	// the modified position, velocity, and an optional warning message.
	// The dt parameter is the time step in seconds since the last update.
	Apply(dt float64, pos vector.Vec3, vel vector.Vec3) (vector.Vec3, vector.Vec3, string)
}

// Chain is a composite environment that applies multiple environment effects in sequence.
type Chain struct {
	Effects []Environment
}

// Apply applies all environment effects in the chain, in order.
// The position and velocity are passed through each effect in sequence,
// with the output of one effect becoming the input to the next.
// The last non-empty warning message is returned.
func (c *Chain) Apply(dt float64, pos vector.Vec3, vel vector.Vec3) (vector.Vec3, vector.Vec3, string) {
	var warning string
	for _, effect := range c.Effects {
		newPos, newVel, w := effect.Apply(dt, pos, vel)
		if w != "" {
			warning = w
		}
		pos, vel = newPos, newVel
	}
	return pos, vel, warning
}

// NoOp is an environment that does nothing.
var NoOp Environment = noOpEnv{}

type noOpEnv struct{}

func (noOpEnv) Apply(dt float64, pos, vel vector.Vec3) (vector.Vec3, vector.Vec3, string) {
	return pos, vel, ""
}
