package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
	"github.com/mattn/go-isatty"
)

type Writer struct{}

func (Writer) Write(w io.Writer, result domain.Result, format application.OutputFormat) error {
	switch format {
	case application.OutputJSON:
		payload := struct {
			Domains []domain.DomainResult `json:"domains"`
			Files   []domain.FileResult   `json:"files,omitempty"`
			Summary struct {
				Pass bool `json:"pass"`
			} `json:"summary"`
			Warnings []string `json:"warnings,omitempty"`
		}{
			Domains: result.Domains,
			Files:   result.Files,
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
	colorize := colorEnabled(w)
	passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#16A34A")).Bold(true)
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#DC2626")).Bold(true)
	for _, d := range result.Domains {
		status := string(d.Status)
		if colorize {
			switch d.Status {
			case domain.StatusPass:
				status = passStyle.Render(status)
			case domain.StatusFail:
				status = failStyle.Render(status)
			}
		}
		_, _ = fmt.Fprintf(tw, "%s\t%.1f%%\t%.1f%%\t%s\n", d.Domain, d.Percent, d.Required, status)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if len(result.Files) > 0 {
		fmt.Fprintln(w, "\nFile rules:")
		ftw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		_, _ = fmt.Fprintln(ftw, "File\tCoverage\tRequired\tStatus")
		colorize := colorEnabled(w)
		passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#16A34A")).Bold(true)
		failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#DC2626")).Bold(true)
		for _, f := range result.Files {
			status := string(f.Status)
			if colorize {
				switch f.Status {
				case domain.StatusPass:
					status = passStyle.Render(status)
				case domain.StatusFail:
					status = failStyle.Render(status)
				}
			}
			_, _ = fmt.Fprintf(ftw, "%s\t%.1f%%\t%.1f%%\t%s\n", f.File, f.Percent, f.Required, status)
		}
		if err := ftw.Flush(); err != nil {
			return err
		}
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintln(w, "\nWarnings:")
		for _, warn := range result.Warnings {
			fmt.Fprintf(w, "  - %s\n", warn)
		}
	}
	return nil
}

func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
}
