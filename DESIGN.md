# Design Note — Airborne Flight Simulator Backend

## Overview

This service simulates a single aircraft in real-time and exposes a REST API to control it.
The aircraft state is updated continuously by a tick-based simulation engine.

Supported commands:
- Go-To point (lat/lon/alt)
- Trajectory (list of waypoints, optional loop)
- Hold / Stop

The system is designed to demonstrate:
- concurrency and clean architecture
- correctness and race-free state handling
- modular environment effects
- streaming observability via SSE

---

## High-Level Components

- **Simulation Engine (internal/sim)**
  - single goroutine actor that owns aircraft state
  - runs a continuous tick loop (20Hz default)
  - applies active command control law
  - applies environment effects (wind, terrain)
  - publishes state snapshots

- **HTTP API (internal/api)**
  - REST endpoints for commands and state
  - SSE endpoint for streaming state updates
  - does not directly touch aircraft state

- **Environment Effects (internal/env)**
  - plugin-style interface: `Apply(dt, pos, vel) -> (pos, vel, warning)`
  - wind drift and terrain-floor safety
  - supports chaining multiple effects in order

---

## Concurrency Model (Actor Pattern)

The engine owns the state and is the only writer.
All other goroutines communicate with it using channels.

### Channels
- `cmdCh`: receives commands (goto / trajectory / hold / stop)
- `stateReqCh`: request/reply channel for GET /state
- `subscribeCh`: add SSE subscribers
- `unsubCh`: remove SSE subscribers

### Why this approach?
- avoids shared-memory races
- eliminates the need for a single “big mutex”
- keeps state transitions deterministic within the engine tick loop

---

## Tick Loop Behavior

At each tick:

1. Read & process pending commands (non-blocking via select)
2. Compute desired velocity from active command:
   - Go-To: steer toward target
   - Trajectory: steer to current waypoint, advance when reached
3. Apply acceleration limits (smooth velocity)
4. Apply environment effects:
   - wind -> position drift
   - terrain -> altitude clipping and safety warnings
5. Integrate position (Euler step)
6. Publish state snapshot to subscribers

---

## Control Law

A simple control law is used:
- horizontal velocity points toward the target
- climb/descent is limited by max climb rate
- acceleration is bounded for stability

Arrival criteria:
- within horizontal tolerance (~25m)
- within vertical tolerance (~10m)

---

## Environment Effects

Wind:
- constant drift added to position (ground track) each tick

Terrain:
- synthetic height function
- enforces: `z >= groundAltitude + safetyMargin`
- clips altitude and cancels descent if below the safety floor

---

## Observability

- `/state` returns the latest snapshot
- `/stream` provides continuous SSE updates (~tick rate)

