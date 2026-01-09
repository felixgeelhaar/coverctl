// Package parsers provides a unified registry for coverage profile parsers.
//
// The registry automatically detects profile formats and selects the appropriate parser.
package parsers

import (
	"fmt"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/coverprofile"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/parsers/cobertura"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/parsers/detector"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/parsers/lcov"
)

// Registry manages multiple profile parsers and auto-detects formats.
type Registry struct {
	detector *detector.Detector
	parsers  map[application.Format]application.ProfileParser
}

// NewRegistry creates a new parser registry with all supported parsers.
func NewRegistry() *Registry {
	return &Registry{
		detector: detector.New(),
		parsers: map[application.Format]application.ProfileParser{
			application.FormatGo:        coverprofile.Parser{},
			application.FormatLCOV:      lcov.New(),
			application.FormatCobertura: cobertura.New(),
		},
	}
}

// Format returns the auto format, since the registry handles all formats.
func (r *Registry) Format() application.Format {
	return application.FormatAuto
}

// Parse parses a coverage profile, auto-detecting the format.
func (r *Registry) Parse(path string) (map[string]domain.CoverageStat, error) {
	format, err := r.detector.DetectFormat(path)
	if err != nil {
		return nil, fmt.Errorf("detect format: %w", err)
	}

	parser, err := r.getParser(format, path)
	if err != nil {
		return nil, err
	}

	return parser.Parse(path)
}

// ParseAll parses multiple profiles, potentially with different formats.
func (r *Registry) ParseAll(paths []string) (map[string]domain.CoverageStat, error) {
	merged := make(map[string]domain.CoverageStat)

	for _, path := range paths {
		stats, err := r.Parse(path)
		if err != nil {
			return nil, err
		}

		for file, stat := range stats {
			existing := merged[file]
			existing.Total += stat.Total
			existing.Covered += stat.Covered
			merged[file] = existing
		}
	}

	return merged, nil
}

// ParseWithFormat parses a profile using a specific format (no auto-detection).
func (r *Registry) ParseWithFormat(path string, format application.Format) (map[string]domain.CoverageStat, error) {
	parser, ok := r.parsers[format]
	if !ok {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	return parser.Parse(path)
}

// getParser returns the appropriate parser for a detected format.
func (r *Registry) getParser(format application.Format, path string) (application.ProfileParser, error) {
	// If format couldn't be detected, try Go profile as default
	if format == application.FormatAuto {
		format = application.FormatGo
	}

	parser, ok := r.parsers[format]
	if !ok {
		return nil, fmt.Errorf("no parser available for format: %s", format)
	}

	return parser, nil
}

// SupportedFormats returns a list of all supported formats.
func (r *Registry) SupportedFormats() []application.Format {
	formats := make([]application.Format, 0, len(r.parsers))
	for format := range r.parsers {
		formats = append(formats, format)
	}
	return formats
}

// DetectLanguage detects the programming language of a project.
func (r *Registry) DetectLanguage(projectDir string) (application.Language, error) {
	return r.detector.DetectLanguage(projectDir)
}

// GetDefaultProfilePaths returns common coverage profile paths for a language.
func (r *Registry) GetDefaultProfilePaths(lang application.Language) []string {
	return r.detector.GetDefaultProfilePaths(lang)
}

// GetDefaultFormat returns the default coverage format for a language.
func (r *Registry) GetDefaultFormat(lang application.Language) application.Format {
	return r.detector.GetDefaultFormat(lang)
}
