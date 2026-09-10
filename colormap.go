package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	ColormapViridis = iota
	ColormapPlasma
	ColormapCoolWarm
	ColormapTurbo
	ColormapCount
)

var ColormapNames = []string{
	"Viridis",
	"Plasma",
	"Cool-Warm",
	"Turbo",
}

// CategoricalPalette provides distinct, high-contrast colors for categorical labels
var CategoricalPalette = []rl.Color{
	{R: 0, G: 200, B: 255, A: 255},   // Cyan
	{R: 255, G: 130, B: 40, A: 255},  // Vibrant Orange
	{R: 190, G: 80, B: 255, A: 255},  // Neon Purple
	{R: 60, G: 230, B: 110, A: 255},  // Lime Green
	{R: 255, G: 215, B: 30, A: 255},  // Golden Yellow
	{R: 255, G: 60, B: 130, A: 255},  // Hot Pink
	{R: 40, G: 220, B: 190, A: 255},  // Turquoise
	{R: 255, G: 90, B: 90, A: 255},   // Bright Coral
	{R: 120, G: 150, B: 255, A: 255}, // Periwinkle Blue
	{R: 200, G: 255, B: 60, A: 255},  // Chartreuse
}

func GetCategoryColor(index int) rl.Color {
	if index < 0 {
		return rl.Gray
	}
	return CategoricalPalette[index%len(CategoricalPalette)]
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lerpColor(c1, c2 rl.Color, t float32) rl.Color {
	t = clamp01(t)
	inv := 1.0 - t
	return rl.Color{
		R: uint8(float32(c1.R)*inv + float32(c2.R)*t),
		G: uint8(float32(c1.G)*inv + float32(c2.G)*t),
		B: uint8(float32(c1.B)*inv + float32(c2.B)*t),
		A: 255,
	}
}

type gradStop struct {
	pos float32
	col rl.Color
}

func evaluateGradient(stops []gradStop, t float32) rl.Color {
	if len(stops) == 0 {
		return rl.White
	}
	if t <= stops[0].pos {
		return stops[0].col
	}
	if t >= stops[len(stops)-1].pos {
		return stops[len(stops)-1].col
	}

	for i := 0; i < len(stops)-1; i++ {
		if t >= stops[i].pos && t <= stops[i+1].pos {
			segT := (t - stops[i].pos) / (stops[i+1].pos - stops[i].pos)
			return lerpColor(stops[i].col, stops[i+1].col, segT)
		}
	}
	return stops[len(stops)-1].col
}

// GetContinuousColor returns a smooth color for t in [0.0, 1.0]
func GetContinuousColor(t float32, mapType int) rl.Color {
	t = clamp01(t)

	switch mapType {
	case ColormapViridis:
		stops := []gradStop{
			{0.00, rl.Color{R: 68, G: 1, B: 84, A: 255}},
			{0.25, rl.Color{R: 59, G: 82, B: 139, A: 255}},
			{0.50, rl.Color{R: 33, G: 145, B: 140, A: 255}},
			{0.75, rl.Color{R: 94, G: 201, B: 98, A: 255}},
			{1.00, rl.Color{R: 253, G: 231, B: 37, A: 255}},
		}
		return evaluateGradient(stops, t)

	case ColormapPlasma:
		stops := []gradStop{
			{0.00, rl.Color{R: 13, G: 8, B: 135, A: 255}},
			{0.30, rl.Color{R: 126, G: 3, B: 168, A: 255}},
			{0.60, rl.Color{R: 204, G: 71, B: 120, A: 255}},
			{0.85, rl.Color{R: 248, G: 149, B: 64, A: 255}},
			{1.00, rl.Color{R: 240, G: 249, B: 33, A: 255}},
		}
		return evaluateGradient(stops, t)

	case ColormapCoolWarm:
		stops := []gradStop{
			{0.00, rl.Color{R: 59, G: 76, B: 192, A: 255}},
			{0.25, rl.Color{R: 122, G: 165, B: 243, A: 255}},
			{0.50, rl.Color{R: 221, G: 221, B: 221, A: 255}},
			{0.75, rl.Color{R: 244, G: 154, B: 123, A: 255}},
			{1.00, rl.Color{R: 180, G: 4, B: 38, A: 255}},
		}
		return evaluateGradient(stops, t)

	case ColormapTurbo:
		stops := []gradStop{
			{0.00, rl.Color{R: 48, G: 18, B: 59, A: 255}},
			{0.20, rl.Color{R: 70, G: 134, B: 251, A: 255}},
			{0.40, rl.Color{R: 27, G: 229, B: 181, A: 255}},
			{0.60, rl.Color{R: 164, G: 252, B: 60, A: 255}},
			{0.80, rl.Color{R: 251, G: 185, B: 56, A: 255}},
			{1.00, rl.Color{R: 194, G: 36, B: 24, A: 255}},
		}
		return evaluateGradient(stops, t)
	}

	return rl.White
}

// PulseValue calculates animated opacity/scale for 4D time modulation
func PulseValue(t4d float32, curTime float64) float32 {
	freq := 1.5 + float64(t4d)*3.0 // 1.5 Hz to 4.5 Hz oscillation
	sine := math.Sin(curTime * freq * 2.0 * math.Pi)
	return float32(0.65 + 0.35*sine)
}
