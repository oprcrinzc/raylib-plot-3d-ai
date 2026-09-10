package main

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed iris.csv
var defaultIrisCSV string

//go:embed wine.csv
var defaultWineCSV string

//go:embed penguins.csv
var defaultPenguinsCSV string

type ColType int

const (
	ColTypeNumeric ColType = iota
	ColTypeCategorical
)

type DataPoint struct {
	Index        int
	RawValues    []float32 // aligned with Dataset.NumericIndices
	CategoryName string
	CategoryIdx  int
}

type Dataset struct {
	Name               string
	AllHeaders         []string
	AllTypes           []ColType
	NumericIndices     []int    // Indices in AllHeaders that are numeric
	NumericNames       []string // Names of numeric columns
	CategoricalIndices []int    // Indices in AllHeaders that are categorical
	CategoricalNames   []string

	Points           []DataPoint
	Categories       []string // Unique category names
	CategoryColIndex int      // Selected categorical column index (in AllHeaders), -1 if none

	MinVals []float32 // Min of each numeric column (len = len(NumericIndices))
	MaxVals []float32 // Max of each numeric column
	MeanVals []float32
}

func (d *Dataset) GetNumericColCount() int {
	return len(d.NumericIndices)
}

func (d *Dataset) GetNormalizedValue(pointIdx int, numColIdx int, outMin, outMax float32) float32 {
	if pointIdx < 0 || pointIdx >= len(d.Points) {
		return 0
	}
	if numColIdx < 0 || numColIdx >= len(d.NumericIndices) {
		return 0
	}
	raw := d.Points[pointIdx].RawValues[numColIdx]
	min := d.MinVals[numColIdx]
	max := d.MaxVals[numColIdx]
	if math.Abs(float64(max-min)) < 1e-6 {
		return (outMin + outMax) * 0.5
	}
	norm01 := (raw - min) / (max - min)
	if norm01 < 0 {
		norm01 = 0
	}
	if norm01 > 1 {
		norm01 = 1
	}
	return outMin + norm01*(outMax-outMin)
}

func (d *Dataset) Get01NormalizedValue(pointIdx int, numColIdx int) float32 {
	return d.GetNormalizedValue(pointIdx, numColIdx, 0.0, 1.0)
}

// ParseCSV parses raw CSV/TSV text into a Dataset
func ParseCSV(name, content string) (*Dataset, error) {
	cleanContent := strings.TrimSpace(content)
	if cleanContent == "" {
		return nil, fmt.Errorf("empty CSV content")
	}

	// Detect delimiter (comma, tab, semicolon)
	firstLine := cleanContent
	if idx := strings.IndexAny(cleanContent, "\r\n"); idx != -1 {
		firstLine = cleanContent[:idx]
	}
	commaCount := strings.Count(firstLine, ",")
	tabCount := strings.Count(firstLine, "\t")
	semiCount := strings.Count(firstLine, ";")

	delimiter := ','
	if tabCount > commaCount && tabCount > semiCount {
		delimiter = '\t'
	} else if semiCount > commaCount && semiCount > tabCount {
		delimiter = ';'
	}

	reader := csv.NewReader(strings.NewReader(cleanContent))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil && len(records) == 0 {
		return nil, fmt.Errorf("CSV read error: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV needs at least 2 rows (header + data)")
	}

	headers := records[0]
	numCols := len(headers)

	// Determine column types by sampling data rows
	numericVotes := make([]int, numCols)
	totalVotes := make([]int, numCols)

	for r := 1; r < len(records); r++ {
		row := records[r]
		for c := 0; c < numCols && c < len(row); c++ {
			val := strings.TrimSpace(row[c])
			if val == "" || val == "NA" || val == "NaN" || val == "?" || val == "null" {
				continue
			}
			totalVotes[c]++
			if _, err := strconv.ParseFloat(val, 32); err == nil {
				numericVotes[c]++
			}
		}
	}

	allTypes := make([]ColType, numCols)
	var numericIndices []int
	var numericNames []string
	var categoricalIndices []int
	var categoricalNames []string

	for c := 0; c < numCols; c++ {
		if totalVotes[c] > 0 && float64(numericVotes[c])/float64(totalVotes[c]) >= 0.7 {
			allTypes[c] = ColTypeNumeric
			numericIndices = append(numericIndices, c)
			cleanHeader := strings.TrimSpace(headers[c])
			if cleanHeader == "" {
				cleanHeader = fmt.Sprintf("Col_%d", c+1)
			}
			numericNames = append(numericNames, cleanHeader)
		} else {
			allTypes[c] = ColTypeCategorical
			categoricalIndices = append(categoricalIndices, c)
			cleanHeader := strings.TrimSpace(headers[c])
			if cleanHeader == "" {
				cleanHeader = fmt.Sprintf("Category_%d", c+1)
			}
			categoricalNames = append(categoricalNames, cleanHeader)
		}
	}

	if len(numericIndices) == 0 {
		return nil, fmt.Errorf("no numeric columns found in dataset")
	}

	// Identify primary categorical column (prefer last column or one with 2..20 unique items)
	catCol := -1
	if len(categoricalIndices) > 0 {
		catCol = categoricalIndices[len(categoricalIndices)-1]
	}

	// Parse records
	var points []DataPoint
	categoryMap := make(map[string]int)
	var categories []string

	numNumCols := len(numericIndices)
	minVals := make([]float32, numNumCols)
	maxVals := make([]float32, numNumCols)
	sumVals := make([]float64, numNumCols)
	countVals := make([]int, numNumCols)

	for i := range minVals {
		minVals[i] = float32(math.MaxFloat32)
		maxVals[i] = -float32(math.MaxFloat32)
	}

	pointIdx := 0
	for r := 1; r < len(records); r++ {
		row := records[r]
		if len(row) < numCols {
			continue
		}

		rawNums := make([]float32, numNumCols)
		valid := true
		for ni, c := range numericIndices {
			valStr := strings.TrimSpace(row[c])
			valFloat, err := strconv.ParseFloat(valStr, 32)
			if err != nil {
				// Missing or invalid numeric
				valid = false
				break
			}
			f32 := float32(valFloat)
			rawNums[ni] = f32
			if f32 < minVals[ni] {
				minVals[ni] = f32
			}
			if f32 > maxVals[ni] {
				maxVals[ni] = f32
			}
			sumVals[ni] += float64(f32)
			countVals[ni]++
		}

		if !valid {
			continue
		}

		catName := "Default"
		catIdx := 0
		if catCol >= 0 && catCol < len(row) {
			catName = strings.TrimSpace(row[catCol])
			if catName == "" {
				catName = "Unknown"
			}
			if idx, ok := categoryMap[catName]; ok {
				catIdx = idx
			} else {
				catIdx = len(categories)
				categoryMap[catName] = catIdx
				categories = append(categories, catName)
			}
		}

		points = append(points, DataPoint{
			Index:        pointIdx,
			RawValues:    rawNums,
			CategoryName: catName,
			CategoryIdx:  catIdx,
		})
		pointIdx++
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("no valid data points parsed from CSV")
	}

	meanVals := make([]float32, numNumCols)
	for i := range meanVals {
		if countVals[i] > 0 {
			meanVals[i] = float32(sumVals[i] / float64(countVals[i]))
		}
	}

	ds := &Dataset{
		Name:               name,
		AllHeaders:         headers,
		AllTypes:           allTypes,
		NumericIndices:     numericIndices,
		NumericNames:       numericNames,
		CategoricalIndices: categoricalIndices,
		CategoricalNames:   categoricalNames,
		Points:             points,
		Categories:         categories,
		CategoryColIndex:   catCol,
		MinVals:            minVals,
		MaxVals:            maxVals,
		MeanVals:           meanVals,
	}

	return ds, nil
}

// LoadIrisDataset loads the embedded Iris dataset
func LoadIrisDataset() *Dataset {
	ds, err := ParseCSV("Fisher's Iris Dataset", defaultIrisCSV)
	if err != nil {
		fmt.Printf("Error loading Iris: %v\n", err)
	}
	return ds
}

// LoadWineDataset loads the embedded Wine Quality dataset
func LoadWineDataset() *Dataset {
	ds, err := ParseCSV("Wine Recognition (13 Features)", defaultWineCSV)
	if err != nil {
		fmt.Printf("Error loading Wine: %v\n", err)
	}
	return ds
}

// LoadPenguinsDataset loads the embedded Palmer Penguins dataset
func LoadPenguinsDataset() *Dataset {
	ds, err := ParseCSV("Palmer Penguins", defaultPenguinsCSV)
	if err != nil {
		fmt.Printf("Error loading Penguins: %v\n", err)
	}
	return ds
}

// LoadSynthetic4DDataset creates a mathematical 4D Hypersphere/Clifford-Torus dataset
func LoadSynthetic4DDataset() *Dataset {
	numPoints := 300
	points := make([]DataPoint, numPoints)
	categories := []string{"Cluster Alpha", "Cluster Beta", "Cluster Gamma"}

	minVals := make([]float32, 4)
	maxVals := make([]float32, 4)
	for i := range minVals {
		minVals[i] = float32(math.MaxFloat32)
		maxVals[i] = -float32(math.MaxFloat32)
	}

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < numPoints; i++ {
		catIdx := i % 3
		var x, y, z, w float32

		u := rng.Float64() * 2 * math.Pi
		v := rng.Float64() * 2 * math.Pi
		noise := (rng.Float64() - 0.5) * 0.15

		switch catIdx {
		case 0:
			// Clifford Torus branch 1
			r1, r2 := 1.0, 0.6
			x = float32(r1*math.Cos(u) + noise)
			y = float32(r1*math.Sin(u) + noise)
			z = float32(r2*math.Cos(v) + noise)
			w = float32(r2*math.Sin(v) + noise)
		case 1:
			// Spherical cluster branch 2
			theta := rng.Float64() * math.Pi
			phi := rng.Float64() * 2 * math.Pi
			r := 0.8 + rng.Float64()*0.4
			x = float32(r*math.Sin(theta)*math.Cos(phi) + 1.2)
			y = float32(r*math.Sin(theta)*math.Sin(phi) - 0.5)
			z = float32(r*math.Cos(theta) + 0.8)
			w = float32(math.Sin(u+v)*0.7 + 0.5)
		case 2:
			// Spiral saddle branch 3
			t := (float64(i) / float64(numPoints)) * 4 * math.Pi
			x = float32(math.Cos(t)*(0.5+0.1*t) - 1.0)
			y = float32(math.Sin(t)*(0.5+0.1*t) + 0.5)
			z = float32((t/(4*math.Pi))*2.0 - 1.0)
			w = float32(math.Cos(t*1.5)*0.8 + 0.2)
		}

		vals := []float32{x, y, z, w}
		for j, v := range vals {
			if v < minVals[j] {
				minVals[j] = v
			}
			if v > maxVals[j] {
				maxVals[j] = v
			}
		}

		points[i] = DataPoint{
			Index:        i,
			RawValues:    vals,
			CategoryName: categories[catIdx],
			CategoryIdx:  catIdx,
		}
	}

	return &Dataset{
		Name:               "Synthetic 4D Hypersurface",
		AllHeaders:         []string{"Dim_X", "Dim_Y", "Dim_Z", "Dim_W (4th)", "Cluster"},
		AllTypes:           []ColType{ColTypeNumeric, ColTypeNumeric, ColTypeNumeric, ColTypeNumeric, ColTypeCategorical},
		NumericIndices:     []int{0, 1, 2, 3},
		NumericNames:       []string{"Dim_X", "Dim_Y", "Dim_Z", "Dim_W (4th)"},
		CategoricalIndices: []int{4},
		CategoricalNames:   []string{"Cluster"},
		Points:             points,
		Categories:         categories,
		CategoryColIndex:   4,
		MinVals:            minVals,
		MaxVals:            maxVals,
		MeanVals:           []float32{0, 0, 0, 0},
	}
}

// LoadDatasetFromFile loads an external CSV file from storage
func LoadDatasetFromFile(path string) (*Dataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	baseName := filepath.Base(path)
	return ParseCSV(baseName, string(data))
}

// ScanCSVFiles looks for CSV files in common Android external storage directories
func ScanCSVFiles() []string {
	var found []string
	searchDirs := []string{
		"/sdcard/Download",
		"/sdcard/Documents",
		"/sdcard",
		"/storage/emulated/0/Download",
		"/storage/emulated/0/Documents",
		"/storage/emulated/0",
		".",
	}

	seen := make(map[string]bool)
	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".csv") {
				full := filepath.Join(dir, e.Name())
				if !seen[full] {
					seen[full] = true
					found = append(found, full)
				}
			}
		}
	}
	return found
}
