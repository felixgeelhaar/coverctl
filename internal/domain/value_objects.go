package domain

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Value object errors.
var (
	ErrInvalidThreshold = errors.New("threshold must be between 0 and 100")
	ErrEmptyDomainName  = errors.New("domain name cannot be empty")
	ErrEmptyFilePath    = errors.New("file path cannot be empty")
)

// Threshold represents a coverage threshold percentage (0-100).
// It is a value object that ensures threshold values are always valid.
type Threshold struct {
	value float64
}

// NewThreshold creates a new Threshold value object.
// Returns an error if the value is not between 0 and 100.
func NewThreshold(value float64) (Threshold, error) {
	if value < 0 || value > 100 {
		return Threshold{}, ErrInvalidThreshold
	}
	return Threshold{value: value}, nil
}

// MustThreshold creates a new Threshold, panicking if invalid.
// Use only when the value is known to be valid at compile time.
func MustThreshold(value float64) Threshold {
	t, err := NewThreshold(value)
	if err != nil {
		panic(err)
	}
	return t
}

// ThresholdFromPtr creates a Threshold from a pointer, using defaultValue if nil.
func ThresholdFromPtr(ptr *float64, defaultValue float64) Threshold {
	if ptr != nil {
		return MustThreshold(*ptr)
	}
	return MustThreshold(defaultValue)
}

// Value returns the threshold percentage value.
func (t Threshold) Value() float64 {
	return t.value
}

// IsMet returns true if the given coverage percentage meets this threshold.
func (t Threshold) IsMet(coveragePercent float64) bool {
	return coveragePercent >= t.value
}

// IsExceededBy returns true if the given coverage exceeds this threshold.
func (t Threshold) IsExceededBy(coveragePercent float64) bool {
	return coveragePercent > t.value
}

// Shortfall returns how many percentage points below the threshold the coverage is.
// Returns 0 if the threshold is met.
func (t Threshold) Shortfall(coveragePercent float64) float64 {
	if t.IsMet(coveragePercent) {
		return 0
	}
	return Round1(t.value - coveragePercent)
}

// String returns a formatted string representation.
func (t Threshold) String() string {
	return fmt.Sprintf("%.1f%%", t.value)
}

// Equals returns true if two thresholds have the same value.
func (t Threshold) Equals(other Threshold) bool {
	return t.value == other.value
}

// ZeroThreshold returns a threshold of 0%.
func ZeroThreshold() Threshold {
	return Threshold{value: 0}
}

// Ptr returns a pointer to the threshold value (for backward compatibility).
func (t Threshold) Ptr() *float64 {
	v := t.value
	return &v
}

// DomainName represents a coverage domain name.
// It is a value object that ensures domain names are never empty.
type DomainName struct {
	value string
}

// NewDomainName creates a new DomainName value object.
// Returns an error if the name is empty.
func NewDomainName(name string) (DomainName, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return DomainName{}, ErrEmptyDomainName
	}
	return DomainName{value: trimmed}, nil
}

// MustDomainName creates a new DomainName, panicking if invalid.
func MustDomainName(name string) DomainName {
	dn, err := NewDomainName(name)
	if err != nil {
		panic(err)
	}
	return dn
}

// String returns the domain name string.
func (n DomainName) String() string {
	return n.value
}

// Equals returns true if two domain names are equal.
func (n DomainName) Equals(other DomainName) bool {
	return n.value == other.value
}

// IsEmpty returns true if the domain name is empty (zero value).
func (n DomainName) IsEmpty() bool {
	return n.value == ""
}

// FilePath represents a normalized file path.
// It is a value object that ensures file paths are cleaned and consistent.
type FilePath struct {
	value string
}

// NewFilePath creates a new FilePath value object.
// The path is cleaned and normalized.
func NewFilePath(path string) (FilePath, error) {
	if path == "" {
		return FilePath{}, ErrEmptyFilePath
	}
	cleaned := filepath.Clean(path)
	normalized := filepath.ToSlash(cleaned)
	return FilePath{value: normalized}, nil
}

// MustFilePath creates a new FilePath, panicking if invalid.
func MustFilePath(path string) FilePath {
	fp, err := NewFilePath(path)
	if err != nil {
		panic(err)
	}
	return fp
}

// String returns the normalized file path string.
func (p FilePath) String() string {
	return p.value
}

// Equals returns true if two file paths are equal.
func (p FilePath) Equals(other FilePath) bool {
	return p.value == other.value
}

// Dir returns the directory portion of the path.
func (p FilePath) Dir() FilePath {
	dir := filepath.Dir(p.value)
	return FilePath{value: filepath.ToSlash(dir)}
}

// Base returns the last element of the path.
func (p FilePath) Base() string {
	return filepath.Base(p.value)
}

// HasPrefix returns true if the path starts with the given prefix.
func (p FilePath) HasPrefix(prefix string) bool {
	return strings.HasPrefix(p.value, prefix)
}

// IsEmpty returns true if the file path is empty (zero value).
func (p FilePath) IsEmpty() bool {
	return p.value == ""
}

// MatchesPattern returns true if the path matches the given glob pattern.
func (p FilePath) MatchesPattern(pattern string) bool {
	matched, _ := filepath.Match(pattern, p.value)
	return matched
}

// MatchesAnyPattern returns true if the path matches any of the given patterns.
func (p FilePath) MatchesAnyPattern(patterns []string) bool {
	for _, pattern := range patterns {
		if p.MatchesPattern(pattern) {
			return true
		}
	}
	return false
}

// Percentage represents a coverage percentage value.
// It is a value object for representing calculated percentages.
type Percentage struct {
	value float64
}

// NewPercentage creates a new Percentage from a raw value.
// The value is rounded to one decimal place.
func NewPercentage(value float64) Percentage {
	return Percentage{value: Round1(value)}
}

// PercentageFromRatio calculates a percentage from covered/total ratio.
func PercentageFromRatio(covered, total int) Percentage {
	if total == 0 {
		return Percentage{value: 0}
	}
	return NewPercentage((float64(covered) / float64(total)) * 100)
}

// Value returns the percentage value.
func (p Percentage) Value() float64 {
	return p.value
}

// String returns a formatted string representation.
func (p Percentage) String() string {
	return fmt.Sprintf("%.1f%%", p.value)
}

// IsZero returns true if the percentage is zero.
func (p Percentage) IsZero() bool {
	return p.value == 0
}

// GreaterThan returns true if this percentage is greater than another.
func (p Percentage) GreaterThan(other Percentage) bool {
	return p.value > other.value
}

// LessThan returns true if this percentage is less than another.
func (p Percentage) LessThan(other Percentage) bool {
	return p.value < other.value
}

// Equals returns true if two percentages are equal.
func (p Percentage) Equals(other Percentage) bool {
	return p.value == other.value
}

// Difference returns the difference between two percentages.
func (p Percentage) Difference(other Percentage) float64 {
	return Round1(p.value - other.value)
}

// MeetsThreshold returns true if this percentage meets the given threshold.
func (p Percentage) MeetsThreshold(threshold Threshold) bool {
	return threshold.IsMet(p.value)
}

// Delta returns the delta from another percentage (this - other).
func (p Percentage) Delta(other Percentage) float64 {
	return Round1(p.value - other.value)
}
