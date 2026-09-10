package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type ModalType int

const (
	ModalNone ModalType = iota
	ModalDatasets
	ModalAxes
	ModalView
	ModalImportCSV
)

type UIManager struct {
	ActiveModal     ModalType
	ColX            int
	ColY            int
	ColZ            int
	Col4D           int
	SelectedAxisTab int // 0=X, 1=Y, 2=Z, 3=4D

	// File browser state
	FoundFiles     []string
	ImportStatus   string
	ImportStatusOk bool
}

func NewUIManager() *UIManager {
	return &UIManager{
		ActiveModal:     ModalNone,
		ColX:            0,
		ColY:            1,
		ColZ:            2,
		Col4D:           3,
		SelectedAxisTab: 0,
		ImportStatus:    "",
	}
}

// DrawTouchButton renders a responsive touch button with hover/press feedback
func DrawTouchButton(rect rl.Rectangle, label string, fontSize int32, bgCol, textCol rl.Color, mousePos rl.Vector2) bool {
	hovered := rl.CheckCollisionPointRec(mousePos, rect)
	clicked := hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton)

	drawBg := bgCol
	if hovered {
		drawBg = rl.Color{
			R: uint8(mathMin(int(bgCol.R)+35, 255)),
			G: uint8(mathMin(int(bgCol.G)+35, 255)),
			B: uint8(mathMin(int(bgCol.B)+35, 255)),
			A: bgCol.A,
		}
	}

	rl.DrawRectangleRounded(rect, 0.25, 6, drawBg)
	borderCol := rl.Color{R: 120, G: 140, B: 180, A: 160}
	if hovered {
		borderCol = rl.Gold
	}
	rl.DrawRectangleRoundedLines(rect, 0.25, 6, borderCol)

	textW := rl.MeasureText(label, fontSize)
	textX := int32(rect.X + (rect.Width-float32(textW))*0.5)
	textY := int32(rect.Y + (rect.Height-float32(fontSize))*0.5)
	rl.DrawText(label, textX, textY, fontSize, textCol)

	return clicked
}

func mathMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (ui *UIManager) DrawUI(
	ds *Dataset,
	renderer *PlotRenderer,
	cam *OrbitCamera,
	screenW, screenH float32,
	mousePos rl.Vector2,
	onDatasetChange func(newDs *Dataset),
) bool {
	uiCaptured := false

	// Top Bar
	topBarH := float32(48)
	rl.DrawRectangle(0, 0, int32(screenW), int32(topBarH), rl.Color{R: 15, G: 20, B: 30, A: 240})
	rl.DrawLine(0, int32(topBarH), int32(screenW), int32(topBarH), rl.Color{R: 45, G: 60, B: 90, A: 255})

	// App Title
	titleText := "IRIS 3D • 4D VISUALIZER"
	rl.DrawText(titleText, 16, 14, 20, rl.RayWhite)

	// Dataset Name badge
	dsName := "No Dataset"
	ptCount := 0
	if ds != nil {
		dsName = ds.Name
		ptCount = len(ds.Points)
	}
	dsBadge := fmt.Sprintf("[%s | %d Samples]", dsName, ptCount)
	rl.DrawText(dsBadge, 290, 16, 16, rl.Color{R: 0, G: 215, B: 255, A: 255})

	// Mapping summary
	if ds != nil && len(ds.NumericNames) > 0 {
		mappingInfo := fmt.Sprintf("X:%s  Y:%s  Z:%s  4D:%s",
			truncStr(ds.NumericNames[ui.ColX], 10),
			truncStr(ds.NumericNames[ui.ColY], 10),
			truncStr(ds.NumericNames[ui.ColZ], 10),
			truncStr(ds.NumericNames[ui.Col4D], 10),
		)
		infoW := rl.MeasureText(mappingInfo, 14)
		rl.DrawText(mappingInfo, int32(screenW)-infoW-16, 17, 14, rl.LightGray)
	}

	// Bottom Floating Control Bar
	btnW := float32(110)
	btnH := float32(42)
	gap := float32(10)
	bottomY := screenH - btnH - 14

	buttons := []struct {
		label string
		col   rl.Color
		fn    func()
	}{
		{
			label: "Datasets",
			col:   rl.Color{R: 35, G: 70, B: 120, A: 240},
			fn: func() {
				if ui.ActiveModal == ModalDatasets {
					ui.ActiveModal = ModalNone
				} else {
					ui.ActiveModal = ModalDatasets
				}
			},
		},
		{
			label: "Axes & 4D",
			col:   rl.Color{R: 35, G: 90, B: 90, A: 240},
			fn: func() {
				if ui.ActiveModal == ModalAxes {
					ui.ActiveModal = ModalNone
				} else {
					ui.ActiveModal = ModalAxes
				}
			},
		},
		{
			label: "View Style",
			col:   rl.Color{R: 80, G: 50, B: 110, A: 240},
			fn: func() {
				if ui.ActiveModal == ModalView {
					ui.ActiveModal = ModalNone
				} else {
					ui.ActiveModal = ModalView
				}
			},
		},
		{
			label: ternary(cam.AutoRotate, "Turntable: ON", "Turntable: OFF"),
			col:   ternaryCol(cam.AutoRotate, rl.Color{R: 30, G: 120, B: 70, A: 240}, rl.Color{R: 50, G: 55, B: 65, A: 240}),
			fn: func() {
				cam.AutoRotate = !cam.AutoRotate
			},
		},
		{
			label: "Zoom +",
			col:   rl.Color{R: 45, G: 50, B: 65, A: 240},
			fn: func() {
				cam.Zoom(-3.5)
			},
		},
		{
			label: "Zoom -",
			col:   rl.Color{R: 45, G: 50, B: 65, A: 240},
			fn: func() {
				cam.Zoom(3.5)
			},
		},
		{
			label: "Reset Cam",
			col:   rl.Color{R: 50, G: 55, B: 70, A: 240},
			fn: func() {
				cam.Reset()
			},
		},
	}

	// Center buttons horizontally
	totalButtonsW := float32(len(buttons))*btnW + float32(len(buttons)-1)*gap
	startX := (screenW - totalButtonsW) * 0.5
	if startX < 10 {
		startX = 10
		btnW = (screenW - 20 - float32(len(buttons)-1)*gap) / float32(len(buttons))
	}

	for i, b := range buttons {
		btnRect := rl.Rectangle{
			X:      startX + float32(i)*(btnW+gap),
			Y:      bottomY,
			Width:  btnW,
			Height: btnH,
		}
		if rl.CheckCollisionPointRec(mousePos, btnRect) {
			uiCaptured = true
		}
		if DrawTouchButton(btnRect, b.label, 14, b.col, rl.White, mousePos) {
			b.fn()
		}
	}

	// Legend / Color Scale HUD (Top Left, below title)
	ui.drawLegendHUD(ds, renderer, 16, topBarH+14)

	// Point Inspection Card (if a point is selected)
	if renderer.SelectedPoint >= 0 && ds != nil && renderer.SelectedPoint < len(ds.Points) {
		if ui.drawInspectionCard(ds, renderer.SelectedPoint, screenW-310, topBarH+14, 294, mousePos) {
			renderer.SelectedPoint = -1
		}
		cardRect := rl.Rectangle{X: screenW - 310, Y: topBarH + 14, Width: 294, Height: 240}
		if rl.CheckCollisionPointRec(mousePos, cardRect) {
			uiCaptured = true
		}
	}

	// Active Modal Window
	if ui.ActiveModal != ModalNone {
		uiCaptured = true
		modalW := float32(560)
		if modalW > screenW-40 {
			modalW = screenW - 40
		}
		modalH := float32(440)
		if modalH > screenH-100 {
			modalH = screenH - 100
		}
		modalX := (screenW - modalW) * 0.5
		modalY := (screenH - modalH) * 0.5

		// Backdrop dim
		rl.DrawRectangle(0, 0, int32(screenW), int32(screenH), rl.Color{R: 0, G: 0, B: 0, A: 160})

		// Modal Box
		modalRect := rl.Rectangle{X: modalX, Y: modalY, Width: modalW, Height: modalH}
		rl.DrawRectangleRounded(modalRect, 0.04, 8, rl.Color{R: 22, G: 28, B: 40, A: 255})
		rl.DrawRectangleRoundedLines(modalRect, 0.04, 8, rl.Color{R: 80, G: 110, B: 160, A: 255})

		// Close button [X]
		closeRect := rl.Rectangle{X: modalX + modalW - 44, Y: modalY + 10, Width: 34, Height: 34}
		if DrawTouchButton(closeRect, "X", 18, rl.Color{R: 180, G: 40, B: 50, A: 255}, rl.White, mousePos) {
			ui.ActiveModal = ModalNone
		}

		switch ui.ActiveModal {
		case ModalDatasets:
			ui.drawDatasetsModal(modalX, modalY, modalW, modalH, mousePos, onDatasetChange)
		case ModalAxes:
			ui.drawAxesModal(ds, renderer, modalX, modalY, modalW, modalH, mousePos)
		case ModalView:
			ui.drawViewModal(renderer, modalX, modalY, modalW, modalH, mousePos)
		case ModalImportCSV:
			ui.drawImportCSVModal(modalX, modalY, modalW, modalH, mousePos, onDatasetChange)
		}
	}

	return uiCaptured
}

func (ui *UIManager) drawLegendHUD(ds *Dataset, renderer *PlotRenderer, x, y float32) {
	if ds == nil {
		return
	}

	hudW := float32(230)
	hudH := float32(115)

	if renderer.CurrentMode4D == Mode4DCategorical && len(ds.Categories) > 0 {
		hudH = float32(36 + len(ds.Categories)*22)
	}

	rl.DrawRectangleRounded(rl.Rectangle{X: x, Y: y, Width: hudW, Height: hudH}, 0.12, 6, rl.Color{R: 15, G: 22, B: 35, A: 210})
	rl.DrawRectangleRoundedLines(rl.Rectangle{X: x, Y: y, Width: hudW, Height: hudH}, 0.12, 6, rl.Color{R: 60, G: 80, B: 115, A: 180})

	if renderer.CurrentMode4D == Mode4DCategorical && len(ds.Categories) > 0 {
		catColName := "Species / Class"
		if ds.CategoryColIndex >= 0 && ds.CategoryColIndex < len(ds.AllHeaders) {
			catColName = ds.AllHeaders[ds.CategoryColIndex]
		}
		rl.DrawText("Class: "+catColName, int32(x)+12, int32(y)+10, 13, rl.Gold)

		for i, cat := range ds.Categories {
			itemY := int32(y) + 32 + int32(i*22)
			col := GetCategoryColor(i)
			rl.DrawCircle(int32(x)+20, itemY+7, 6, col)
			rl.DrawCircleLines(int32(x)+20, itemY+7, 6, rl.White)
			rl.DrawText(truncStr(cat, 18), int32(x)+34, itemY, 13, rl.RayWhite)
		}
	} else {
		// Continuous 4D Colormap bar
		dim4Name := "4th Dimension"
		if ui.Col4D >= 0 && ui.Col4D < len(ds.NumericNames) {
			dim4Name = ds.NumericNames[ui.Col4D]
		}
		rl.DrawText("4D: "+dim4Name, int32(x)+12, int32(y)+10, 13, rl.Gold)
		rl.DrawText(Mode4DNames[renderer.CurrentMode4D], int32(x)+12, int32(y)+28, 11, rl.SkyBlue)

		// Gradient Bar
		barX := int32(x) + 12
		barY := int32(y) + 50
		barW := int32(206)
		barH := int32(14)

		for bx := int32(0); bx < barW; bx++ {
			t := float32(bx) / float32(barW)
			c := GetContinuousColor(t, renderer.CurrentColormap)
			rl.DrawLine(barX+bx, barY, barX+bx, barY+barH, c)
		}
		rl.DrawRectangleLines(barX-1, barY-1, barW+2, barH+2, rl.Color{R: 150, G: 170, B: 200, A: 255})

		// Min / Max labels
		minVal := ds.MinVals[ui.Col4D]
		maxVal := ds.MaxVals[ui.Col4D]
		rl.DrawText(fmt.Sprintf("%.2f", minVal), barX, barY+18, 11, rl.LightGray)
		maxStr := fmt.Sprintf("%.2f", maxVal)
		maxW := rl.MeasureText(maxStr, 11)
		rl.DrawText(maxStr, barX+barW-maxW, barY+18, 11, rl.LightGray)
		rl.DrawText(ColormapNames[renderer.CurrentColormap], barX+(barW/2)-20, barY+18, 11, rl.Gold)
	}
}

func (ui *UIManager) drawInspectionCard(ds *Dataset, ptIdx int, x, y, w float32, mousePos rl.Vector2) bool {
	pt := &ds.Points[ptIdx]
	h := float32(40 + len(ds.NumericIndices)*24)
	if h > 300 {
		h = 300
	}

	cardRect := rl.Rectangle{X: x, Y: y, Width: w, Height: h}
	rl.DrawRectangleRounded(cardRect, 0.08, 6, rl.Color{R: 20, G: 25, B: 38, A: 245})
	rl.DrawRectangleRoundedLines(cardRect, 0.08, 6, rl.Color{R: 0, G: 215, B: 255, A: 220})

	// Header
	header := fmt.Sprintf("Sample #%d: %s", pt.Index+1, pt.CategoryName)
	rl.DrawText(header, int32(x)+12, int32(y)+12, 14, rl.Gold)

	// Close [X]
	closeRect := rl.Rectangle{X: x + w - 32, Y: y + 8, Width: 24, Height: 24}
	closed := DrawTouchButton(closeRect, "X", 12, rl.Color{R: 120, G: 40, B: 40, A: 255}, rl.White, mousePos)

	rl.DrawLine(int32(x)+10, int32(y)+34, int32(x+w)-10, int32(y)+34, rl.Color{R: 50, G: 70, B: 100, A: 255})

	// Display values
	for i := range ds.NumericIndices {
		if i >= len(pt.RawValues) {
			break
		}
		itemY := int32(y) + 42 + int32(i*22)
		if float32(itemY) > y+h-20 {
			break
		}
		name := ds.NumericNames[i]
		val := pt.RawValues[i]

		tag := ""
		if i == ui.ColX {
			tag = "[X]"
		} else if i == ui.ColY {
			tag = "[Y]"
		} else if i == ui.ColZ {
			tag = "[Z]"
		} else if i == ui.Col4D {
			tag = "[4D]"
		}

		lineText := fmt.Sprintf("%-14s : %.2f %s", truncStr(name, 12), val, tag)
		valCol := rl.RayWhite
		if tag != "" {
			valCol = rl.Color{R: 255, G: 230, B: 100, A: 255}
		}
		rl.DrawText(lineText, int32(x)+14, itemY, 13, valCol)
	}

	return closed
}

func (ui *UIManager) drawDatasetsModal(x, y, w, h float32, mousePos rl.Vector2, onDatasetChange func(newDs *Dataset)) {
	rl.DrawText("SELECT DATASET", int32(x)+24, int32(y)+20, 20, rl.RayWhite)
	rl.DrawText("Choose a dataset to visualize in 3D / 4D:", int32(x)+24, int32(y)+48, 14, rl.Gray)

	options := []struct {
		title string
		desc  string
		fn    func() *Dataset
	}{
		{
			title: "Fisher's Iris Dataset (Default)",
			desc:  "150 flowers: Sepal & Petal Length/Width (3 species: Setosa, Versicolor, Virginica)",
			fn:    LoadIrisDataset,
		},
		{
			title: "Wine Quality / Recognition",
			desc:  "178 wines with 13 chemical features (Alcohol, Flavanoids, Color, Proline, etc.)",
			fn:    LoadWineDataset,
		},
		{
			title: "Palmer Archipelago Penguins",
			desc:  "344 penguins: Bill Length, Bill Depth, Flipper Length, Body Mass (3 species)",
			fn:    LoadPenguinsDataset,
		},
		{
			title: "Synthetic 4D Hypersurface",
			desc:  "300 mathematical 4D points (Clifford Torus, Hypersphere & Spiral clusters)",
			fn:    LoadSynthetic4DDataset,
		},
	}

	btnY := y + 80
	btnH := float32(56)

	for _, opt := range options {
		rect := rl.Rectangle{X: x + 24, Y: btnY, Width: w - 48, Height: btnH}
		hovered := rl.CheckCollisionPointRec(mousePos, rect)
		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			ds := opt.fn()
			if ds != nil {
				ui.setDefaultMappings(ds)
				onDatasetChange(ds)
				ui.ActiveModal = ModalNone
			}
		}

		bgCol := rl.Color{R: 35, G: 45, B: 65, A: 255}
		if hovered {
			bgCol = rl.Color{R: 50, G: 75, B: 115, A: 255}
		}
		rl.DrawRectangleRounded(rect, 0.2, 6, bgCol)
		rl.DrawRectangleRoundedLines(rect, 0.2, 6, rl.Color{R: 80, G: 110, B: 160, A: 255})

		rl.DrawText(opt.title, int32(x)+38, int32(btnY)+10, 16, rl.Gold)
		rl.DrawText(opt.desc, int32(x)+38, int32(btnY)+32, 12, rl.LightGray)

		btnY += btnH + 12
	}

	// Custom Import Button
	importRect := rl.Rectangle{X: x + 24, Y: btnY + 8, Width: w - 48, Height: 48}
	if DrawTouchButton(importRect, "+ Import Custom CSV / Storage File", 16, rl.Color{R: 20, G: 100, B: 140, A: 255}, rl.White, mousePos) {
		ui.FoundFiles = ScanCSVFiles()
		ui.ActiveModal = ModalImportCSV
	}
}

func (ui *UIManager) drawAxesModal(ds *Dataset, renderer *PlotRenderer, x, y, w, h float32, mousePos rl.Vector2) {
	rl.DrawText("AXES & 4TH DIMENSION MAPPING", int32(x)+24, int32(y)+18, 18, rl.RayWhite)

	if ds == nil || len(ds.NumericNames) == 0 {
		rl.DrawText("No dataset loaded!", int32(x)+24, int32(y)+60, 16, rl.Red)
		return
	}

	// Dimension Selector Tabs: X, Y, Z, 4D
	tabs := []string{"X Axis", "Y Axis", "Z Axis", "4th Dimension", "4D Mode"}
	tabW := (w - 48) / float32(len(tabs))
	tabY := y + 50

	for i, t := range tabs {
		tabRect := rl.Rectangle{X: x + 24 + float32(i)*tabW, Y: tabY, Width: tabW - 4, Height: 34}
		isActive := ui.SelectedAxisTab == i
		bg := rl.Color{R: 35, G: 45, B: 65, A: 255}
		if isActive {
			bg = rl.Color{R: 0, G: 150, B: 200, A: 255}
		}
		if DrawTouchButton(tabRect, t, 13, bg, rl.White, mousePos) {
			ui.SelectedAxisTab = i
		}
	}

	contentY := tabY + 46
	listH := h - 160

	if ui.SelectedAxisTab < 4 {
		// Column picker for X, Y, Z, 4D
		activeCol := ui.ColX
		tabName := "X"
		if ui.SelectedAxisTab == 1 {
			activeCol = ui.ColY
			tabName = "Y"
		} else if ui.SelectedAxisTab == 2 {
			activeCol = ui.ColZ
			tabName = "Z"
		} else if ui.SelectedAxisTab == 3 {
			activeCol = ui.Col4D
			tabName = "4D"
		}

		rl.DrawText(fmt.Sprintf("Select column for %s Axis:", tabName), int32(x)+26, int32(contentY), 14, rl.Gold)

		itemY := contentY + 26
		for i, name := range ds.NumericNames {
			if float32(itemY) > y+h-50 {
				break
			}
			rect := rl.Rectangle{X: x + 24, Y: itemY, Width: w - 48, Height: 38}
			isSelected := i == activeCol
			bg := rl.Color{R: 30, G: 38, B: 55, A: 255}
			if isSelected {
				bg = rl.Color{R: 20, G: 110, B: 90, A: 255}
			}

			label := fmt.Sprintf("%s (Min: %.2f, Max: %.2f)", name, ds.MinVals[i], ds.MaxVals[i])
			if isSelected {
				label = "[ACTIVE] " + label
			}
			if DrawTouchButton(rect, label, 14, bg, rl.White, mousePos) {
				if ui.SelectedAxisTab == 0 {
					ui.ColX = i
				} else if ui.SelectedAxisTab == 1 {
					ui.ColY = i
				} else if ui.SelectedAxisTab == 2 {
					ui.ColZ = i
				} else if ui.SelectedAxisTab == 3 {
					ui.Col4D = i
				}
			}
			itemY += 44
		}
	} else {
		// 4D Mode Picker
		rl.DrawText("Select 4th Dimension Visual Representation:", int32(x)+26, int32(contentY), 14, rl.Gold)
		itemY := contentY + 28
		for m := 0; m < Mode4DCount; m++ {
			rect := rl.Rectangle{X: x + 24, Y: itemY, Width: w - 48, Height: 44}
			isSelected := m == renderer.CurrentMode4D
			bg := rl.Color{R: 30, G: 38, B: 55, A: 255}
			if isSelected {
				bg = rl.Color{R: 110, G: 60, B: 150, A: 255}
			}
			lbl := Mode4DNames[m]
			if isSelected {
				lbl = "[ACTIVE] " + lbl
			}
			if DrawTouchButton(rect, lbl, 15, bg, rl.White, mousePos) {
				renderer.CurrentMode4D = m
			}
			itemY += 50
		}
	}

	// Done button
	doneRect := rl.Rectangle{X: x + w - 130, Y: y + h - 46, Width: 106, Height: 36}
	if DrawTouchButton(doneRect, "Done", 14, rl.Color{R: 40, G: 140, B: 80, A: 255}, rl.White, mousePos) {
		ui.ActiveModal = ModalNone
	}
	_ = listH
}

func (ui *UIManager) drawViewModal(renderer *PlotRenderer, x, y, w, h float32, mousePos rl.Vector2) {
	rl.DrawText("VIEW & GRAPHICS OPTIONS", int32(x)+24, int32(y)+20, 18, rl.RayWhite)

	// Colormap Selection
	rl.DrawText("4D Continuous Colormap:", int32(x)+24, int32(y)+56, 14, rl.Gold)
	cmapW := (w - 48) / float32(len(ColormapNames))
	for i, cname := range ColormapNames {
		rect := rl.Rectangle{X: x + 24 + float32(i)*cmapW, Y: y + 80, Width: cmapW - 6, Height: 38}
		isSelected := renderer.CurrentColormap == i
		bg := rl.Color{R: 35, G: 45, B: 65, A: 255}
		if isSelected {
			bg = rl.Color{R: 0, G: 160, B: 190, A: 255}
		}
		if DrawTouchButton(rect, cname, 13, bg, rl.White, mousePos) {
			renderer.CurrentColormap = i
		}
	}

	// Point Size
	rl.DrawText("Point Sphere Size:", int32(x)+24, int32(y)+136, 14, rl.Gold)
	sizes := []struct {
		name string
		val  float32
	}{
		{"Small", 0.18},
		{"Medium", 0.28},
		{"Large", 0.42},
		{"X-Large", 0.58},
	}
	sizeW := (w - 48) / float32(len(sizes))
	for i, s := range sizes {
		rect := rl.Rectangle{X: x + 24 + float32(i)*sizeW, Y: y + 160, Width: sizeW - 6, Height: 38}
		isSelected := mathAbs(renderer.BasePointSize-s.val) < 0.05
		bg := rl.Color{R: 35, G: 45, B: 65, A: 255}
		if isSelected {
			bg = rl.Color{R: 200, G: 120, B: 30, A: 255}
		}
		if DrawTouchButton(rect, s.name, 13, bg, rl.White, mousePos) {
			renderer.BasePointSize = s.val
		}
	}

	// Toggles (Drop lines, Floor Grid, Box)
	toggleY := y + 224
	toggleW := (w - 48) / 3.0

	// Drop Lines
	rDrop := rl.Rectangle{X: x + 24, Y: toggleY, Width: toggleW - 6, Height: 44}
	dropLbl := ternary(renderer.ShowDropLines, "Drop Lines: ON", "Drop Lines: OFF")
	dropBg := ternaryCol(renderer.ShowDropLines, rl.Color{R: 30, G: 110, B: 70, A: 255}, rl.Color{R: 50, G: 55, B: 65, A: 255})
	if DrawTouchButton(rDrop, dropLbl, 13, dropBg, rl.White, mousePos) {
		renderer.ShowDropLines = !renderer.ShowDropLines
	}

	// Grid
	rGrid := rl.Rectangle{X: x + 24 + toggleW, Y: toggleY, Width: toggleW - 6, Height: 44}
	gridLbl := ternary(renderer.ShowGrid, "Floor Grid: ON", "Floor Grid: OFF")
	gridBg := ternaryCol(renderer.ShowGrid, rl.Color{R: 30, G: 110, B: 70, A: 255}, rl.Color{R: 50, G: 55, B: 65, A: 255})
	if DrawTouchButton(rGrid, gridLbl, 13, gridBg, rl.White, mousePos) {
		renderer.ShowGrid = !renderer.ShowGrid
	}

	// Box Frame
	rBox := rl.Rectangle{X: x + 24 + toggleW*2, Y: toggleY, Width: toggleW - 6, Height: 44}
	boxLbl := ternary(renderer.ShowBox, "Box Cage: ON", "Box Cage: OFF")
	boxBg := ternaryCol(renderer.ShowBox, rl.Color{R: 30, G: 110, B: 70, A: 255}, rl.Color{R: 50, G: 55, B: 65, A: 255})
	if DrawTouchButton(rBox, boxLbl, 13, boxBg, rl.White, mousePos) {
		renderer.ShowBox = !renderer.ShowBox
	}

	// Done button
	doneRect := rl.Rectangle{X: x + w - 130, Y: y + h - 46, Width: 106, Height: 36}
	if DrawTouchButton(doneRect, "Done", 14, rl.Color{R: 40, G: 140, B: 80, A: 255}, rl.White, mousePos) {
		ui.ActiveModal = ModalNone
	}
}

func (ui *UIManager) drawImportCSVModal(x, y, w, h float32, mousePos rl.Vector2, onDatasetChange func(newDs *Dataset)) {
	rl.DrawText("IMPORT CUSTOM CSV DATASET", int32(x)+24, int32(y)+18, 18, rl.RayWhite)
	rl.DrawText("Load any CSV dataset from storage (3D/4D plotting):", int32(x)+24, int32(y)+44, 13, rl.Gray)

	// Rescan button
	scanRect := rl.Rectangle{X: x + w - 150, Y: y + 14, Width: 100, Height: 28}
	if DrawTouchButton(scanRect, "Rescan Files", 12, rl.Color{R: 40, G: 70, B: 110, A: 255}, rl.White, mousePos) {
		ui.FoundFiles = ScanCSVFiles()
	}

	// File List
	listY := y + 70
	rl.DrawText(fmt.Sprintf("Found CSV files on device (%d):", len(ui.FoundFiles)), int32(x)+26, int32(listY), 13, rl.SkyBlue)

	itemY := listY + 22
	maxItems := 4
	if len(ui.FoundFiles) == 0 {
		rl.DrawText("No .csv files found in /sdcard/Download or Documents.", int32(x)+36, int32(itemY)+8, 13, rl.DarkGray)
		rl.DrawText("Tip: Place your CSV files in your device's Download folder.", int32(x)+36, int32(itemY)+28, 12, rl.Gray)
		itemY += 60
	} else {
		for i, fpath := range ui.FoundFiles {
			if i >= maxItems {
				break
			}
			rect := rl.Rectangle{X: x + 24, Y: itemY, Width: w - 48, Height: 38}
			if DrawTouchButton(rect, "Open: "+fpath, 13, rl.Color{R: 30, G: 45, B: 65, A: 255}, rl.White, mousePos) {
				ds, err := LoadDatasetFromFile(fpath)
				if err != nil {
					ui.ImportStatus = "Error: " + err.Error()
					ui.ImportStatusOk = false
				} else {
					ui.setDefaultMappings(ds)
					onDatasetChange(ds)
					ui.ImportStatus = fmt.Sprintf("Successfully loaded %s (%d rows)", ds.Name, len(ds.Points))
					ui.ImportStatusOk = true
					ui.ActiveModal = ModalNone
				}
			}
			itemY += 44
		}
	}

	// Quick Load Synthetic / Demo Custom Dataset button
	demoRect := rl.Rectangle{X: x + 24, Y: itemY + 10, Width: w - 48, Height: 44}
	if DrawTouchButton(demoRect, "Load Demonstration Custom 4D Dataset", 14, rl.Color{R: 110, G: 70, B: 30, A: 255}, rl.White, mousePos) {
		// Generate demo CSV content
		csvDemo := "Temperature,Pressure,Humidity,CO2_Level,Status\n"
		for k := 0; k < 120; k++ {
			temp := 18.0 + float64(k%30)*0.7
			press := 1000.0 + float64((k*7)%50)*1.2
			hum := 30.0 + float64((k*13)%60)*0.8
			co2 := 400.0 + float64((k*23)%800)*1.1
			status := "Normal"
			if co2 > 900 {
				status = "Warning"
			} else if temp > 35 {
				status = "High_Temp"
			}
			csvDemo += fmt.Sprintf("%.1f,%.1f,%.1f,%.1f,%s\n", temp, press, hum, co2, status)
		}
		ds, err := ParseCSV("Custom Sensor 4D Dataset", csvDemo)
		if err == nil {
			ui.setDefaultMappings(ds)
			onDatasetChange(ds)
			ui.ActiveModal = ModalNone
		}
	}

	// Status line
	if ui.ImportStatus != "" {
		stCol := rl.Green
		if !ui.ImportStatusOk {
			stCol = rl.Red
		}
		rl.DrawText(ui.ImportStatus, int32(x)+26, int32(y)+int32(h)-36, 13, stCol)
	}

	// Done button
	doneRect := rl.Rectangle{X: x + w - 130, Y: y + h - 46, Width: 106, Height: 36}
	if DrawTouchButton(doneRect, "Cancel", 14, rl.Color{R: 70, G: 75, B: 85, A: 255}, rl.White, mousePos) {
		ui.ActiveModal = ModalNone
	}
}

func (ui *UIManager) setDefaultMappings(ds *Dataset) {
	n := len(ds.NumericNames)
	ui.ColX = 0
	if n > 1 {
		ui.ColY = 1
	} else {
		ui.ColY = 0
	}
	if n > 2 {
		ui.ColZ = 2
	} else {
		ui.ColZ = 0
	}
	if n > 3 {
		ui.Col4D = 3
	} else {
		ui.Col4D = 0
	}
}

// IsPointerOnUI checks if touch/mouse coordinates fall on any interactive UI element
func (ui *UIManager) IsPointerOnUI(screenW, screenH float32, mousePos rl.Vector2, hasSelectedPoint bool) bool {
	if ui.ActiveModal != ModalNone {
		return true
	}
	// Top status / title bar
	if mousePos.Y <= 52 {
		return true
	}
	// Bottom control buttons bar
	btnH := float32(42)
	bottomY := screenH - btnH - 18
	if mousePos.Y >= bottomY {
		return true
	}
	// Point inspection card (top right)
	if hasSelectedPoint {
		cardRect := rl.Rectangle{X: screenW - 310, Y: 48 + 14, Width: 294, Height: 240}
		if rl.CheckCollisionPointRec(mousePos, cardRect) {
			return true
		}
	}
	return false
}

func truncStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func ternaryCol(cond bool, a, b rl.Color) rl.Color {
	if cond {
		return a
	}
	return b
}

func mathAbs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
