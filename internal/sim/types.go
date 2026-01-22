package sim

import (
	"time"
)

type AircraftState struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	Alt float64 `json:"alt"` // meters

	// "Air" velocity (commanded / controlled)
	Vx float64 `json:"vx"`
	Vy float64 `json:"vy"`
	Vz float64 `json:"vz"`

	HeadingDeg float64   `json:"headingDeg"`
	TS         time.Time `json:"ts"`

	ActiveCommand string `json:"activeCommand,omitempty"`
	TargetIndex   int    `json:"targetIndex,omitempty"`
	Warning       string `json:"warning,omitempty"`
}
