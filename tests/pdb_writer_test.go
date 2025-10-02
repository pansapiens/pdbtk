package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/perry/pdbtk/pdbtk/cmd"
)

func TestExtractAltLocFromAtomName(t *testing.T) {
	tests := []struct {
		input    string
		expected byte
	}{
		{"CA A", 'A'},
		{"CB B", 'B'},
		{"N  A", 'A'},
		{"CA", ' '},
		{"CB", ' '},
		{"N", ' '},
		{"CA Z", 'Z'},
		{"", ' '},
		{"   ", ' '},
	}

	for _, test := range tests {
		result := cmd.ExtractAltLocFromAtomName(test.input)
		if result != test.expected {
			t.Errorf("extractAltLocFromAtomName(%q) = %c, expected %c", test.input, result, test.expected)
		}
	}
}

func TestRemoveAltLocFromAtomName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CA A", "CA"},
		{"CB B", "CB"},
		{"N  A", "N"},
		{"CA", "CA"},
		{"CB", "CB"},
		{"N", "N"},
		{"CA Z", "CA"},
		{"", ""},
		{"   ", ""},
	}

	for _, test := range tests {
		result := cmd.RemoveAltLocFromAtomName(test.input)
		if result != test.expected {
			t.Errorf("removeAltLocFromAtomName(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestAltLocPreservation(t *testing.T) {
	// Create a test PDB file with ALTLOC fields
	testPDB := `HEADER    TEST STRUCTURE WITH ALTLOC                       01-JAN-01   TEST
ATOM    812  CA ACYS B  37      14.459  21.739  -3.403  0.63 29.15           C  
ATOM    813  CA BCYS B  37      14.407  21.835  -3.480  0.37 28.44           C  
ATOM    814  CB ACYS B  37      15.123  22.456  -2.234  0.63 28.15           C  
ATOM    815  CB BCYS B  37      15.089  22.567  -2.301  0.37 27.44           C  
END`

	// Write test file
	err := os.WriteFile("test_altloc_preservation.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_altloc_preservation.pdb")

	// Test extracting chain B
	cmd := exec.Command("../bin/pdbtk", "extract", "--chains", "B", "test_altloc_preservation.pdb")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run extract command: %v", err)
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	// Check that ALTLOC fields are preserved
	altLocCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "ATOM") {
			// Check if ALTLOC field (column 17) is preserved
			if len(line) >= 17 {
				altLoc := line[16] // Column 17 (0-indexed)
				if altLoc == 'A' || altLoc == 'B' {
					altLocCount++
				}
			}
		}
	}

	// We should have 4 atoms with ALTLOC fields (2 A's and 2 B's)
	if altLocCount != 4 {
		t.Errorf("Expected 4 atoms with ALTLOC fields, got %d", altLocCount)
	}

	// Check specific ALTLOC preservation
	expectedAltLocs := []byte{'A', 'B', 'A', 'B'}
	atomLines := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "ATOM") {
			if len(line) >= 17 {
				altLoc := line[16]
				if atomLines < len(expectedAltLocs) {
					if altLoc != expectedAltLocs[atomLines] {
						t.Errorf("Atom %d: expected ALTLOC %c, got %c", atomLines+1, expectedAltLocs[atomLines], altLoc)
					}
				}
				atomLines++
			}
		}
	}
}

func TestAltLocFiltering(t *testing.T) {
	// Create a test PDB file with ALTLOC fields
	testPDB := `HEADER    TEST STRUCTURE WITH ALTLOC                       01-JAN-01   TEST
ATOM    812  CA ACYS B  37      14.459  21.739  -3.403  0.63 29.15           C  
ATOM    813  CA BCYS B  37      14.407  21.835  -3.480  0.37 28.44           C  
ATOM    814  CB ACYS B  37      15.123  22.456  -2.234  0.63 28.15           C  
ATOM    815  CB BCYS B  37      15.089  22.567  -2.301  0.37 27.44           C  
ATOM    816  N   CYS B  38      16.000  23.000  -1.000  1.00 25.00           N  
END`

	// Write test file
	err := os.WriteFile("test_altloc_filtering.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_altloc_filtering.pdb")

	// Test cases
	testCases := []struct {
		name          string
		altlocFlag    string
		expectedAtoms int
		description   string
	}{
		{
			name:          "No ALTLOC filter - all atoms",
			altlocFlag:    "",
			expectedAtoms: 5,
			description:   "Should preserve all atoms including duplicates",
		},
		{
			name:          "ALTLOC A only",
			altlocFlag:    "A",
			expectedAtoms: 3,
			description:   "Should keep ALTLOC A atoms and atoms without ALTLOC",
		},
		{
			name:          "ALTLOC B only",
			altlocFlag:    "B",
			expectedAtoms: 3,
			description:   "Should keep ALTLOC B atoms and atoms without ALTLOC",
		},
		{
			name:          "ALTLOC first - takes first when duplicates exist",
			altlocFlag:    "first",
			expectedAtoms: 3,
			description:   "Should take first ALTLOC when duplicates exist, keep atoms without ALTLOC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build command
			args := []string{"extract", "--chains", "B"}
			if tc.altlocFlag != "" {
				args = append(args, "--altloc", tc.altlocFlag)
			}
			args = append(args, "test_altloc_filtering.pdb")

			cmd := exec.Command("../bin/pdbtk", args...)
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("Failed to run extract command: %v", err)
			}

			outputStr := string(output)
			lines := strings.Split(outputStr, "\n")

			// Count atoms
			atomCount := 0
			for _, line := range lines {
				if strings.HasPrefix(line, "ATOM") {
					atomCount++
				}
			}

			// Check atom count
			if atomCount != tc.expectedAtoms {
				t.Errorf("%s: Expected %d atoms, got %d", tc.description, tc.expectedAtoms, atomCount)
			}
		})
	}
}

func TestAltLocFilteringWithStdin(t *testing.T) {
	// Test ALTLOC filtering with stdin input
	testPDB := `HEADER    TEST STRUCTURE WITH ALTLOC                       01-JAN-01   TEST
ATOM    812  CA ACYS B  37      14.459  21.739  -3.403  0.63 29.15           C  
ATOM    813  CA BCYS B  37      14.407  21.835  -3.480  0.37 28.44           C  
END`

	cmd := exec.Command("../bin/pdbtk", "extract", "--chains", "B", "--altloc", "A")
	cmd.Stdin = strings.NewReader(testPDB)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run extract command with stdin: %v", err)
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	// Should have only 1 atom (CA with ALTLOC A)
	atomCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "ATOM") {
			atomCount++
			if len(line) >= 17 {
				altLoc := line[16]
				if altLoc != 'A' {
					t.Errorf("Expected ALTLOC A, got %c", altLoc)
				}
			}
		}
	}

	if atomCount != 1 {
		t.Errorf("Expected 1 atom, got %d", atomCount)
	}
}

func TestAltLocPreservationWithChainExtraction(t *testing.T) {
	// Test that ALTLOC is preserved when extracting specific chains
	testPDB := `HEADER    TEST STRUCTURE WITH ALTLOC                       01-JAN-01   TEST
ATOM    100  CA  ALA A  10      10.000  10.000  10.000  1.00 20.00           C  
ATOM    200  CA ACYS B  20      20.000  20.000  20.000  0.63 29.15           C  
ATOM    201  CA BCYS B  20      20.100  20.100  20.100  0.37 28.44           C  
ATOM    202  CB ACYS B  20      21.000  21.000  21.000  0.63 28.15           C  
ATOM    203  CB BCYS B  20      21.100  21.100  21.100  0.37 27.44           C  
END`

	err := os.WriteFile("test_altloc_chain_extraction.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_altloc_chain_extraction.pdb")

	cmd := exec.Command("../bin/pdbtk", "extract", "--chains", "B", "test_altloc_chain_extraction.pdb")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run extract command: %v", err)
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	// Verify ALTLOC is preserved for chain B atoms
	altlocACount := 0
	altlocBCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "ATOM") {
			if len(line) >= 17 {
				altLoc := line[16]
				if altLoc == 'A' {
					altlocACount++
				} else if altLoc == 'B' {
					altlocBCount++
				}
			}
		}
	}

	// Should have 2 atoms with ALTLOC A and 2 with ALTLOC B
	if altlocACount != 2 {
		t.Errorf("Expected 2 atoms with ALTLOC A, got %d", altlocACount)
	}
	if altlocBCount != 2 {
		t.Errorf("Expected 2 atoms with ALTLOC B, got %d", altlocBCount)
	}
}

func TestAltLocFilteringWithoutChains(t *testing.T) {
	// Test --altloc without --chains flag
	testPDB := `HEADER    TEST STRUCTURE WITH ALTLOC                       01-JAN-01   TEST
ATOM    100  CA AALA A  10      10.000  10.000  10.000  0.50 20.00           C  
ATOM    101  CA BALA A  10      10.100  10.100  10.100  0.50 20.00           C  
ATOM    200  CA ACYS B  20      20.000  20.000  20.000  0.63 29.15           C  
ATOM    201  CA BCYS B  20      20.100  20.100  20.100  0.37 28.44           C  
END`

	err := os.WriteFile("test_altloc_no_chains.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_altloc_no_chains.pdb")

	// Test --altloc A without --chains (should filter all chains)
	cmd := exec.Command("../bin/pdbtk", "extract", "--altloc", "A", "test_altloc_no_chains.pdb")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run extract command: %v", err)
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	// Count atoms and verify they all have ALTLOC A
	atomCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "ATOM") {
			atomCount++
			if len(line) >= 17 {
				altLoc := line[16]
				if altLoc != 'A' {
					t.Errorf("Expected ALTLOC A, got %c in line: %s", altLoc, line)
				}
			}
		}
	}

	// Should have 2 atoms (CA A from chain A and CA A from chain B)
	if atomCount != 2 {
		t.Errorf("Expected 2 atoms, got %d", atomCount)
	}
}

func TestAltLocFirstWithMultipleResidues(t *testing.T) {
	// Test --altloc first with multiple residues having ALTLOC
	testPDB := `HEADER    TEST STRUCTURE WITH ALTLOC                       01-JAN-01   TEST
ATOM    200  CA ACYS B  20      20.000  20.000  20.000  0.63 29.15           C  
ATOM    201  CA BCYS B  20      20.100  20.100  20.100  0.37 28.44           C  
ATOM    202  CB ACYS B  20      21.000  21.000  21.000  0.63 28.15           C  
ATOM    203  CB BCYS B  20      21.100  21.100  21.100  0.37 27.44           C  
ATOM    204  N   CYS B  21      22.000  22.000  22.000  1.00 25.00           N  
ATOM    300  CA ASER C  30      30.000  30.000  30.000  0.70 30.00           C  
ATOM    301  CA BSER C  30      30.100  30.100  30.100  0.30 30.00           C  
END`

	err := os.WriteFile("test_altloc_first_multi.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_altloc_first_multi.pdb")

	cmd := exec.Command("../bin/pdbtk", "extract", "--altloc", "first", "test_altloc_first_multi.pdb")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run extract command: %v", err)
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	// Should have 4 atoms: CA A (res 20), CB A (res 20), N (res 21), CA A (res 30)
	atomCount := 0
	altlocACount := 0
	noAltlocCount := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "ATOM") {
			atomCount++
			if len(line) >= 17 {
				altLoc := line[16]
				if altLoc == 'A' {
					altlocACount++
				} else if altLoc == ' ' {
					noAltlocCount++
				} else {
					t.Errorf("Unexpected ALTLOC %c in line: %s", altLoc, line)
				}
			}
		}
	}

	if atomCount != 4 {
		t.Errorf("Expected 4 atoms, got %d", atomCount)
	}
	if altlocACount != 3 {
		t.Errorf("Expected 3 atoms with ALTLOC A, got %d", altlocACount)
	}
	if noAltlocCount != 1 {
		t.Errorf("Expected 1 atom without ALTLOC, got %d", noAltlocCount)
	}
}
