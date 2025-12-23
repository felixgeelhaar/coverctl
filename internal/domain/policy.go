package domain

import "math"

// CoverageStat summarizes covered vs total statements.
type CoverageStat struct {
	Covered int
	Total   int
}

func (c CoverageStat) Percent() float64 {
	if c.Total == 0 {
		return 0
	}
	return (float64(c.Covered) / float64(c.Total)) * 100
}

// Domain defines a named coverage scope and its policy.
type Domain struct {
	Name    string
	Match   []string
	Min     *float64
	Warn    *float64 // Optional warn threshold (must be >= Min)
	Exclude []string // Optional patterns to exclude from this domain
}

// Policy defines default and domain-specific coverage requirements.
type Policy struct {
	DefaultMin float64
	Domains    []Domain
}

type Status string

const (
	StatusPass Status = "PASS"
	StatusFail Status = "FAIL"
	StatusWarn Status = "WARN"
)

type DomainResult struct {
	Domain   string   `json:"domain"`
	Covered  int      `json:"covered"`
	Total    int      `json:"total"`
	Percent  float64  `json:"percent"`
	Required float64  `json:"required"`
	Status   Status   `json:"status"`
	Delta    *float64 `json:"delta,omitempty"` // Change from previous run
}

type FileRule struct {
	Match []string
	Min   float64
}

type FileResult struct {
	File     string  `json:"file"`
	Covered  int     `json:"covered"`
	Total    int     `json:"total"`
	Percent  float64 `json:"percent"`
	Required float64 `json:"required"`
	Status   Status  `json:"status"`
}

type Result struct {
	Domains  []DomainResult `json:"domains"`
	Files    []FileResult   `json:"files,omitempty"`
	Passed   bool           `json:"passed"`
	Warnings []string       `json:"warnings,omitempty"`
}

func Evaluate(policy Policy, coverage map[string]CoverageStat) Result {
	results := make([]DomainResult, 0, len(policy.Domains))
	passed := true

	for _, d := range policy.Domains {
		stat := coverage[d.Name]
		required := policy.DefaultMin
		if d.Min != nil {
			required = *d.Min
		}
		percent := round1(stat.Percent())
		status := StatusPass
		if percent < required {
			status = StatusFail
			passed = false
		} else if d.Warn != nil && percent < *d.Warn {
			// Above min but below warn threshold
			status = StatusWarn
		}
		results = append(results, DomainResult{
			Domain:   d.Name,
			Covered:  stat.Covered,
			Total:    stat.Total,
			Percent:  percent,
			Required: required,
			Status:   status,
		})
	}

	return Result{Domains: results, Passed: passed}
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
