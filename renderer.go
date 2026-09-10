package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	Mode4DColorContinuous = iota
	Mode4DCategorical
	Mode4DSize
	Mode4DDualColorSize
	Mode4DPulse
	Mode4DCount
)

var Mode4DNames = []string{
	"4D: Continuous Color",
	"4D: Class Category",
	"4D: Sphere Size",
	"4D: Color + Size",
	"4D: Pulse Animation",
}

type PlotRenderer struct {
	BoxMin          rl.Vector3
	BoxMax          rl.Vector3
	ShowGrid        bool
	ShowBox         bool
	ShowDropLines   bool
	BasePointSize   float32 // 0.15 to 0.50
	CurrentMode4D   int
	CurrentColormap int
	SelectedPoint   int
}

func NewPlotRenderer() *PlotRenderer {
	return &PlotRenderer{
		BoxMin:          rl.Vector3{X: -6.0, Y: -4.0, Z: -6.0},
		BoxMax:          rl.Vector3{X: 6.0, Y: 4.0, Z: 6.0},
		ShowGrid:        true,
		ShowBox:         true,
		ShowDropLines:   true,
		BasePointSize:   0.28,
		CurrentMode4D:   Mode4DCategorical,
		CurrentColormap: ColormapViridis,
		SelectedPoint:   -1,
	}
}

func (r *PlotRenderer) Draw3D(ds *Dataset, colX, colY, colZ, col4D int, cam rl.Camera3D, curTime float64) []rl.Vector3 {
	if ds == nil || len(ds.Points) == 0 {
		return nil
	}

	pointPositions := make([]rl.Vector3, len(ds.Points))

	// 1. Draw Bounding Box & Floor Grid
	if r.ShowGrid {
		// Floor grid on Y = BoxMin.Y
		yFloor := r.BoxMin.Y
		gridCol := rl.Color{R: 50, G: 65, B: 85, A: 255}
		gridStep := float32(1.5)

		for x := r.BoxMin.X; x <= r.BoxMax.X+0.01; x += gridStep {
			rl.DrawLine3D(
				rl.Vector3{X: x, Y: yFloor, Z: r.BoxMin.Z},
				rl.Vector3{X: x, Y: yFloor, Z: r.BoxMax.Z},
				gridCol,
			)
		}
		for z := r.BoxMin.Z; z <= r.BoxMax.Z+0.01; z += gridStep {
			rl.DrawLine3D(
				rl.Vector3{X: r.BoxMin.X, Y: yFloor, Z: z},
				rl.Vector3{X: r.BoxMax.X, Y: yFloor, Z: z},
				gridCol,
			)
		}
	}

	if r.ShowBox {
		boxCol := rl.Color{R: 70, G: 85, B: 110, A: 160}
		// Corner pillars
		rl.DrawLine3D(rl.Vector3{X: r.BoxMin.X, Y: r.BoxMin.Y, Z: r.BoxMin.Z}, rl.Vector3{X: r.BoxMin.X, Y: r.BoxMax.Y, Z: r.BoxMin.Z}, boxCol)
		rl.DrawLine3D(rl.Vector3{X: r.BoxMax.X, Y: r.BoxMin.Y, Z: r.BoxMin.Z}, rl.Vector3{X: r.BoxMax.X, Y: r.BoxMax.Y, Z: r.BoxMin.Z}, boxCol)
		rl.DrawLine3D(rl.Vector3{X: r.BoxMin.X, Y: r.BoxMin.Y, Z: r.BoxMax.Z}, rl.Vector3{X: r.BoxMin.X, Y: r.BoxMax.Y, Z: r.BoxMax.Z}, boxCol)
		rl.DrawLine3D(rl.Vector3{X: r.BoxMax.X, Y: r.BoxMin.Y, Z: r.BoxMax.Z}, rl.Vector3{X: r.BoxMax.X, Y: r.BoxMax.Y, Z: r.BoxMax.Z}, boxCol)

		// Top frame
		rl.DrawLine3D(rl.Vector3{X: r.BoxMin.X, Y: r.BoxMax.Y, Z: r.BoxMin.Z}, rl.Vector3{X: r.BoxMax.X, Y: r.BoxMax.Y, Z: r.BoxMin.Z}, boxCol)
		rl.DrawLine3D(rl.Vector3{X: r.BoxMax.X, Y: r.BoxMax.Y, Z: r.BoxMin.Z}, rl.Vector3{X: r.BoxMax.X, Y: r.BoxMax.Y, Z: r.BoxMax.Z}, boxCol)
		rl.DrawLine3D(rl.Vector3{X: r.BoxMax.X, Y: r.BoxMax.Y, Z: r.BoxMax.Z}, rl.Vector3{X: r.BoxMin.X, Y: r.BoxMax.Y, Z: r.BoxMax.Z}, boxCol)
		rl.DrawLine3D(rl.Vector3{X: r.BoxMin.X, Y: r.BoxMax.Y, Z: r.BoxMax.Z}, rl.Vector3{X: r.BoxMin.X, Y: r.BoxMax.Y, Z: r.BoxMin.Z}, boxCol)
	}

	// 2. Main Axes with distinct colors
	origin := rl.Vector3{X: r.BoxMin.X, Y: r.BoxMin.Y, Z: r.BoxMin.Z}
	xEnd := rl.Vector3{X: r.BoxMax.X + 0.6, Y: r.BoxMin.Y, Z: r.BoxMin.Z}
	yEnd := rl.Vector3{X: r.BoxMin.X, Y: r.BoxMax.Y + 0.6, Z: r.BoxMin.Z}
	zEnd := rl.Vector3{X: r.BoxMin.X, Y: r.BoxMin.Y, Z: r.BoxMax.Z + 0.6}

	// X Axis (Red)
	rl.DrawLine3D(origin, xEnd, rl.Color{R: 255, G: 60, B: 60, A: 255})
	rl.DrawSphere(xEnd, 0.2, rl.Color{R: 255, G: 60, B: 60, A: 255})

	// Y Axis (Green)
	rl.DrawLine3D(origin, yEnd, rl.Color{R: 60, G: 240, B: 80, A: 255})
	rl.DrawSphere(yEnd, 0.2, rl.Color{R: 60, G: 240, B: 80, A: 255})

	// Z Axis (Blue)
	rl.DrawLine3D(origin, zEnd, rl.Color{R: 60, G: 160, B: 255, A: 255})
	rl.DrawSphere(zEnd, 0.2, rl.Color{R: 60, G: 160, B: 255, A: 255})

	// 3. Render Data Points
	for i := range ds.Points {
		pt := &ds.Points[i]

		posX := ds.GetNormalizedValue(i, colX, r.BoxMin.X, r.BoxMax.X)
		posY := ds.GetNormalizedValue(i, colY, r.BoxMin.Y, r.BoxMax.Y)
		posZ := ds.GetNormalizedValue(i, colZ, r.BoxMin.Z, r.BoxMax.Z)
		pos := rl.Vector3{X: posX, Y: posY, Z: posZ}
		pointPositions[i] = pos

		t4d := ds.Get01NormalizedValue(i, col4D)

		// Determine Color & Radius based on 4D mode
		radius := r.BasePointSize
		var pointCol rl.Color

		switch r.CurrentMode4D {
		case Mode4DColorContinuous:
			pointCol = GetContinuousColor(t4d, r.CurrentColormap)
		case Mode4DCategorical:
			if len(ds.Categories) > 0 {
				pointCol = GetCategoryColor(pt.CategoryIdx)
			} else {
				pointCol = GetContinuousColor(t4d, r.CurrentColormap)
			}
		case Mode4DSize:
			pointCol = GetContinuousColor(t4d, r.CurrentColormap)
			radius = r.BasePointSize * (0.4 + 1.2*t4d)
		case Mode4DDualColorSize:
			pointCol = GetContinuousColor(t4d, r.CurrentColormap)
			radius = r.BasePointSize * (0.4 + 1.2*t4d)
		case Mode4DPulse:
			pointCol = GetContinuousColor(t4d, r.CurrentColormap)
			pulse := PulseValue(t4d, curTime)
			radius = r.BasePointSize * pulse
		}

		// Drop line to floor
		if r.ShowDropLines {
			floorPos := rl.Vector3{X: posX, Y: r.BoxMin.Y, Z: posZ}
			dropCol := rl.Color{R: pointCol.R, G: pointCol.G, B: pointCol.B, A: 50}
			rl.DrawLine3D(pos, floorPos, dropCol)
			rl.DrawCircle3D(floorPos, radius*0.5, rl.Vector3{X: 1, Y: 0, Z: 0}, 90.0, dropCol)
		}

		// Draw Sphere
		rl.DrawSphere(pos, radius, pointCol)
		darkRing := rl.Color{R: uint8(float32(pointCol.R) * 0.4), G: uint8(float32(pointCol.G) * 0.4), B: uint8(float32(pointCol.B) * 0.4), A: 220}
		rl.DrawSphereWires(pos, radius*1.02, 5, 5, darkRing)

		// Selected point highlight
		if i == r.SelectedPoint {
			glowPulse := float32(1.8 + 0.3*PulseValue(0.5, curTime*2.0))
			rl.DrawSphereWires(pos, radius*glowPulse, 8, 8, rl.Yellow)
			rl.DrawLine3D(rl.Vector3{X: pos.X - 1.0, Y: pos.Y, Z: pos.Z}, rl.Vector3{X: pos.X + 1.0, Y: pos.Y, Z: pos.Z}, rl.Gold)
			rl.DrawLine3D(rl.Vector3{X: pos.X, Y: pos.Y - 1.0, Z: pos.Z}, rl.Vector3{X: pos.X, Y: pos.Y + 1.0, Z: pos.Z}, rl.Gold)
			rl.DrawLine3D(rl.Vector3{X: pos.X, Y: pos.Y, Z: pos.Z - 1.0}, rl.Vector3{X: pos.X, Y: pos.Y, Z: pos.Z + 1.0}, rl.Gold)
		}
	}

	return pointPositions
}

// Draw2DLabels renders projected 3D axis titles and ticks onto the 2D viewport
func (r *PlotRenderer) Draw2DLabels(ds *Dataset, colX, colY, colZ, col4D int, cam rl.Camera3D) {
	if ds == nil || len(ds.NumericIndices) == 0 {
		return
	}

	xName := ds.NumericNames[colX]
	yName := ds.NumericNames[colY]
	zName := ds.NumericNames[colZ]

	xMinStr := fmt.Sprintf("%.2f", ds.MinVals[colX])
	xMaxStr := fmt.Sprintf("%.2f", ds.MaxVals[colX])
	yMinStr := fmt.Sprintf("%.2f", ds.MinVals[colY])
	yMaxStr := fmt.Sprintf("%.2f", ds.MaxVals[colY])
	zMinStr := fmt.Sprintf("%.2f", ds.MinVals[colZ])
	zMaxStr := fmt.Sprintf("%.2f", ds.MaxVals[colZ])

	// Project positions
	xTipScreen := rl.GetWorldToScreen(rl.Vector3{X: r.BoxMax.X + 0.9, Y: r.BoxMin.Y, Z: r.BoxMin.Z}, cam)
	yTipScreen := rl.GetWorldToScreen(rl.Vector3{X: r.BoxMin.X, Y: r.BoxMax.Y + 0.9, Z: r.BoxMin.Z}, cam)
	zTipScreen := rl.GetWorldToScreen(rl.Vector3{X: r.BoxMin.X, Y: r.BoxMin.Y, Z: r.BoxMax.Z + 0.9}, cam)

	originScreen := rl.GetWorldToScreen(rl.Vector3{X: r.BoxMin.X, Y: r.BoxMin.Y, Z: r.BoxMin.Z}, cam)

	// Draw Axis Names & Ranges
	if xTipScreen.X > 0 && xTipScreen.Y > 0 {
		text := fmt.Sprintf("X: %s [%s..%s]", xName, xMinStr, xMaxStr)
		rl.DrawRectangle(int32(xTipScreen.X)-4, int32(xTipScreen.Y)-2, int32(len(text)*7)+8, 18, rl.Color{R: 20, G: 20, B: 25, A: 200})
		rl.DrawText(text, int32(xTipScreen.X), int32(xTipScreen.Y), 12, rl.Color{R: 255, G: 120, B: 120, A: 255})
	}

	if yTipScreen.X > 0 && yTipScreen.Y > 0 {
		text := fmt.Sprintf("Y: %s [%s..%s]", yName, yMinStr, yMaxStr)
		rl.DrawRectangle(int32(yTipScreen.X)-4, int32(yTipScreen.Y)-2, int32(len(text)*7)+8, 18, rl.Color{R: 20, G: 20, B: 25, A: 200})
		rl.DrawText(text, int32(yTipScreen.X), int32(yTipScreen.Y), 12, rl.Color{R: 120, G: 255, B: 140, A: 255})
	}

	if zTipScreen.X > 0 && zTipScreen.Y > 0 {
		text := fmt.Sprintf("Z: %s [%s..%s]", zName, zMinStr, zMaxStr)
		rl.DrawRectangle(int32(zTipScreen.X)-4, int32(zTipScreen.Y)-2, int32(len(text)*7)+8, 18, rl.Color{R: 20, G: 20, B: 25, A: 200})
		rl.DrawText(text, int32(zTipScreen.X), int32(zTipScreen.Y), 12, rl.Color{R: 120, G: 190, B: 255, A: 255})
	}

	if originScreen.X > 0 && originScreen.Y > 0 {
		rl.DrawText("(Min,Min,Min)", int32(originScreen.X)+5, int32(originScreen.Y)+5, 10, rl.LightGray)
	}
}

// PickPoint performs 3D ray collision against rendered points
func (r *PlotRenderer) PickPoint(cam rl.Camera3D, tapPos rl.Vector2, positions []rl.Vector3) int {
	if len(positions) == 0 {
		return -1
	}

	ray := rl.GetScreenToWorldRay(tapPos, cam)
	nearestDist := float32(1e9)
	hitIndex := -1
	hitRadius := r.BasePointSize * 2.2 // Generous hit area for touchscreen tapping!

	for i, pos := range positions {
		collision := rl.GetRayCollisionSphere(ray, pos, hitRadius)
		if collision.Hit && collision.Distance < nearestDist {
			nearestDist = collision.Distance
			hitIndex = i
		}
	}

	return hitIndex
}
