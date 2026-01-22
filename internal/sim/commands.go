package sim

import "time"

type CommandType string

const (
	CmdGoTo       CommandType = "goto"
	CmdTrajectory CommandType = "trajectory"
	CmdHold       CommandType = "hold"
	CmdStop       CommandType = "stop"
)

type Command interface {
	Type() CommandType
	ReceivedAt() time.Time
}

type GoToCommand struct {
	At    time.Time
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Alt   float64 `json:"alt"`
	Speed float64 `json:"speed,omitempty"` // m/s
}

func (c GoToCommand) Type() CommandType     { return CmdGoTo }
func (c GoToCommand) ReceivedAt() time.Time { return c.At }

type Waypoint struct {
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Alt   float64 `json:"alt"`
	Speed float64 `json:"speed,omitempty"` // m/s optional
}

type TrajectoryCommand struct {
	At        time.Time
	Waypoints []Waypoint `json:"waypoints"`
	Loop      bool       `json:"loop,omitempty"`
}

func (c TrajectoryCommand) Type() CommandType     { return CmdTrajectory }
func (c TrajectoryCommand) ReceivedAt() time.Time { return c.At }

type HoldCommand struct{ At time.Time }

func (c HoldCommand) Type() CommandType     { return CmdHold }
func (c HoldCommand) ReceivedAt() time.Time { return c.At }

type StopCommand struct{ At time.Time }

func (c StopCommand) Type() CommandType     { return CmdStop }
func (c StopCommand) ReceivedAt() time.Time { return c.At }
