package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type OrbitCamera struct {
	Camera          rl.Camera3D
	Target          rl.Vector3
	Distance        float32
	TargetDistance  float32
	Azimuth         float32 // Horizontal rotation in radians
	Elevation       float32 // Vertical rotation in radians
	AutoRotate      bool
	AutoRotateSpeed float32

	prevTouch0 rl.Vector2
	prevTouch1 rl.Vector2
	isPinching bool

	velAzimuth   float32
	velElevation float32
}

func NewOrbitCamera() *OrbitCamera {
	c := &OrbitCamera{
		Target:          rl.Vector3{X: 0, Y: 0, Z: 0},
		Distance:        22.0,
		TargetDistance:  22.0,
		Azimuth:         0.785, // ~45 degrees
		Elevation:       0.45,  // ~25 degrees
		AutoRotate:      false,
		AutoRotateSpeed: 0.35,
	}

	c.Camera = rl.Camera3D{
		Target:     c.Target,
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       45.0,
		Projection: rl.CameraPerspective,
	}

	c.updateCameraPos()
	return c
}

func (c *OrbitCamera) Reset() {
	c.Target = rl.Vector3{X: 0, Y: 0, Z: 0}
	c.TargetDistance = 22.0
	c.Azimuth = 0.785
	c.Elevation = 0.45
	c.velAzimuth = 0
	c.velElevation = 0
}

func (c *OrbitCamera) Zoom(delta float32) {
	c.TargetDistance += delta
	if c.TargetDistance < 6.0 {
		c.TargetDistance = 6.0
	}
	if c.TargetDistance > 75.0 {
		c.TargetDistance = 75.0
	}
}

func (c *OrbitCamera) updateCameraPos() {
	cosElev := float32(math.Cos(float64(c.Elevation)))
	sinElev := float32(math.Sin(float64(c.Elevation)))
	sinAzim := float32(math.Sin(float64(c.Azimuth)))
	cosAzim := float32(math.Cos(float64(c.Azimuth)))

	c.Camera.Position = rl.Vector3{
		X: c.Target.X + c.Distance*cosElev*sinAzim,
		Y: c.Target.Y + c.Distance*sinElev,
		Z: c.Target.Z + c.Distance*cosElev*cosAzim,
	}
	c.Camera.Target = c.Target
}

func (c *OrbitCamera) HandleInput(dt float32, uiCaptured bool) {
	if uiCaptured {
		c.isPinching = false
		return
	}

	touchCount := rl.GetTouchPointCount()

	if touchCount >= 2 {
		// Pinch to Zoom & Pan
		t0 := rl.GetTouchPosition(0)
		t1 := rl.GetTouchPosition(1)

		if c.isPinching {
			prevDist := float32(math.Hypot(float64(c.prevTouch0.X-c.prevTouch1.X), float64(c.prevTouch0.Y-c.prevTouch1.Y)))
			currDist := float32(math.Hypot(float64(t0.X-t1.X), float64(t0.Y-t1.Y)))
			deltaDist := currDist - prevDist

			c.Zoom(-deltaDist * 0.08)

			// Pan target
			midX := (t0.X + t1.X) * 0.5
			midY := (t0.Y + t1.Y) * 0.5
			prevMidX := (c.prevTouch0.X + c.prevTouch1.X) * 0.5
			prevMidY := (c.prevTouch0.Y + c.prevTouch1.Y) * 0.5

			dMidX := (midX - prevMidX) * 0.02
			dMidY := (midY - prevMidY) * 0.02

			sinAzim := float32(math.Sin(float64(c.Azimuth)))
			cosAzim := float32(math.Cos(float64(c.Azimuth)))

			c.Target.X -= cosAzim*dMidX
			c.Target.Z += sinAzim*dMidX
			c.Target.Y += dMidY
		}

		c.prevTouch0 = t0
		c.prevTouch1 = t1
		c.isPinching = true
		c.velAzimuth = 0
		c.velElevation = 0
		return
	} else {
		c.isPinching = false
	}

	// Single finger or Mouse drag
	if rl.IsMouseButtonDown(rl.MouseLeftButton) {
		delta := rl.GetMouseDelta()
		sensitivity := float32(0.005)

		c.velAzimuth = -delta.X * sensitivity
		c.velElevation = delta.Y * sensitivity

		c.Azimuth += c.velAzimuth
		c.Elevation += c.velElevation
	} else {
		// Damping
		c.Azimuth += c.velAzimuth
		c.Elevation += c.velElevation
		c.velAzimuth *= 0.88
		c.velElevation *= 0.88
		if math.Abs(float64(c.velAzimuth)) < 0.0001 {
			c.velAzimuth = 0
		}
		if math.Abs(float64(c.velElevation)) < 0.0001 {
			c.velElevation = 0
		}
	}

	// Mouse wheel zoom
	wheel := rl.GetMouseWheelMove()
	if wheel != 0 {
		c.Zoom(-wheel * 2.0)
	}
}

func (c *OrbitCamera) Update(dt float32) {
	if c.AutoRotate {
		c.Azimuth += c.AutoRotateSpeed * dt
	}

	// Clamp elevation to prevent flip over poles
	maxElev := float32(math.Pi*0.5 - 0.05)
	minElev := float32(-math.Pi*0.5 + 0.05)
	if c.Elevation > maxElev {
		c.Elevation = maxElev
	}
	if c.Elevation < minElev {
		c.Elevation = minElev
	}

	// Smooth zoom lerp
	c.Distance += (c.TargetDistance - c.Distance) * 0.2

	c.updateCameraPos()
}
