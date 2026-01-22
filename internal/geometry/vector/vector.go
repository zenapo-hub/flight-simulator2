// Package vector provides 3D vector operations
package vector

// NewVec3 creates a new 3D vector with the given components
func NewVec3(x, y, z float64) Vec3 {
	return Vec3{X: x, Y: y, Z: z}
}

// Vec3 represents a 3D vector in local ENU (East-North-Up) coordinates
// with X=east, Y=north, Z=up (meters)
type Vec3 struct{ X, Y, Z float64 }

// Add returns the sum of two vectors
func (v Vec3) Add(o Vec3) Vec3 { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }

// Sub returns the difference between two vectors
func (v Vec3) Sub(o Vec3) Vec3 { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }

// Mul scales a vector by a scalar
func (v Vec3) Mul(k float64) Vec3 { return Vec3{v.X * k, v.Y * k, v.Z * k} }

// Norm returns the vector's magnitude (Euclidean norm)
func (v Vec3) Norm() float64 { return v.Dot(v) }

// Dot returns the dot product of two vectors
func (v Vec3) Dot(o Vec3) float64 { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }

// Cross returns the cross product of two vectors
func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		X: v.Y*o.Z - v.Z*o.Y,
		Y: v.Z*o.X - v.X*o.Z,
		Z: v.X*o.Y - v.Y*o.X,
	}
}

// Normalize returns a unit vector in the same direction
func (v Vec3) Normalize() Vec3 {
	norm := v.Norm()
	if norm == 0 {
		return Vec3{}
	}
	return v.Mul(1 / norm)
}
