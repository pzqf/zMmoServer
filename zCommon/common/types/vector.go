package types

import (
	"math"
)

type Vector3 struct {
	X, Y, Z float32
}

func NewVector3(x, y, z float32) Vector3 {
	return Vector3{X: x, Y: y, Z: z}
}

func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{
		X: v.X + other.X,
		Y: v.Y + other.Y,
		Z: v.Z + other.Z,
	}
}

func (v Vector3) Sub(other Vector3) Vector3 {
	return Vector3{
		X: v.X - other.X,
		Y: v.Y - other.Y,
		Z: v.Z - other.Z,
	}
}

func (v Vector3) Mul(scalar float32) Vector3 {
	return Vector3{
		X: v.X * scalar,
		Y: v.Y * scalar,
		Z: v.Z * scalar,
	}
}

func (v Vector3) Div(scalar float32) Vector3 {
	return Vector3{
		X: v.X / scalar,
		Y: v.Y / scalar,
		Z: v.Z / scalar,
	}
}

func (v Vector3) Length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

func (v Vector3) Normalize() Vector3 {
	length := v.Length()
	if length == 0 {
		return Vector3{0, 0, 0}
	}
	return v.Div(length)
}

func (v Vector3) Distance(other Vector3) float32 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	dz := v.Z - other.Z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}

func (v Vector3) DistanceTo(other Vector3) float32 {
	return v.Distance(other)
}

func (v Vector3) DistanceSquared(other Vector3) float32 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	dz := v.Z - other.Z
	return dx*dx + dy*dy + dz*dz
}

func (v Vector3) Lerp(other Vector3, t float32) Vector3 {
	return Vector3{
		X: v.X + (other.X-v.X)*t,
		Y: v.Y + (other.Y-v.Y)*t,
		Z: v.Z + (other.Z-v.Z)*t,
	}
}

type Quaternion struct {
	X, Y, Z, W float32
}

func NewQuaternion(x, y, z, w float32) Quaternion {
	return Quaternion{X: x, Y: y, Z: z, W: w}
}

func EulerToQuaternion(roll, pitch, yaw float32) Quaternion {
	cr := float32(math.Cos(float64(roll / 2)))
	cp := float32(math.Cos(float64(pitch / 2)))
	cy := float32(math.Cos(float64(yaw / 2)))
	sr := float32(math.Sin(float64(roll / 2)))
	sp := float32(math.Sin(float64(pitch / 2)))
	sy := float32(math.Sin(float64(yaw / 2)))

	return Quaternion{
		X: sr*cp*cy - cr*sp*sy,
		Y: cr*sp*cy + sr*cp*sy,
		Z: cr*cp*sy - sr*sp*cy,
		W: cr*cp*cy + sr*sp*sy,
	}
}

func (q Quaternion) QuaternionToEuler() (roll, pitch, yaw float32) {
	sinr_cosp := 2 * (q.W*q.X + q.Y*q.Z)
	cosr_cosp := 1 - 2*(q.X*q.X+q.Y*q.Y)
	roll = float32(math.Atan2(float64(sinr_cosp), float64(cosr_cosp)))

	sinp := 2 * (q.W*q.Y - q.Z*q.X)
	if math.Abs(float64(sinp)) >= 1 {
		pitch = float32(math.Copysign(math.Pi/2, float64(sinp)))
	} else {
		pitch = float32(math.Asin(float64(sinp)))
	}

	siny_cosp := 2 * (q.W*q.Z + q.X*q.Y)
	cosy_cosp := 1 - 2*(q.Y*q.Y+q.Z*q.Z)
	yaw = float32(math.Atan2(float64(siny_cosp), float64(cosy_cosp)))

	return
}

type Collider struct {
	Type    string
	Radius  float32
	Width   float32
	Height  float32
	Depth   float32
	Offset  Vector3
	Trigger bool
}

func NewSphereCollider(radius float32, offset Vector3, trigger bool) Collider {
	return Collider{
		Type:    "sphere",
		Radius:  radius,
		Offset:  offset,
		Trigger: trigger,
	}
}

func NewBoxCollider(width, height, depth float32, offset Vector3, trigger bool) Collider {
	return Collider{
		Type:    "box",
		Width:   width,
		Height:  height,
		Depth:   depth,
		Offset:  offset,
		Trigger: trigger,
	}
}

type Position struct {
	MapID int32
	Pos   Vector3
}

func NewPosition(mapID int32, pos Vector3) Position {
	return Position{
		MapID: mapID,
		Pos:   pos,
	}
}
