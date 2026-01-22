package sim

import (
	"flight-simulator2/internal/geometry/vector"
	"math"
)

type GeoRef struct {
	OriginLat float64
	OriginLon float64
}

const metersPerDegLat = 111_320.0

func (g GeoRef) metersPerDegLon() float64 {
	return metersPerDegLat * math.Cos(g.OriginLat*math.Pi/180.0)
}

func (g GeoRef) GeoToLocal(lat, lon, alt float64) vector.Vec3 {
	dLat := lat - g.OriginLat
	dLon := lon - g.OriginLon
	return vector.Vec3{
		X: dLon * g.metersPerDegLon(), // east
		Y: dLat * metersPerDegLat,     // north
		Z: alt,
	}
}

func (g GeoRef) LocalToGeo(p vector.Vec3) (lat, lon, alt float64) {
	lat = g.OriginLat + p.Y/metersPerDegLat
	lon = g.OriginLon + p.X/g.metersPerDegLon()
	alt = p.Z
	return
}

func HeadingDegFromVec(v vector.Vec3) float64 {
	// Heading: 0=north, 90=east
	if math.Abs(v.X) < 1e-9 && math.Abs(v.Y) < 1e-9 {
		return 0
	}
	angleRad := math.Atan2(v.X, v.Y)
	deg := angleRad * 180.0 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}
