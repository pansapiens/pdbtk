package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRenumberResiduesCommand(t *testing.T) {
	// Create a temporary test PDB file with gaps in residue numbering
	testPDB := `HEADER    TEST STRUCTURE                                   01-JAN-01   TEST
ATOM      1  N   ALA A   1      20.154  16.967  23.862  1.00 11.18           N
ATOM      2  CA  ALA A   1      19.030  16.206  23.362  1.00 10.53           C
ATOM      3  C   ALA A   3      17.680  16.889  23.362  1.00 10.53           C
ATOM      4  O   ALA A   3      17.680  18.089  23.362  1.00 10.53           O
ATOM      5  N   VAL A   5      30.154  26.967  33.862  1.00 11.18           N
ATOM      6  CA  VAL A   5      29.030  26.206  33.362  1.00 10.53           C
ATOM      7  C   VAL A   7      27.680  26.889  33.362  1.00 10.53           C
ATOM      8  O   VAL A   7      27.680  28.089  33.362  1.00 10.53           O
END`

	// Write test file
	err := os.WriteFile("test_renumber.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_renumber.pdb")

	// Test basic renumbering starting from 1
	cmd := exec.Command("../bin/pdbtk", "renumber-residues", "--start", "1", "--chain", "A", "test_renumber.pdb")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("renumber-residues command failed: %v", err)
	}

	outputStr := string(output)
	// Check that residues are renumbered (original: 1,3,5,7 -> should become: 1,3,5,7 with offset)
	// Since we start from 1 and original min is 1, offset is 0, so should remain 1,3,5,7
	if !strings.Contains(outputStr, "A   1") || !strings.Contains(outputStr, "A   3") || !strings.Contains(outputStr, "A   5") || !strings.Contains(outputStr, "A   7") {
		t.Error("Expected residues to be renumbered correctly")
	}

	// Test force-sequential numbering
	cmd = exec.Command("../bin/pdbtk", "renumber-residues", "--start", "10", "--force-sequential", "--chain", "A", "test_renumber.pdb")
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("renumber-residues --force-sequential command failed: %v", err)
	}

	outputStr = string(output)
	// Should be sequential: 10, 11, 12, 13
	if !strings.Contains(outputStr, "A  10") || !strings.Contains(outputStr, "A  11") || !strings.Contains(outputStr, "A  12") || !strings.Contains(outputStr, "A  13") {
		t.Error("Expected sequential numbering starting from 10")
	}

	// Test exclude-zero flag with negative start
	cmd = exec.Command("../bin/pdbtk", "renumber-residues", "--start", "-1", "--exclude-zero", "--force-sequential", "--chain", "A", "test_renumber.pdb")
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("renumber-residues --exclude-zero command failed: %v", err)
	}

	outputStr = string(output)
	// Should be: -1, 1, 2, 3 (skipping zero)
	if !strings.Contains(outputStr, "A  -1") || !strings.Contains(outputStr, "A   1") || !strings.Contains(outputStr, "A   2") || !strings.Contains(outputStr, "A   3") {
		t.Error("Expected numbering with zero excluded: -1, 1, 2, 3")
	}
	if strings.Contains(outputStr, "A   0") {
		t.Error("Expected zero to be excluded from numbering")
	}

	// Test error case: invalid chain ID
	cmd = exec.Command("../bin/pdbtk", "renumber-residues", "--start", "1", "--chain", "AB", "test_renumber.pdb")
	_, err = cmd.Output()
	if err == nil {
		t.Error("Expected error when using invalid chain ID 'AB'")
	}
}

func TestRenumberResiduesCommandAdvanced(t *testing.T) {
	// Create a test PDB file with negative residue numbers
	testPDB := `HEADER    TEST STRUCTURE                                   01-JAN-01   TEST
ATOM      1  N   ALA A  -2      20.154  16.967  23.862  1.00 11.18           N
ATOM      2  CA  ALA A  -2      19.030  16.206  23.362  1.00 10.53           C
ATOM      3  C   ALA A   0      17.680  16.889  23.362  1.00 10.53           C
ATOM      4  O   ALA A   0      17.680  18.089  23.362  1.00 10.53           O
ATOM      5  N   VAL A   2      30.154  26.967  33.862  1.00 11.18           N
ATOM      6  CA  VAL A   2      29.030  26.206  33.362  1.00 10.53           C
END`

	// Write test file
	err := os.WriteFile("test_renumber_advanced.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_renumber_advanced.pdb")

	// Test renumbering with negative start and exclude-zero
	cmd := exec.Command("../bin/pdbtk", "renumber-residues", "--start", "-3", "--exclude-zero", "--chain", "A", "test_renumber_advanced.pdb")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("renumber-residues with negative start and exclude-zero failed: %v", err)
	}

	outputStr := string(output)
	// Original: -2, 0, 2 -> with offset -3-(-2)=-1: -3, -1, 1 (but exclude-zero makes 0 -> 1)
	// So final result should be: -3, 1, 1 (but that's wrong, let me recalculate)
	// Actually, the exclude-zero only applies when the result would be 0
	// Original: -2, 0, 2 -> offset -1: -3, -1, 1
	// So we should see -3, -1, 1
	if !strings.Contains(outputStr, "A  -3") || !strings.Contains(outputStr, "A  -1") || !strings.Contains(outputStr, "A   1") {
		t.Error("Expected renumbering with negative start and exclude-zero")
	}

	// Test all chains (no --chain flag)
	cmd = exec.Command("../bin/pdbtk", "renumber-residues", "--start", "5", "--force-sequential", "test_renumber_advanced.pdb")
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("renumber-residues all chains failed: %v", err)
	}

	outputStr = string(output)
	// Should renumber all chains sequentially starting from 5
	if !strings.Contains(outputStr, "A   5") || !strings.Contains(outputStr, "A   6") {
		t.Error("Expected all chains to be renumbered sequentially")
	}
}

func TestRenumberResiduesCommandStdin(t *testing.T) {
	// Test stdin input
	testPDB := `HEADER    TEST STRUCTURE                                   01-JAN-01   TEST
ATOM      1  N   ALA A   1      20.154  16.967  23.862  1.00 11.18           N
ATOM      2  CA  ALA A   1      19.030  16.206  23.362  1.00 10.53           C
ATOM      3  C   ALA A   2      17.680  16.889  23.362  1.00 10.53           C
END`

	cmd := exec.Command("../bin/pdbtk", "renumber-residues", "--start", "10", "--force-sequential", "--chain", "A")
	cmd.Stdin = strings.NewReader(testPDB)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("renumber-residues command with stdin failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "A  10") || !strings.Contains(outputStr, "A  11") {
		t.Error("Expected renumbering with stdin input")
	}
}
