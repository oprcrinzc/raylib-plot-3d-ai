package main

import (
	"testing"
)

func TestIrisDataset(t *testing.T) {
	ds := LoadIrisDataset()
	if ds == nil {
		t.Fatalf("Failed to load Iris dataset")
	}
	if len(ds.Points) != 150 {
		t.Errorf("Expected 150 points in Iris dataset, got %d", len(ds.Points))
	}
	if len(ds.NumericNames) != 4 {
		t.Errorf("Expected 4 numeric columns in Iris, got %d", len(ds.NumericNames))
	}
	if len(ds.Categories) != 3 {
		t.Errorf("Expected 3 species in Iris, got %d", len(ds.Categories))
	}

	// Verify normalization
	for i := 0; i < len(ds.Points); i++ {
		norm := ds.Get01NormalizedValue(i, 0)
		if norm < 0.0 || norm > 1.0 {
			t.Errorf("Normalized value out of bounds: %f", norm)
		}
	}
}

func TestWineDataset(t *testing.T) {
	ds := LoadWineDataset()
	if ds == nil {
		t.Fatalf("Failed to load Wine dataset")
	}
	if len(ds.Points) != 178 {
		t.Errorf("Expected 178 points in Wine dataset, got %d", len(ds.Points))
	}
	if len(ds.NumericNames) < 10 {
		t.Errorf("Expected >= 10 numeric features in Wine, got %d", len(ds.NumericNames))
	}
}

func TestPenguinsDataset(t *testing.T) {
	ds := LoadPenguinsDataset()
	if ds == nil {
		t.Fatalf("Failed to load Penguins dataset")
	}
	if len(ds.Points) < 300 {
		t.Errorf("Expected >= 300 points in Penguins dataset, got %d", len(ds.Points))
	}
}

func TestSynthetic4DDataset(t *testing.T) {
	ds := LoadSynthetic4DDataset()
	if ds == nil {
		t.Fatalf("Failed to load Synthetic dataset")
	}
	if len(ds.Points) != 300 {
		t.Errorf("Expected 300 points in Synthetic dataset, got %d", len(ds.Points))
	}
	if len(ds.NumericNames) != 4 {
		t.Errorf("Expected 4 continuous dimensions, got %d", len(ds.NumericNames))
	}
}

func TestColormaps(t *testing.T) {
	for m := 0; m < ColormapCount; m++ {
		c0 := GetContinuousColor(0.0, m)
		cMid := GetContinuousColor(0.5, m)
		c1 := GetContinuousColor(1.0, m)
		if c0.A != 255 || cMid.A != 255 || c1.A != 255 {
			t.Errorf("Colormap %d produced invalid alpha", m)
		}
	}
}
