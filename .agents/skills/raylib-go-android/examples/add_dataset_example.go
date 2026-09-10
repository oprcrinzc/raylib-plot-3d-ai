// Example demonstrating how to add a new embedded or dynamic scientific dataset
// to the Iris 3D/4D visualizer.
package main

import (
	"math"
	"math/rand"
)

// Example 1: Creating a Synthetic Lorenz Attractor 4D Dataset
func LoadLorenzAttractorDataset() *Dataset {
	numPoints := 500
	points := make([]DataPoint, numPoints)
	categories := []string{"Wing Left", "Wing Right"}

	minVals := make([]float32, 4)
	maxVals := make([]float32, 4)
	for i := range minVals {
		minVals[i] = float32(math.MaxFloat32)
		maxVals[i] = -float32(math.MaxFloat32)
	}

	// Lorenz system parameters
	sigma := 10.0
	rho := 28.0
	beta := 8.0 / 3.0
	dt := 0.01

	x, y, z := 0.1, 0.0, 0.0

	for i := 0; i < numPoints; i++ {
		// Runge-Kutta or Euler step
		dx := sigma * (y - x) * dt
		dy = (x*(rho-z) - y) * dt
		dz := (x*y - beta*z) * dt
		x += dx
		y += dy
		z += dz

		// 4th dimension: kinetic velocity magnitude
		velocity := float32(math.Sqrt(dx*dx + dy*dy + dz*dz))

		vals := []float32{float32(x), float32(y), float32(z), velocity}
		for j, v := range vals {
			if v < minVals[j] {
				minVals[j] = v
			}
			if v > maxVals[j] {
				maxVals[j] = v
			}
		}

		catIdx := 0
		if x > 0 {
			catIdx = 1
		}

		points[i] = DataPoint{
			Index:        i,
			RawValues:    vals,
			CategoryName: categories[catIdx],
			CategoryIdx:  catIdx,
		}
	}

	return &Dataset{
		Name:               "Lorenz Attractor 4D",
		AllHeaders:         []string{"X", "Y", "Z", "Velocity", "Wing"},
		AllTypes:           []ColType{ColTypeNumeric, ColTypeNumeric, ColTypeNumeric, ColTypeNumeric, ColTypeCategorical},
		NumericIndices:     []int{0, 1, 2, 3},
		NumericNames:       []string{"X", "Y", "Z", "Velocity"},
		CategoricalIndices: []int{4},
		CategoricalNames:   []string{"Wing"},
		Points:             points,
		Categories:         categories,
		CategoryColIndex:   4,
		MinVals:            minVals,
		MaxVals:            maxVals,
		MeanVals:           []float32{0, 0, 0, 0},
	}
}

// Example 2: How to embed a new CSV file
//
// 1. Add your file: `my_data.csv`
// 2. In dataset.go, add:
//
//    //go:embed my_data.csv
//    var defaultMyDataCSV string
//
//    func LoadMyDataset() *Dataset {
//        ds, err := ParseCSV("My Custom Dataset", defaultMyDataCSV)
//        if err != nil {
//            fmt.Printf("Error: %v\n", err)
//        }
//        return ds
//    }
//
// 3. In ui.go (DrawDatasetsModal), add the button to the dataset selection list:
//    {
//        name: "My Custom Dataset",
//        loader: LoadMyDataset,
//    },
