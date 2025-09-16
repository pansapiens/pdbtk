package cmd

import (
	"os"
	"testing"
)

func TestExtractCommand(t *testing.T) {
	// Create a temporary test PDB file
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
	err := os.WriteFile("test.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test.pdb")

	// Test file existence check
	err = checkFileExists("test.pdb")
	if err != nil {
		t.Errorf("checkFileExists failed for existing file: %v", err)
	}

	err = checkFileExists("nonexistent.pdb")
	if err == nil {
		t.Error("checkFileExists should fail for nonexistent file")
	}
}
