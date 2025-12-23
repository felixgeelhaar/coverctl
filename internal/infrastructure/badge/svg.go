package badge

import (
	"fmt"
	"html/template"
	"io"
)

type Style string

const (
	StyleFlat       Style = "flat"
	StyleFlatSquare Style = "flat-square"
)

type Options struct {
	Label   string
	Percent float64
	Style   Style
}

const svgTemplate = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="{{.Width}}" height="20" role="img" aria-label="{{.Label}}: {{.PercentText}}">
  <title>{{.Label}}: {{.PercentText}}</title>
  <linearGradient id="s" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r">
    <rect width="{{.Width}}" height="20" rx="{{.Rx}}" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#r)">
    <rect width="{{.LabelWidth}}" height="20" fill="#555"/>
    <rect x="{{.LabelWidth}}" width="{{.ValueWidth}}" height="20" fill="{{.Color}}"/>
    <rect width="{{.Width}}" height="20" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="110">
    <text aria-hidden="true" x="{{.LabelX}}" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="{{.LabelTextWidth}}">{{.Label}}</text>
    <text x="{{.LabelX}}" y="140" transform="scale(.1)" fill="#fff" textLength="{{.LabelTextWidth}}">{{.Label}}</text>
    <text aria-hidden="true" x="{{.ValueX}}" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="{{.ValueTextWidth}}">{{.PercentText}}</text>
    <text x="{{.ValueX}}" y="140" transform="scale(.1)" fill="#fff" textLength="{{.ValueTextWidth}}">{{.PercentText}}</text>
  </g>
</svg>`

type templateData struct {
	Label          string
	PercentText    string
	Color          string
	Width          int
	LabelWidth     int
	ValueWidth     int
	LabelX         int
	ValueX         int
	LabelTextWidth int
	ValueTextWidth int
	Rx             int
}

func Generate(w io.Writer, opts Options) error {
	if opts.Style == "" {
		opts.Style = StyleFlat
	}

	percentText := formatPercent(opts.Percent)
	color := colorForPercent(opts.Percent)

	// Calculate widths based on text length
	labelWidth := len(opts.Label)*7 + 10
	valueWidth := len(percentText)*7 + 10
	totalWidth := labelWidth + valueWidth

	rx := 3
	if opts.Style == StyleFlatSquare {
		rx = 0
	}

	data := templateData{
		Label:          opts.Label,
		PercentText:    percentText,
		Color:          color,
		Width:          totalWidth,
		LabelWidth:     labelWidth,
		ValueWidth:     valueWidth,
		LabelX:         labelWidth * 5,          // Centered in label section (scaled by 10)
		ValueX:         (labelWidth + valueWidth/2) * 10, // Centered in value section
		LabelTextWidth: (len(opts.Label) * 7) * 10,
		ValueTextWidth: (len(percentText) * 7) * 10,
		Rx:             rx,
	}

	tmpl, err := template.New("badge").Parse(svgTemplate)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	return tmpl.Execute(w, data)
}

func formatPercent(p float64) string {
	if p == float64(int(p)) {
		return fmt.Sprintf("%.0f%%", p)
	}
	return fmt.Sprintf("%.1f%%", p)
}

func colorForPercent(p float64) string {
	switch {
	case p >= 90:
		return "#4c1" // Bright green
	case p >= 75:
		return "#97ca00" // Light green
	case p >= 60:
		return "#dfb317" // Yellow
	default:
		return "#e05d44" // Red
	}
}
