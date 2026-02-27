package testdb

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProjectRoot represents the strategy to use to find the root directory of the
// project.
type ProjectRoot int

// Supported project root strategies.
const (
	// Uses the directory of the path to the .git directory.
	GitRoot ProjectRoot = iota
	// Uses the directory of the path extracted from the GOMOD Go environment
	// variable.
	GoModule
	TestData
)

func (pr ProjectRoot) getDir() (string, error) {
	switch pr {
	case GitRoot:
		return gitRootDir()
	case GoModule:
		return goModuleDir()
	case TestData:
		return "testdata", nil
	default:
		return "", fmt.Errorf("invalid ProjectRoot %d", pr)
	}
}

func gitRootDir() (string, error) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Stdout = writer

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to find git root path: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}

func goModuleDir() (string, error) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	cmd := exec.Command("go", "env", "--json")
	cmd.Stdout = writer

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to find go mod root path: %w", err)
	}

	var goEnv struct {
		GOMOD string
	}
	if err := json.Unmarshal(buf.Bytes(), &goEnv); err != nil {
		return "", fmt.Errorf("failed to get GOMOD environment variable: %w", err)
	}

	return filepath.Dir(goEnv.GOMOD), nil
}
