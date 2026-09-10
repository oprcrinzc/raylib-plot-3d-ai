package main

import (
	"math"
	"runtime"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	rl.SetMain(main)
}

func main() {
	rl.SetConfigFlags(rl.FlagVsyncHint)
	rl.InitWindow(0, 0, "Iris 3D - Multi-D Visualizer")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	// Load default dataset (Fisher's Iris)
	currentDataset := LoadIrisDataset()

	// Initialize systems
	cam := NewOrbitCamera()
	renderer := NewPlotRenderer()
	ui := NewUIManager()

	if currentDataset != nil {
		ui.setDefaultMappings(currentDataset)
	}

	// Touch & Tap detection state
	var touchStartPos rl.Vector2
	var touchStartTime float64
	var isTouching bool
	var pointPositions []rl.Vector3

	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()
		curTime := rl.GetTime()

		if runtime.GOOS == "android" && rl.IsKeyPressed(rl.KeyBack) {
			if ui.ActiveModal != ModalNone {
				ui.ActiveModal = ModalNone
			} else if renderer.SelectedPoint >= 0 {
				renderer.SelectedPoint = -1
			} else {
				break
			}
		}

		screenW := float32(rl.GetScreenWidth())
		screenH := float32(rl.GetScreenHeight())
		mousePos := rl.GetMousePosition()

		// Touch tap vs drag detection
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			touchStartPos = mousePos
			touchStartTime = curTime
			isTouching = true
		}

		// Check UI capture without double-invoking UI callbacks
		uiCaptured := ui.IsPointerOnUI(screenW, screenH, mousePos, renderer.SelectedPoint >= 0)

		// Check for point selection tap on release
		if isTouching && rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			isTouching = false
			duration := curTime - touchStartTime
			dist := float32(math.Hypot(float64(mousePos.X-touchStartPos.X), float64(mousePos.Y-touchStartPos.Y)))

			if !uiCaptured && duration < 0.35 && dist < 20.0 {
				// Tap on 3D data point!
				hit := renderer.PickPoint(cam.Camera, mousePos, pointPositions)
				if hit >= 0 {
					renderer.SelectedPoint = hit
				} else {
					renderer.SelectedPoint = -1
				}
			}
		}

		// Update Orbit Camera
		cam.HandleInput(dt, uiCaptured)
		cam.Update(dt)

		// Render Frame
		rl.BeginDrawing()
		// Deep sci-fi dark background
		rl.ClearBackground(rl.Color{R: 12, G: 16, B: 24, A: 255})

		// 3D Scene
		rl.BeginMode3D(cam.Camera)
		pointPositions = renderer.Draw3D(
			currentDataset,
			ui.ColX,
			ui.ColY,
			ui.ColZ,
			ui.Col4D,
			cam.Camera,
			curTime,
		)
		rl.EndMode3D()

		// 2D Projected labels in world space
		renderer.Draw2DLabels(
			currentDataset,
			ui.ColX,
			ui.ColY,
			ui.ColZ,
			ui.Col4D,
			cam.Camera,
		)

		// 2D UI Overlay (rendered on top)
		ui.DrawUI(
			currentDataset,
			renderer,
			cam,
			screenW,
			screenH,
			mousePos,
			func(newDs *Dataset) {
				currentDataset = newDs
				renderer.SelectedPoint = -1
			},
		)

		rl.EndDrawing()
	}
}
