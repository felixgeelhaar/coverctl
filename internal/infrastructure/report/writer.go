package report

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
)

type Writer struct{}

func (Writer) Write(w io.Writer, result domain.Result, format application.OutputFormat) error {
	switch format {
	case application.OutputJSON:
		payload := struct {
			Domains []domain.DomainResult `json:"domains"`
			Summary struct {
				Pass bool `json:"pass"`
			} `json:"summary"`
			Warnings []string `json:"warnings,omitempty"`
		}{
			Domains: result.Domains,
		}
		payload.Summary.Pass = result.Passed
		payload.Warnings = result.Warnings
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	case application.OutputText, "":
		return writeText(w, result)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func writeText(w io.Writer, result domain.Result) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "Domain\tCoverage\tRequired\tStatus")
	for _, d := range result.Domains {
		_, _ = fmt.Fprintf(tw, "%s\t%.1f%%\t%.1f%%\t%s\n", d.Domain, d.Percent, d.Required, d.Status)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintln(w, "\nWarnings:")
		for _, warn := range result.Warnings {
			fmt.Fprintf(w, "  - %s\n", warn)
		}
	}
	return nil
}
