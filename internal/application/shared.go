package application

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/coverctl/internal/domain"
)

// loadOrDetectConfig loads config from path or auto-detects if not found.
func loadOrDetectConfig(loader ConfigLoader, detector Autodetector, configPath string) (Config, []domain.Domain, error) {
	exists, err := loader.Exists(configPath)
	if err != nil {
		return Config{}, nil, err
	}

	var cfg Config
	if !exists {
		cfg, err = detector.Detect()
		if err != nil {
			return Config{}, nil, err
		}
	} else {
		cfg, err = loader.Load(configPath)
		if err != nil {
			return Config{}, nil, err
		}
	}

	if len(cfg.Policy.Domains) == 0 {
		return Config{}, nil, fmt.Errorf("no domains configured")
	}

	return cfg, cfg.Policy.Domains, nil
}

// selectRunner returns the appropriate coverage runner based on language preference.
func selectRunner(registry RunnerRegistry, defaultRunner CoverageRunner, lang, cfgLang Language) (CoverageRunner, error) {
	effectiveLang := lang
	if effectiveLang == "" || effectiveLang == LanguageAuto {
		effectiveLang = cfgLang
	}

	if registry != nil && effectiveLang != "" && effectiveLang != LanguageAuto {
		return registry.GetRunner(effectiveLang)
	}

	if registry != nil {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		return registry.DetectRunner(wd)
	}

	if defaultRunner != nil {
		return defaultRunner, nil
	}

	return nil, fmt.Errorf("no coverage runner configured")
}

// applyDeltas calculates and applies coverage deltas from history to the result.
// This is a thin wrapper around the domain method for backward compatibility.
func applyDeltas(result *domain.Result, history domain.History) {
	result.ApplyDeltas(history)
}
