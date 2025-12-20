package annotations

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/coverctl/internal/application"
)

type Scanner struct{}

const (
	maxScanLines     = 20
	pragmaIgnore     = "coverctl:ignore"
	pragmaDomainPref = "coverctl:domain="
)

func (Scanner) Scan(_ context.Context, moduleRoot string, files []string) (map[string]application.Annotation, error) {
	annotations := make(map[string]application.Annotation)
	for _, file := range files {
		if filepath.Ext(file) != ".go" {
			continue
		}
		path := file
		if moduleRoot != "" {
			path = filepath.Join(moduleRoot, filepath.FromSlash(file))
		}
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		scanner := bufio.NewScanner(f)
		lineNo := 0
		var ann application.Annotation
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if strings.Contains(line, pragmaIgnore) {
				ann.Ignore = true
			}
			if idx := strings.Index(line, pragmaDomainPref); idx != -1 {
				value := strings.TrimSpace(line[idx+len(pragmaDomainPref):])
				fields := strings.Fields(value)
				if len(fields) > 0 {
					ann.Domain = fields[0]
				}
			}
			if lineNo >= maxScanLines {
				break
			}
		}
		_ = f.Close()
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		if ann.Ignore || ann.Domain != "" {
			annotations[file] = ann
		}
	}
	return annotations, nil
}
