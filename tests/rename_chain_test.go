package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRenameChainCommand(t *testing.T) {
	// Create a temporary test PDB file with multiple chains
	testPDB := `HEADER    TEST STRUCTURE                                   01-JAN-01   TEST
ATOM      1  N   ALA A   1      20.154  16.967  23.862  1.00 11.18           N
ATOM      2  CA  ALA A   1      19.030  16.206  23.362  1.00 10.53           C
ATOM      3  C   ALA A   1      17.680  16.889  23.362  1.00 10.53           C
ATOM      4  O   ALA A   1      17.680  18.089  23.362  1.00 10.53           O
ATOM      5  N   VAL B   1      30.154  26.967  33.862  1.00 11.18           N
ATOM      6  CA  VAL B   1      29.030  26.206  33.362  1.00 10.53           C
ATOM      7  C   VAL B   1      27.680  26.889  33.362  1.00 10.53           C
ATOM      8  O   VAL B   1      27.680  28.089  33.362  1.00 10.53           O
END`

	// Write test file
	err := os.WriteFile("test_rename.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_rename.pdb")

	// Test successful chain rename
	cmd := exec.Command("../bin/pdbtk", "rename-chain", "A", "--to", "X", "test_rename.pdb")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("rename-chain command failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "X   1") {
		t.Error("Expected chain A to be renamed to X, but output doesn't contain 'X   1'")
	}
	if strings.Contains(outputStr, "A   1") {
		t.Error("Expected chain A to be renamed, but output still contains 'A   1'")
	}

	// Test error case: non-existent chain
	cmd = exec.Command("../bin/pdbtk", "rename-chain", "Z", "--to", "Y", "test_rename.pdb")
	_, err = cmd.Output()
	if err == nil {
		t.Error("Expected error when trying to rename non-existent chain Z")
	}

	// Test error case: invalid chain ID
	cmd = exec.Command("../bin/pdbtk", "rename-chain", "AB", "--to", "C", "test_rename.pdb")
	_, err = cmd.Output()
	if err == nil {
		t.Error("Expected error when using invalid chain ID 'AB'")
	}

	// Test error case: invalid new chain ID
	cmd = exec.Command("../bin/pdbtk", "rename-chain", "A", "--to", "CD", "test_rename.pdb")
	_, err = cmd.Output()
	if err == nil {
		t.Error("Expected error when using invalid new chain ID 'CD'")
	}

	// Test warning case: new chain already exists
	cmd = exec.Command("../bin/pdbtk", "rename-chain", "A", "--to", "B", "test_rename.pdb")
	_, err = cmd.Output()
	if err != nil {
		t.Fatalf("rename-chain command failed when renaming to existing chain: %v", err)
	}
	// Note: The warning is printed to stderr, so we can't easily test it here
	// but the command should succeed
}

func TestRenameChainCommandStdin(t *testing.T) {
	// Test stdin input
	testPDB := `HEADER    TEST STRUCTURE                                   01-JAN-01   TEST
ATOM      1  N   ALA A   1      20.154  16.967  23.862  1.00 11.18           N
ATOM      2  CA  ALA A   1      19.030  16.206  23.362  1.00 10.53           C
END`

	cmd := exec.Command("../bin/pdbtk", "rename-chain", "A", "--to", "X")
	cmd.Stdin = strings.NewReader(testPDB)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("rename-chain command with stdin failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "X   1") {
		t.Error("Expected chain A to be renamed to X with stdin input")
	}
}
