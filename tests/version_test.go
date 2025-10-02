package tests

import (
	"os/exec"
	"strings"
	"testing"

	pdbtkcmd "github.com/perry/pdbtk/pdbtk/cmd"
)

func TestVersionCommand(t *testing.T) {
	// Test version command
	cmd := exec.Command("../bin/pdbtk", "version")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	outputStr := strings.TrimSpace(string(output))
	expectedVersion := pdbtkcmd.Version
	if outputStr != expectedVersion {
		t.Errorf("Expected version '%s', got '%s'", expectedVersion, outputStr)
	}
}

func TestVersionCommandHelp(t *testing.T) {
	// Test version command help
	cmd := exec.Command("../bin/pdbtk", "version", "--help")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("version --help command failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Print the version number") {
		t.Error("Expected help text to contain 'Print the version number'")
	}
	if !strings.Contains(outputStr, "pdbtk version") {
		t.Error("Expected help text to contain 'pdbtk version'")
	}
}
