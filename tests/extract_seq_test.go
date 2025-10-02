package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestExtractSeqWithSeqRes(t *testing.T) {
	// Test that --seqres flag uses SEQRES records when available
	testPDB := `HEADER    TEST STRUCTURE                                    01-JAN-01   TEST
SEQRES   1 A    5  ALA GLY CYS ASP GLU
ATOM      1  CA  ALA A   1      10.000  10.000  10.000  1.00 20.00           C
ATOM      2  CA  GLY A   2      11.000  11.000  11.000  1.00 20.00           C
ATOM      3  CA  CYS A   3      12.000  12.000  12.000  1.00 20.00           C
END`

	err := os.WriteFile("test_extract_seq_seqres.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_extract_seq_seqres.pdb")

	// Test with --seqres flag (should get full SEQRES sequence)
	cmd := exec.Command("../bin/pdbtk", "extract-seq", "--seqres", "test_extract_seq_seqres.pdb")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run extract-seq with --seqres: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "AGCDE") {
		t.Errorf("Expected full SEQRES sequence 'AGCDE', got: %s", outputStr)
	}

	// Test without --seqres flag (should extract from ATOM records, only 3 residues)
	cmd2 := exec.Command("../bin/pdbtk", "extract-seq", "test_extract_seq_seqres.pdb")
	output2, err := cmd2.Output()
	if err != nil {
		t.Fatalf("Failed to run extract-seq without --seqres: %v", err)
	}

	outputStr2 := string(output2)
	if !strings.Contains(outputStr2, "AGC") || strings.Contains(outputStr2, "AGCDE") {
		t.Errorf("Expected ATOM-based sequence 'AGC' (not full SEQRES), got: %s", outputStr2)
	}
}

func TestExtractSeqWithoutSeqRes(t *testing.T) {
	// Test that default behavior falls back to ATOM records when no SEQRES
	testPDB := `HEADER    TEST STRUCTURE                                    01-JAN-01   TEST
ATOM      1  CA  ALA A   1      10.000  10.000  10.000  1.00 20.00           C
ATOM      2  CA  GLY A   2      11.000  11.000  11.000  1.00 20.00           C
ATOM      3  CA  CYS A   3      12.000  12.000  12.000  1.00 20.00           C
END`

	err := os.WriteFile("test_extract_seq_no_seqres.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_extract_seq_no_seqres.pdb")

	// Test without --seqres flag (should extract from ATOM records)
	cmd := exec.Command("../bin/pdbtk", "extract-seq", "test_extract_seq_no_seqres.pdb")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run extract-seq: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "AGC") {
		t.Errorf("Expected ATOM-based sequence 'AGC', got: %s", outputStr)
	}
}

func TestExtractSeqWithGaps(t *testing.T) {
	// Test that gaps are handled correctly when extracting from ATOM records
	testPDB := `HEADER    TEST STRUCTURE                                    01-JAN-01   TEST
ATOM      1  CA  ALA A  10      10.000  10.000  10.000  1.00 20.00           C
ATOM      2  CA  GLY A  15      15.000  15.000  15.000  1.00 20.00           C
ATOM      3  CA  CYS A  16      16.000  16.000  16.000  1.00 20.00           C
END`

	err := os.WriteFile("test_extract_seq_gaps.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_extract_seq_gaps.pdb")

	cmd := exec.Command("../bin/pdbtk", "extract-seq", "test_extract_seq_gaps.pdb")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run extract-seq: %v", err)
	}

	outputStr := string(output)
	// Should have A at position 10, then 4 gaps (11-14), then GC at 15-16
	if !strings.Contains(outputStr, "A----GC") {
		t.Errorf("Expected sequence with gaps 'A----GC', got: %s", outputStr)
	}
}

func TestExtractSeqSeqResOnlyNoOutput(t *testing.T) {
	// Test that --seqres returns nothing and prints warning when SEQRES is not available
	testPDB := `HEADER    TEST STRUCTURE                                    01-JAN-01   TEST
ATOM      1  CA  ALA A   1      10.000  10.000  10.000  1.00 20.00           C
ATOM      2  CA  GLY A   2      11.000  11.000  11.000  1.00 20.00           C
END`

	err := os.WriteFile("test_extract_seq_no_seqres_flag.pdb", []byte(testPDB), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_extract_seq_no_seqres_flag.pdb")

	// Test with --seqres flag when no SEQRES exists (should return nothing and print warning)
	cmd := exec.Command("../bin/pdbtk", "extract-seq", "--seqres", "test_extract_seq_no_seqres_flag.pdb")
	output, err := cmd.CombinedOutput() // Capture both stdout and stderr
	if err != nil {
		t.Fatalf("Failed to run extract-seq with --seqres: %v", err)
	}

	outputStr := string(output)
	// Should contain warning message
	if !strings.Contains(outputStr, "Warning") || !strings.Contains(outputStr, "no SEQRES records found") {
		t.Errorf("Expected warning message about missing SEQRES, got: %s", outputStr)
	}

	// Should have no sequence in output (other than warning)
	if strings.Contains(outputStr, ">test") {
		// If there's a FASTA header, check that there's no sequence after it
		lines := strings.Split(outputStr, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, ">") && i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if nextLine != "" && !strings.HasPrefix(nextLine, "Warning") {
					t.Errorf("Expected no sequence output with --seqres when SEQRES not available")
				}
			}
		}
	}
}

func TestExtractSeqPipedInput(t *testing.T) {
	// Test extracting from piped input without SEQRES
	testPDB := `HEADER    TEST STRUCTURE                                    01-JAN-01   TEST
ATOM      1  CA  VAL A   8      10.000  10.000  10.000  1.00 20.00           C
ATOM      2  CA  ASN A   9      11.000  11.000  11.000  1.00 20.00           C
ATOM      3  CA  THR A  10      12.000  12.000  12.000  1.00 20.00           C
END`

	cmd := exec.Command("../bin/pdbtk", "extract-seq")
	cmd.Stdin = strings.NewReader(testPDB)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run extract-seq with stdin: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "VNT") {
		t.Errorf("Expected sequence 'VNT', got: %s", outputStr)
	}
}
