package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

const cifFixture = `data_1ABC
#
_entry.id   1ABC
#
loop_
_atom_site.group_PDB
_atom_site.id
_atom_site.type_symbol
_atom_site.label_atom_id
_atom_site.label_alt_id
_atom_site.label_comp_id
_atom_site.label_asym_id
_atom_site.label_seq_id
_atom_site.pdbx_PDB_ins_code
_atom_site.Cartn_x
_atom_site.Cartn_y
_atom_site.Cartn_z
_atom_site.occupancy
_atom_site.B_iso_or_equiv
_atom_site.auth_seq_id
_atom_site.auth_comp_id
_atom_site.auth_asym_id
_atom_site.auth_atom_id
_atom_site.pdbx_PDB_model_num
ATOM   1 N N   . ALA A 1 . 10.000 10.000 10.000 1.00 11.18 1  ALA A N   1
ATOM   2 C CA  . ALA A 1 . 11.000 10.000 10.000 1.00 10.53 1  ALA A CA  1
ATOM   3 C CA  . GLY A 2 . 12.000 10.000 10.000 1.00 12.00 2  GLY A CA  1
ATOM   4 C CA  . VAL B 1 . 20.000 20.000 20.000 1.00 15.00 1  VAL B CA  1
HETATM 5 ZN ZN  . ZN  C . . 30.000 30.000 30.000 1.00 36.00 99 ZN  A ZN  1
HETATM 6 O  O   . HOH D . . 40.000 40.000 40.000 1.00 40.00 100 HOH A O  1
#
`

func writeCIF(t *testing.T, name string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(cifFixture), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	c := exec.Command("../bin/pdbtk", args...)
	out, err := c.CombinedOutput()
	return string(out), err
}

func TestExtractFromCIFWritesCIF(t *testing.T) {
	writeCIF(t, "test_mmcif_in.cif")
	defer os.Remove("test_mmcif_in.cif")

	// With no --output and no --out-format, the input format carries through.
	out, err := run(t, "extract", "--chain", "A", "test_mmcif_in.cif")
	if err != nil {
		t.Fatalf("extract from cif: %v\n%s", err, out)
	}
	if !strings.HasPrefix(out, "data_1ABC") {
		t.Errorf("expected mmCIF output, got:\n%s", out)
	}
	if !strings.Contains(out, "_atom_site.group_PDB") {
		t.Errorf("expected an atom_site loop, got:\n%s", out)
	}
	if strings.Contains(out, "VAL") {
		t.Errorf("chain B leaked into a chain A extraction:\n%s", out)
	}
	if !strings.Contains(out, "ZN") {
		t.Errorf("chain A hetero atom should be kept by default:\n%s", out)
	}
	if strings.Contains(out, "HOH") {
		t.Errorf("waters should be dropped by default:\n%s", out)
	}
}

func TestCIFToPDBConversion(t *testing.T) {
	writeCIF(t, "test_mmcif_conv.cif")
	defer os.Remove("test_mmcif_conv.cif")

	out, err := run(t, "extract", "--chain", "A", "--out-format", "pdb", "test_mmcif_conv.cif")
	if err != nil {
		t.Fatalf("cif to pdb: %v\n%s", err, out)
	}
	if !strings.HasPrefix(out, "HEADER") {
		t.Errorf("expected PDB output, got:\n%s", out)
	}
	for _, want := range []string{
		"ATOM      1  N   ALA A   1      10.000  10.000  10.000  1.00 11.18           N",
		"HETATM    5 ZN    ZN A  99      30.000  30.000  30.000  1.00 36.00          ZN",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing expected PDB record:\n want %q\n got:\n%s", want, out)
		}
	}
}

func TestOutputFormatFollowsOutputExtension(t *testing.T) {
	writeCIF(t, "test_mmcif_ext.cif")
	defer os.Remove("test_mmcif_ext.cif")
	defer os.Remove("test_mmcif_ext_out.pdb")

	if out, err := run(t, "extract", "--chain", "A", "--output", "test_mmcif_ext_out.pdb", "test_mmcif_ext.cif"); err != nil {
		t.Fatalf("extract: %v\n%s", err, out)
	}
	written, err := os.ReadFile("test_mmcif_ext_out.pdb")
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if !strings.HasPrefix(string(written), "HEADER") {
		t.Errorf("a .pdb --output should produce PDB, got:\n%s", string(written))
	}
}

func TestPDBToCIFRoundTripIsLossless(t *testing.T) {
	testPDB := `HEADER                                                        1ABC
SEQRES   1 A    2  ALA GLY
LINK         CA  ALA A   1                ZN    ZN A  99     1555   1555  2.31
ATOM      1  N   ALA A   1      10.000  10.000  10.000  1.00 11.18           N
ATOM      2  CA  ALA A   1      11.000  10.000  10.000  1.00 10.53           C
ATOM      3  CA  GLY A   2      12.000  10.000  10.000  1.00 12.00           C
HETATM    4 ZN    ZN A  99      30.000  30.000  30.000  1.00 36.00          ZN
END
`
	if err := os.WriteFile("test_roundtrip.pdb", []byte(testPDB), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_roundtrip.pdb")
	defer os.Remove("test_roundtrip_mid.cif")

	direct, err := run(t, "extract", "--chain", "A", "test_roundtrip.pdb")
	if err != nil {
		t.Fatalf("direct extract: %v\n%s", err, direct)
	}
	if out, err := run(t, "extract", "--chain", "A", "--output", "test_roundtrip_mid.cif", "test_roundtrip.pdb"); err != nil {
		t.Fatalf("to cif: %v\n%s", err, out)
	}
	viaCIF, err := run(t, "extract", "--chain", "A", "--out-format", "pdb", "test_roundtrip_mid.cif")
	if err != nil {
		t.Fatalf("back to pdb: %v\n%s", err, viaCIF)
	}

	for _, prefix := range []string{"ATOM", "HETATM", "TER", "LINK", "SEQRES"} {
		if a, b := recordsWithPrefix(direct, prefix), recordsWithPrefix(viaCIF, prefix); a != b {
			t.Errorf("%s records changed across a PDB->CIF->PDB round trip:\n direct:\n%s\n via CIF:\n%s", prefix, a, b)
		}
	}
}

func TestStdinFormatSniffing(t *testing.T) {
	writeCIF(t, "test_mmcif_stdin.cif")
	defer os.Remove("test_mmcif_stdin.cif")

	content, err := os.ReadFile("test_mmcif_stdin.cif")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	c := exec.Command("../bin/pdbtk", "extract", "--chain", "A", "--out-format", "pdb")
	c.Stdin = strings.NewReader(string(content))
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("extract from cif on stdin: %v\n%s", err, out)
	}
	if !strings.HasPrefix(string(out), "HEADER") {
		t.Errorf("expected PDB output from sniffed CIF stdin, got:\n%s", string(out))
	}
	if !strings.Contains(string(out), "ALA A   1") {
		t.Errorf("expected chain A coordinates, got:\n%s", string(out))
	}
}

func TestLongChainIDRequiresForceLossyPDB(t *testing.T) {
	writeCIF(t, "test_mmcif_long.cif")
	defer os.Remove("test_mmcif_long.cif")

	// mmCIF happily holds a multi-character chain ID.
	out, err := run(t, "rename-chain", "A", "--to", "LONG", "test_mmcif_long.cif")
	if err != nil {
		t.Fatalf("rename to a long chain ID in cif: %v\n%s", err, out)
	}
	if !strings.Contains(out, "LONG") {
		t.Errorf("expected the renamed chain in mmCIF output:\n%s", out)
	}

	// Writing it as PDB must fail rather than silently truncating.
	out, err = run(t, "rename-chain", "A", "--to", "LONG", "--out-format", "pdb", "test_mmcif_long.cif")
	if err == nil {
		t.Fatalf("expected an error writing a long chain ID to PDB, got:\n%s", out)
	}
	if !strings.Contains(out, "force-lossy-pdb") {
		t.Errorf("error should point at --force-lossy-pdb, got:\n%s", out)
	}

	out, err = run(t, "rename-chain", "A", "--to", "LONG", "--out-format", "pdb", "--force-lossy-pdb", "test_mmcif_long.cif")
	if err != nil {
		t.Fatalf("force-lossy conversion: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Warning") || !strings.Contains(out, "LOSSY CONVERSION") {
		t.Errorf("expected a warning and a REMARK about the lossy conversion:\n%s", out)
	}
	if !strings.Contains(out, "ALA L   1") {
		t.Errorf("expected the chain truncated to its first character:\n%s", out)
	}
}

func TestExtractSeqFromCIF(t *testing.T) {
	writeCIF(t, "test_mmcif_seq.cif")
	defer os.Remove("test_mmcif_seq.cif")

	out, err := run(t, "extract-seq", "--chain", "A", "test_mmcif_seq.cif")
	if err != nil {
		t.Fatalf("extract-seq from cif: %v\n%s", err, out)
	}
	if !strings.Contains(out, ">test_mmcif_seq_A") {
		t.Errorf("expected a FASTA header for chain A, got:\n%s", out)
	}
	// The zinc and water must not appear as residues in the sequence.
	if !strings.Contains(out, "AG\n") {
		t.Errorf("expected sequence AG for chain A, got:\n%s", out)
	}
}

func TestUnknownFormatIsReported(t *testing.T) {
	if err := os.WriteFile("test_unknown.dat", []byte("not a structure file\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_unknown.dat")

	out, err := run(t, "extract", "--chain", "A", "test_unknown.dat")
	if err == nil {
		t.Fatalf("expected an error for an unrecognised file, got:\n%s", out)
	}
	if !strings.Contains(out, "in-format") {
		t.Errorf("error should mention --in-format, got:\n%s", out)
	}
}

func recordsWithPrefix(output, prefix string) string {
	var out []string
	for _, l := range strings.Split(output, "\n") {
		if strings.HasPrefix(l, prefix) {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
