package main

import (
	"fmt"
	"math"
	"time"

	"flight-simulator2/internal/env"
	"flight-simulator2/internal/geometry/vector"
)

func main() {
	fmt.Println("Flight Simulator - Environment Effects Demo")
	fmt.Println("----------------------------------------")

	// Create environment effects
	wind := env.Wind{Wx: 5.0, Wy: 2.0} // 5 m/s east, 2 m/s north
	terrain := env.Terrain{SafetyMarginM: 80.0}

	// Create a chain of effects (order matters!)
	environment := env.Chain{
		Effects: []env.Environment{wind, terrain},
	}

	// Initial aircraft state - starting closer to terrain
	pos := vector.Vec3{X: 0, Y: 0, Z: 200}   // Start at 200m altitude (closer to terrain)
	vel := vector.Vec3{X: 100, Y: 0, Z: -15} // Moving east at 100 m/s, descending at 15 m/s

	fmt.Println("Starting simulation...")
	fmt.Printf("Initial position: (%.1f, %.1f, %.1f)\n", pos.X, pos.Y, pos.Z)
	fmt.Printf("Initial velocity (Air): (%.1f, %.1f, %.1f) m/s\n", vel.X, vel.Y, vel.Z)

	// Run simulation for 10 seconds
	dt := 1.0 // Time step in seconds

	prevPos := pos // used to compute ground velocity from actual motion

	for t := 0.0; t <= 10.0; t += dt {
		// Apply environment effects
		newPos, newVel, warning := environment.Apply(dt, pos, vel)

		// Update position based on velocity (simple Euler integration)
		newPos.X += newVel.X * dt
		newPos.Y += newVel.Y * dt
		newPos.Z += newVel.Z * dt

		// Compute ground velocity from actual motion: (pos_now - pos_prev) / dt
		groundVel := vector.Vec3{
			X: (newPos.X - prevPos.X) / dt,
			Y: (newPos.Y - prevPos.Y) / dt,
			Z: (newPos.Z - prevPos.Z) / dt,
		}

		groundSpeed := math.Sqrt(groundVel.X*groundVel.X + groundVel.Y*groundVel.Y)

		// Estimate wind from difference (XY only) between ground and air velocity
		// (This is just for demonstration/debug)
		windEst := vector.Vec3{
			X: groundVel.X - newVel.X,
			Y: groundVel.Y - newVel.Y,
			Z: 0,
		}

		groundAlt := terrain.GroundAltitude(newPos)

		fmt.Printf("\nTime: %.1fs\n", t+dt)
		fmt.Printf("Position: (%.1f, %.1f, %.1f) m\n", newPos.X, newPos.Y, newPos.Z)
		fmt.Printf("Ground altitude: %.1f m\n", groundAlt)
		fmt.Printf("Altitude AGL: %.1f m\n", newPos.Z-groundAlt)

		fmt.Printf("Velocity (Air):    (%.1f, %.1f, %.1f) m/s\n", newVel.X, newVel.Y, newVel.Z)
		fmt.Printf("Velocity (Ground): (%.1f, %.1f, %.1f) m/s | GroundSpeed=%.1f m/s\n",
			groundVel.X, groundVel.Y, groundVel.Z, groundSpeed)

		fmt.Printf("Estimated wind (XY): (%.1f, %.1f) m/s\n", windEst.X, windEst.Y)

		if warning != "" {
			fmt.Printf("🚨 WARNING: %s\n", warning)
		} else {
			fmt.Println("✅ No terrain warnings")
		}

		// Update for next iteration
		prevPos = newPos
		pos, vel = newPos, newVel

		time.Sleep(500 * time.Millisecond) // Slow down for demonstration
	}
}
