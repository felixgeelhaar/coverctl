package badge

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenerateBadge(t *testing.T) {
	buf := new(bytes.Buffer)
	opts := Options{
		Label:   "coverage",
		Percent: 85.5,
		Style:   StyleFlat,
	}
	if err := Generate(buf, opts); err != nil {
		t.Fatalf("generate: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "<svg") {
		t.Fatal("expected SVG element")
	}
	if !strings.Contains(output, "coverage") {
		t.Fatal("expected label in output")
	}
	if !strings.Contains(output, "85.5%") {
		t.Fatal("expected percentage in output")
	}
}

func TestGenerateBadgeColors(t *testing.T) {
	tests := []struct {
		name      string
		percent   float64
		wantColor string
	}{
		{"low", 50, "#e05d44"},      // red
		{"medium", 65, "#dfb317"},   // yellow
		{"good", 80, "#97ca00"},     // light green
		{"excellent", 90, "#4c1"},   // green
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			opts := Options{
				Label:   "coverage",
				Percent: tc.percent,
				Style:   StyleFlat,
			}
			if err := Generate(buf, opts); err != nil {
				t.Fatalf("generate: %v", err)
			}
			if !strings.Contains(buf.String(), tc.wantColor) {
				t.Fatalf("expected color %s for %f%%", tc.wantColor, tc.percent)
			}
		})
	}
}

func TestGenerateBadgeFlatSquareStyle(t *testing.T) {
	buf := new(bytes.Buffer)
	opts := Options{
		Label:   "coverage",
		Percent: 75,
		Style:   StyleFlatSquare,
	}
	if err := Generate(buf, opts); err != nil {
		t.Fatalf("generate: %v", err)
	}
	output := buf.String()
	// Flat square has no border radius
	if strings.Contains(output, "rx=\"3\"") {
		t.Fatal("flat-square should not have rounded corners")
	}
}

func TestGenerateBadgeDefaultStyle(t *testing.T) {
	buf := new(bytes.Buffer)
	opts := Options{
		Label:   "coverage",
		Percent: 75,
		// No style specified - should default to flat
	}
	if err := Generate(buf, opts); err != nil {
		t.Fatalf("generate: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "<svg") {
		t.Fatal("expected SVG element")
	}
}

func TestGenerateBadgeCustomLabel(t *testing.T) {
	buf := new(bytes.Buffer)
	opts := Options{
		Label:   "test coverage",
		Percent: 100,
		Style:   StyleFlat,
	}
	if err := Generate(buf, opts); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(buf.String(), "test coverage") {
		t.Fatal("expected custom label")
	}
}

func TestGenerateBadge100Percent(t *testing.T) {
	buf := new(bytes.Buffer)
	opts := Options{
		Label:   "coverage",
		Percent: 100,
		Style:   StyleFlat,
	}
	if err := Generate(buf, opts); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(buf.String(), "100%") {
		t.Fatal("expected 100%")
	}
}

func TestGenerateBadge0Percent(t *testing.T) {
	buf := new(bytes.Buffer)
	opts := Options{
		Label:   "coverage",
		Percent: 0,
		Style:   StyleFlat,
	}
	if err := Generate(buf, opts); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(buf.String(), "0%") {
		t.Fatal("expected 0%")
	}
}
