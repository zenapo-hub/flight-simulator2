# ✈️ Airborne Flight Simulator Backend (Go)

A concurrent backend service that simulates an aircraft flying in real-time and allows clients to command it using:

- **Go-To Point** (lat/lon/alt)
- **Trajectory** (list of waypoints)

Includes:
- REST API for commands and state query
- SSE streaming for live state updates
- Modular environment effects (wind drift + terrain floor)

---

## ✅ Requirements

Choose one of the following:

### Option A: Run locally (Go)
- Go 1.20+ recommended

### Option B: Run with Docker (recommended)
- Docker + Docker Compose (v2)

---

## 🏃 Run the Server

### ✅ Build & run locally (Docker Compose)

From repository root:

```bash
docker compose up --build
```

Server starts at:

- `http://localhost:8080`

Verify it is alive:

```bash
curl -s http://localhost:8080/health
# ok
```

Stop:

```bash
docker compose down
```

---

### Run locally (Go)

From repository root:

```bash
go run ./cmd/server
```

Server starts at:

- `http://localhost:8080`

Verify it is alive:

```bash
curl -s http://localhost:8080/health
# ok
```

---

## 🏗️ Project Structure

```
flight-simulator2/
├── cmd/
│   └── server/              # Main entry point (wiring, engine + http server)
│       └── main.go
├── internal/
│   ├── api/                 # HTTP endpoints + SSE stream
│   │   └── http.go
│   ├── env/                 # Environment effects (Wind, Terrain, Chain)
│   │   ├── env.go
│   │   ├── wind.go
│   │   └── terrain.go
│   ├── geometry/
│   │   └── vector/          # Math primitives (Vec3, helpers)
│   └── sim/                 # Simulation engine + commands + state
│       ├── engine.go
│       ├── geo.go
│       ├── commands.go
│       └── types.go
├── examples/
│   └── environment_demo/    # Standalone demo for wind/terrain effects
├── README.md
└── DESIGN.md
```

Notes:
- The simulator uses a single vector implementation under `internal/geometry/vector` (used by both `sim` and `env`).
- The simulation operates in a local **ENU** (East/North/Up) coordinate system in meters, converted to/from lat/lon around a fixed origin.

---

## 📡 API Endpoints

### Health
**GET** `/health`

```bash
curl -s http://localhost:8080/health
```

---

### Get Current State
**GET** `/state`

```bash
curl -s http://localhost:8080/state | jq
```

Example response:

```json
{
  "lat": 32.14723477905029,
  "lon": 34.88470210908122,
  "alt": 103.59993897051552,
  "vx": -78.49996242993329,
  "vy": -15.419335215860063,
  "vz": 0,
  "headingDeg": 258.88717047809934,
  "ts": "2026-01-22T19:35:11.924518667+02:00",
  "activeCommand": "trajectory"
}
```

Field meanings:
- `lat, lon, alt` – position (degrees, degrees, meters)
- `vx, vy, vz` – **air velocity** in local meters/sec (east/north/up)
- `headingDeg` – heading derived from velocity:
  - 0° = north, 90° = east, 180° = south, 270° = west
- `ts` – timestamp
- `activeCommand` – `"goto" | "trajectory" | "hold"` (field omitted when idle)

---

## 🎮 Commands

### 1) Go-To Point
**POST** `/command/goto`

```bash
curl -s -X POST http://localhost:8080/command/goto \
  -H "Content-Type: application/json" \
  -d '{
    "lat": 32.0900,
    "lon": 34.8000,
    "alt": 200.0,
    "speed": 120.0
  }' | jq
```

Response:
```json
{
  "status": "accepted",
  "type": "goto"
}
```

Notes:
- `speed` is optional (m/s). If omitted, a default speed is used.
- A new command replaces any currently active command.

---

### 2) Trajectory (Waypoints)
**POST** `/command/trajectory`

```bash
curl -s -X POST http://localhost:8080/command/trajectory \
  -H "Content-Type: application/json" \
  -d '{
    "waypoints": [
      {"lat": 32.0, "lon": 34.0, "alt": 100.0},
      {"lat": 32.5, "lon": 34.5, "alt": 150.0}
    ],
    "loop": false
  }' | jq
```

Response:
```json
{
  "count": 2,
  "status": "accepted",
  "type": "trajectory"
}
```

Notes:
- Waypoints are executed in order.
- `loop=true` repeats the trajectory after the last waypoint.
- An optional `speed` can be given per waypoint:
  ```json
  {"lat": 32.0, "lon": 34.0, "alt": 100.0, "speed": 90.0}
  ```

---

### 3) Hold (stop movement and wait)
**POST** `/command/hold`

```bash
curl -s -X POST http://localhost:8080/command/hold | jq
```

---

### 4) Stop (clear command + zero velocity)
**POST** `/command/stop`

```bash
curl -s -X POST http://localhost:8080/command/stop | jq
```

---

## 📺 Live Telemetry Streaming (SSE)

**GET** `/stream`

Streams state updates at the engine tick rate (default ~20Hz):

```bash
curl -N http://localhost:8080/stream
```

You will see events like:

```text
event: state
data: {"lat":...,"lon":...,"alt":...,"vx":...}
```

---

## 🌬️ Environment Effects

Environment effects are modular and applied during each simulation tick.

### Wind
- Implemented as a constant drift applied to position (ground track).
- Does not accumulate into velocity (prevents artificial acceleration).

### Terrain
- Synthetic terrain (sine/cosine) used for demo purposes.
- Enforces a safety floor:
  - `altitude >= terrainAltitude + safetyMargin`
- If the aircraft goes below the floor, altitude is clipped and a warning is emitted.
- Terrain altitude can be queried via `Terrain.GroundAltitude(pos)`.

---

## 📚 Examples

### Environment Demo

A standalone demo to show wind drift + terrain floor behavior:

```bash
cd examples/environment_demo
go run main.go
```

---

## 📌 Repo Entry Point

Server main:
- `cmd/server/main.go`

Core logic:
- `internal/sim` – simulation engine + commands + state
- `internal/api` – HTTP endpoints + SSE stream
- `internal/env` – wind/terrain effects + chaining
- `internal/geometry/vector` – shared Vec3 math primitives
