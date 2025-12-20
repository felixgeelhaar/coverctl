package gotool

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ModuleResolver struct{}

func (m ModuleResolver) ModuleRoot(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "env", "GOMOD")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	gomod := strings.TrimSpace(out.String())
	if gomod == "" || gomod == os.DevNull {
		return "", errors.New("module root not found")
	}
	return filepath.Dir(gomod), nil
}
