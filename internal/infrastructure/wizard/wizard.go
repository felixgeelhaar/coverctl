package wizard

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
)

type (
	wizardState int

	initWizardModel struct {
		state      wizardState
		defaultMin float64
		domains    []wizardDomain
		cursor     int
		confirmed  bool
		aborted    bool
		exclude    []string
	}

	wizardDomain struct {
		domain   domain.Domain
		min      float64
		override bool
	}
)

const (
	stateIntro wizardState = iota
	stateEdit
	stateConfirm
)

func Run(cfg application.Config, stdout io.Writer, stdin io.Reader) (application.Config, bool, error) {
	return runInitWizard(cfg, stdout, stdin)
}

func runInitWizard(cfg application.Config, stdout io.Writer, stdin io.Reader) (application.Config, bool, error) {
	model := newInitWizardModel(cfg)
	program := tea.NewProgram(model, tea.WithInput(stdin), tea.WithOutput(stdout))
	res, err := program.Run()
	if err != nil {
		return cfg, false, err
	}
	finalModel, ok := res.(*initWizardModel)
	if !ok {
		return cfg, false, fmt.Errorf("unexpected wizard state")
	}
	if finalModel.aborted || !finalModel.confirmed {
		return cfg, false, nil
	}
	return finalModel.toConfig(), true, nil
}

func newInitWizardModel(cfg application.Config) *initWizardModel {
	defaultMin := cfg.Policy.DefaultMin
	if defaultMin <= 0 {
		defaultMin = 80
	}
	domains := make([]wizardDomain, len(cfg.Policy.Domains))
	for i, d := range cfg.Policy.Domains {
		minVal := defaultMin
		override := false
		if d.Min != nil {
			minVal = *d.Min
			override = true
		}
		domains[i] = wizardDomain{
			domain:   d,
			min:      minVal,
			override: override,
		}
	}
	if len(domains) == 0 {
		domains = append(domains, wizardDomain{domain: domain.Domain{Name: "module", Match: []string{"./..."}, Min: nil}, min: defaultMin})
	}
	return &initWizardModel{
		state:      stateIntro,
		defaultMin: defaultMin,
		domains:    domains,
		cursor:     0,
		exclude:    append([]string(nil), cfg.Exclude...),
	}
}

func (m *initWizardModel) Init() tea.Cmd {
	return nil
}

func (m *initWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.aborted = true
			return m, tea.Quit
		case "enter":
			switch m.state {
			case stateIntro:
				m.state = stateEdit
			case stateEdit:
				m.state = stateConfirm
			case stateConfirm:
				m.confirmed = true
				return m, tea.Quit
			}
		case "esc":
			if m.state == stateConfirm {
				m.state = stateEdit
			}
		case "up":
			if m.state == stateEdit {
				m.moveCursor(-1)
			}
		case "down":
			if m.state == stateEdit {
				m.moveCursor(1)
			}
		case "left", "-":
			if m.state == stateEdit {
				m.adjustSelection(-5)
			}
		case "right", "+":
			if m.state == stateEdit {
				m.adjustSelection(5)
			}
		}
	}
	return m, nil
}

func (m *initWizardModel) View() string {
	switch m.state {
	case stateIntro:
		return m.viewIntro()
	case stateEdit:
		return m.viewEdit()
	case stateConfirm:
		return m.viewConfirm()
	default:
		return ""
	}
}

func (m *initWizardModel) moveCursor(delta int) {
	max := len(m.domains)
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor > max {
		m.cursor = max
	}
}

func (m *initWizardModel) adjustSelection(delta float64) {
	if m.cursor == 0 {
		m.adjustDefault(delta)
		return
	}
	m.adjustDomain(m.cursor-1, delta)
}

func (m *initWizardModel) adjustDefault(delta float64) {
	m.defaultMin = clamp(m.defaultMin+delta, 0, 100)
	for i := range m.domains {
		if !m.domains[i].override {
			m.domains[i].min = m.defaultMin
		}
	}
}

func (m *initWizardModel) adjustDomain(index int, delta float64) {
	if index < 0 || index >= len(m.domains) {
		return
	}
	value := clamp(m.domains[index].min+delta, 0, 100)
	m.domains[index].min = value
	if !m.domains[index].override {
		m.domains[index].override = true
	}
}

func (m *initWizardModel) viewIntro() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\ncoverctl init wizard\n\n")
	fmt.Fprintf(&b, "coverctl detected %d domains. The wizard helps you review coverage thresholds.\n\n", len(m.domains))
	fmt.Fprintf(&b, "Press Enter to continue, or Ctrl+C to cancel. Default coverage is %.0f%%.\n", m.defaultMin)
	return b.String()
}

func (m *initWizardModel) viewEdit() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nReview and adjust thresholds\n\n")
	fmt.Fprintf(&b, "Use ↑/↓ to move, ←/→ or +/- to change values.\n")
	fmt.Fprintf(&b, "Default min (affects non-customized domains):\n")
	indicator := "  "
	if m.cursor == 0 {
		indicator = "> "
	}
	fmt.Fprintf(&b, "%s%.0f%%\n\n", indicator, m.defaultMin)
	fmt.Fprintf(&b, "Domains:\n")
	for idx, dom := range m.domains {
		prefix := "  "
		if m.cursor == idx+1 {
			prefix = "> "
		}
		custom := ""
		if dom.override {
			custom = " (custom)"
		}
		fmt.Fprintf(&b, "%s%s: %.0f%%%s\n", prefix, dom.domain.Name, dom.min, custom)
	}
	fmt.Fprintf(&b, "\nEnter to continue, q to cancel.\n")
	return b.String()
}

func (m *initWizardModel) viewConfirm() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nReady to write configuration\n\n")
	fmt.Fprintf(&b, "Default min coverage: %.0f%%\n", m.defaultMin)
	fmt.Fprintf(&b, "Domains summary:\n")
	for _, dom := range m.domains {
		fmt.Fprintf(&b, "  %s: %.0f%%\n", dom.domain.Name, dom.min)
	}
	if len(m.exclude) > 0 {
		fmt.Fprintf(&b, "\nConfigured exclusions:\n")
		for _, pattern := range m.exclude {
			fmt.Fprintf(&b, "  - %s\n", pattern)
		}
	} else {
		fmt.Fprintf(&b, "\nNo exclusions configured.\n")
	}
	fmt.Fprintf(&b, "\nPress Enter to save, Esc to go back, q to cancel.\n")
	return b.String()
}

func (m *initWizardModel) toConfig() application.Config {
	cfg := application.Config{
		Policy: domain.Policy{
			DefaultMin: m.defaultMin,
			Domains:    make([]domain.Domain, len(m.domains)),
		},
		Exclude: append([]string(nil), m.exclude...),
	}
	for i, dom := range m.domains {
		d := dom.domain
		d.Min = nil
		if dom.override {
			min := dom.min
			d.Min = &min
		}
		cfg.Policy.Domains[i] = d
	}
	return cfg
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
