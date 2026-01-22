package sim

import (
	"context"
	"flight-simulator2/internal/env"
	"flight-simulator2/internal/geometry/vector"
	"math"
	"time"
)

type stateReq struct {
	reply chan AircraftState
}

type subscribeReq struct {
	ch chan AircraftState
}

type Engine struct {
	geo GeoRef

	// Actor channels
	cmdCh       chan Command
	stateReqCh  chan stateReq
	subscribeCh chan subscribeReq
	unsubCh     chan chan AircraftState

	tickHz      float64
	environment env.Environment
}

type Config struct {
	OriginLat float64
	OriginLon float64
	TickHz    float64

	Environment env.Environment
}

func New(cfg Config) *Engine {
	if cfg.TickHz <= 0 {
		cfg.TickHz = 20
	}
	return &Engine{
		geo:         GeoRef{OriginLat: cfg.OriginLat, OriginLon: cfg.OriginLon},
		cmdCh:       make(chan Command, 128),
		stateReqCh:  make(chan stateReq, 32),
		subscribeCh: make(chan subscribeReq, 32),
		unsubCh:     make(chan chan AircraftState, 32),
		tickHz:      cfg.TickHz,
		environment: cfg.Environment,
	}
}

func (e *Engine) Submit(cmd Command) {
	select {
	case e.cmdCh <- cmd:
	default:
		// drop if overloaded (or you can block / log)
	}
}

func (e *Engine) GetState(ctx context.Context) (AircraftState, error) {
	req := stateReq{reply: make(chan AircraftState, 1)}
	select {
	case e.stateReqCh <- req:
	case <-ctx.Done():
		return AircraftState{}, ctx.Err()
	}

	select {
	case st := <-req.reply:
		return st, nil
	case <-ctx.Done():
		return AircraftState{}, ctx.Err()
	}
}

func (e *Engine) Subscribe(ctx context.Context) (<-chan AircraftState, func()) {
	ch := make(chan AircraftState, 32)

	select {
	case e.subscribeCh <- subscribeReq{ch: ch}:
	case <-ctx.Done():
		close(ch)
		return ch, func() {}
	}

	unsub := func() {
		select {
		case e.unsubCh <- ch:
		default:
		}
	}
	return ch, unsub
}

func (e *Engine) Run(ctx context.Context) error {
	// Actor-owned state
	now := time.Now()

	pos := e.geo.GeoToLocal(e.geo.OriginLat, e.geo.OriginLon, 1000) // start at 1000m
	vel := vector.Vec3{}                                            // "air" velocity

	var active Command
	var traj []Waypoint
	trajIdx := 0
	trajLoop := false

	subs := map[chan AircraftState]struct{}{}

	// Simple tuning
	posTolM := 25.0
	altTolM := 10.0
	defaultSpeed := 80.0
	maxClimbRate := 8.0
	maxHorizAccel := 12.0
	maxVertAccel := 5.0

	buildSnapshot := func(ts time.Time, warning string) AircraftState {
		lat, lon, alt := e.geo.LocalToGeo(pos)
		st := AircraftState{
			Lat: lat, Lon: lon, Alt: alt,
			Vx: vel.X, Vy: vel.Y, Vz: vel.Z,
			HeadingDeg:  HeadingDegFromVec(vel),
			TS:          ts,
			Warning:     warning,
			TargetIndex: trajIdx,
		}
		if active != nil {
			st.ActiveCommand = string(active.Type())
		}
		return st
	}

	publish := func(st AircraftState) {
		for ch := range subs {
			select {
			case ch <- st:
			default:
				// slow subscriber -> drop frame
			}
		}
	}

	setActive := func(cmd Command) {
		active = cmd
		traj = nil
		trajIdx = 0
		trajLoop = false

		if tc, ok := cmd.(TrajectoryCommand); ok {
			traj = tc.Waypoints
			trajIdx = 0
			trajLoop = tc.Loop
		}
	}

	dist2D := func(a vector.Vec3) float64 {
		return math.Sqrt(a.X*a.X + a.Y*a.Y)
	}

	normalize2D := func(v vector.Vec3) vector.Vec3 {
		n := dist2D(v)
		if n < 1e-9 {
			return vector.Vec3{}
		}
		return vector.Vec3{X: v.X / n, Y: v.Y / n, Z: 0}
	}

	computeDesiredVel := func(target vector.Vec3, speed float64) vector.Vec3 {
		delta := vector.Vec3{X: target.X - pos.X, Y: target.Y - pos.Y, Z: target.Z - pos.Z}
		horiz := vector.Vec3{X: delta.X, Y: delta.Y, Z: 0}
		hDist := dist2D(horiz)

		desired := vector.Vec3{}

		if hDist > posTolM {
			dir := normalize2D(horiz)
			desired.X = dir.X * speed
			desired.Y = dir.Y * speed
		}

		if delta.Z > altTolM {
			desired.Z = maxClimbRate
		} else if delta.Z < -altTolM {
			desired.Z = -maxClimbRate
		} else {
			desired.Z = 0
		}

		return desired
	}

	approach := func(cur, des float64, amax float64, dt float64) float64 {
		diff := des - cur
		maxStep := amax * dt
		if diff > maxStep {
			return cur + maxStep
		}
		if diff < -maxStep {
			return cur - maxStep
		}
		return des
	}

	approachVel := func(cur, des vector.Vec3, dt float64) vector.Vec3 {
		return vector.Vec3{
			X: approach(cur.X, des.X, maxHorizAccel, dt),
			Y: approach(cur.Y, des.Y, maxHorizAccel, dt),
			Z: approach(cur.Z, des.Z, maxVertAccel, dt),
		}
	}

	tick := time.NewTicker(time.Duration(float64(time.Second) / e.tickHz))
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			for ch := range subs {
				close(ch)
			}
			return nil

		case req := <-e.subscribeCh:
			subs[req.ch] = struct{}{}
			req.ch <- buildSnapshot(now, "")

		case ch := <-e.unsubCh:
			if _, ok := subs[ch]; ok {
				delete(subs, ch)
				close(ch)
			}

		case req := <-e.stateReqCh:
			req.reply <- buildSnapshot(now, "")

		case cmd := <-e.cmdCh:
			switch cmd.Type() {
			case CmdStop:
				active = nil
				traj = nil
				trajIdx = 0
				vel = vector.Vec3{}

			case CmdHold:
				active = cmd
				traj = nil
				trajIdx = 0
				vel = vector.Vec3{}

			case CmdGoTo, CmdTrajectory:
				setActive(cmd)
			}

		case t := <-tick.C:
			dt := t.Sub(now).Seconds()
			if dt <= 0 {
				dt = 1.0 / e.tickHz
			}
			now = t

			warning := ""

			// compute desired velocity from active command
			desired := vector.Vec3{}
			if active != nil {
				switch c := active.(type) {
				case GoToCommand:
					target := e.geo.GeoToLocal(c.Lat, c.Lon, c.Alt)
					speed := c.Speed
					if speed <= 0 {
						speed = defaultSpeed
					}

					desired = computeDesiredVel(target, speed)

					// arrival check
					d := vector.Vec3{X: target.X - pos.X, Y: target.Y - pos.Y, Z: target.Z - pos.Z}
					if dist2D(vector.Vec3{X: d.X, Y: d.Y}) <= posTolM && math.Abs(d.Z) <= altTolM {
						active = nil
						desired = vector.Vec3{}
					}

				case TrajectoryCommand:
					if len(traj) == 0 || trajIdx < 0 || trajIdx >= len(traj) {
						active = nil
						desired = vector.Vec3{}
						break
					}

					wp := traj[trajIdx]
					target := e.geo.GeoToLocal(wp.Lat, wp.Lon, wp.Alt)
					speed := wp.Speed
					if speed <= 0 {
						speed = defaultSpeed
					}

					desired = computeDesiredVel(target, speed)

					d := vector.Vec3{X: target.X - pos.X, Y: target.Y - pos.Y, Z: target.Z - pos.Z}
					if dist2D(vector.Vec3{X: d.X, Y: d.Y}) <= posTolM && math.Abs(d.Z) <= altTolM {
						trajIdx++
						if trajIdx >= len(traj) {
							if trajLoop {
								trajIdx = 0
							} else {
								active = nil
								desired = vector.Vec3{}
							}
						}
					}

				case HoldCommand:
					desired = vector.Vec3{}
				}
			}

			// smooth toward desired velocity (air velocity)
			vel = approachVel(vel, desired, dt)

			// apply environment effects (wind affects position, terrain clips altitude, etc.)
			if e.environment != nil {
				p2, v2, warn := e.environment.Apply(dt, pos, vel)
				pos, vel = p2, v2
				warning = warn
			}

			// integrate position by air velocity (wind drift already applied in env)
			pos.X += vel.X * dt
			pos.Y += vel.Y * dt
			pos.Z += vel.Z * dt

			st := buildSnapshot(now, warning)
			publish(st)
		}
	}
}
