package structure

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const samplePDB = `HEADER                                                        1ABC
SEQRES   1 A    3  ALA GLY CYS
MODRES 1ABC MSE A    2  MET  SELENOMETHIONINE
LINK         SG  CYS A   3                ZN    ZN A  99     1555   1555  2.31
ATOM      1  N   ALA A   1      10.000  10.000  10.000  1.00 11.18           N
ATOM      2  CA AALA A   1      11.000  10.000  10.000  0.60 10.53           C
ATOM      3  CA BALA A   1      11.100  10.100  10.100  0.40 12.53           C
ATOM      4 SE   MSE A   2      12.000  10.000  10.000  1.00 20.00          SE
ATOM      5  SG  CYS A   3      13.000  10.000  10.000  1.00 21.00           S
HETATM    6 ZN    ZN A  99      20.000  20.000  20.000  1.00 36.00          ZN
HETATM    7  O   HOH A 100      40.000  40.000  40.000  1.00 36.00           O
END
`

func mustReadPDB(t *testing.T, s string) *Structure {
	t.Helper()
	st, err := ReadPDB([]byte(s))
	if err != nil {
		t.Fatalf("ReadPDB: %v", err)
	}
	return st
}

func TestReadPDBKeepsEverything(t *testing.T) {
	s := mustReadPDB(t, samplePDB)

	if s.ID != "1ABC" {
		t.Errorf("ID = %q, want 1ABC", s.ID)
	}
	if len(s.Atoms) != 7 {
		t.Fatalf("kept %d atoms, want 7 (waters and post-TER hetero must survive parsing)", len(s.Atoms))
	}
	if len(s.Links) != 1 {
		t.Fatalf("parsed %d LINK records, want 1", len(s.Links))
	}
	if len(s.SeqRes) != 1 || len(s.SeqRes[0].Residues) != 3 {
		t.Fatalf("SeqRes = %+v, want one chain of 3 residues", s.SeqRes)
	}
	if s.ModRes["MSE"] != "MET" {
		t.Errorf("MODRES mapping = %v, want MSE->MET", s.ModRes)
	}

	// Two-character elements must not be mistaken for alpha carbons.
	if got := s.Atoms[3].Element; got != "SE" {
		t.Errorf("MSE selenium element = %q, want SE", got)
	}
	if got := s.Atoms[5].Element; got != "ZN" {
		t.Errorf("zinc element = %q, want ZN", got)
	}
	if s.Atoms[1].AltLoc != "A" || s.Atoms[2].AltLoc != "B" {
		t.Errorf("altLocs = %q/%q, want A/B", s.Atoms[1].AltLoc, s.Atoms[2].AltLoc)
	}
	if s.Atoms[1].Occupancy != 0.60 || s.Atoms[1].BFactor != 10.53 {
		t.Errorf("occupancy/B = %v/%v, want 0.60/10.53", s.Atoms[1].Occupancy, s.Atoms[1].BFactor)
	}
	if !s.Atoms[6].IsWater() {
		t.Error("HOH residue not recognised as water")
	}
}

func TestPDBRoundTripPreservesColumns(t *testing.T) {
	s := mustReadPDB(t, samplePDB)
	var buf bytes.Buffer
	if err := WritePDB(&buf, s, WriteOptions{Version: "test", CommandLine: "test"}); err != nil {
		t.Fatalf("WritePDB: %v", err)
	}

	want := coordinateLines(samplePDB)
	got := coordinateLines(buf.String())
	if len(want) != len(got) {
		t.Fatalf("wrote %d coordinate lines, want %d", len(got), len(want))
	}
	for i := range want {
		// Columns 13-66 span atom name through B-factor: the fields that were
		// corrupted before the format-neutral model was introduced.
		if want[i][12:66] != got[i][12:66] {
			t.Errorf("line %d columns 13-66 differ:\n want %q\n got  %q", i+1, want[i][12:66], got[i][12:66])
		}
	}
}

func TestTERClosesPolymerBeforeHetero(t *testing.T) {
	s := mustReadPDB(t, samplePDB)
	var buf bytes.Buffer
	if err := WritePDB(&buf, s, WriteOptions{Version: "test", CommandLine: "test"}); err != nil {
		t.Fatalf("WritePDB: %v", err)
	}

	var order []string
	for _, l := range strings.Split(buf.String(), "\n") {
		switch {
		case strings.HasPrefix(l, "TER"):
			order = append(order, "TER")
		case strings.HasPrefix(l, "HETATM"):
			order = append(order, "HETATM")
		}
	}
	// TER must terminate the polymer chain before the zinc and water records.
	want := []string{"TER", "HETATM", "HETATM"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Errorf("record order = %v, want %v", order, want)
	}
}

func TestCIFRoundTripPreservesAtoms(t *testing.T) {
	s := mustReadPDB(t, samplePDB)

	var cif bytes.Buffer
	if err := WriteCIF(&cif, s, WriteOptions{Version: "test", CommandLine: "test"}); err != nil {
		t.Fatalf("WriteCIF: %v", err)
	}
	back, err := ReadCIF(cif.Bytes())
	if err != nil {
		t.Fatalf("ReadCIF: %v\n%s", err, cif.String())
	}

	if back.ID != s.ID {
		t.Errorf("ID %q -> %q", s.ID, back.ID)
	}
	if len(back.Atoms) != len(s.Atoms) {
		t.Fatalf("atom count %d -> %d", len(s.Atoms), len(back.Atoms))
	}
	for i := range s.Atoms {
		a, b := s.Atoms[i], back.Atoms[i]
		if a.Name != b.Name || a.ResName != b.ResName || a.ChainID != b.ChainID ||
			a.ResSeq != b.ResSeq || a.AltLoc != b.AltLoc || a.Element != b.Element ||
			a.Hetatm != b.Hetatm || a.X != b.X || a.Y != b.Y || a.Z != b.Z ||
			a.Occupancy != b.Occupancy || a.BFactor != b.BFactor {
			t.Fatalf("atom %d changed:\n in  %+v\n out %+v", i, *a, *b)
		}
	}
	if len(back.Links) != 1 || back.Links[0].A.SymOp != "1555" {
		t.Errorf("LINK symmetry operators lost: %+v", back.Links)
	}
	if len(back.SeqRes) != 1 || len(back.SeqRes[0].Residues) != 3 {
		t.Errorf("SeqRes lost: %+v", back.SeqRes)
	}
}

func TestFitPDBRejectsOversizedFields(t *testing.T) {
	s := mustReadPDB(t, samplePDB)
	s.Atoms[0].ChainID = "AAA"
	s.Atoms[1].ResName = "ABCDE"

	problems := FitPDB(s)
	if len(problems) != 2 {
		t.Fatalf("FitPDB reported %d problems, want 2: %v", len(problems), problems)
	}

	if err := WritePDB(&bytes.Buffer{}, s, WriteOptions{}); err == nil {
		t.Error("WritePDB accepted a structure that does not fit the PDB format")
	}
	if err := WritePDB(&bytes.Buffer{}, s, WriteOptions{ForceLossy: true}); err != nil {
		t.Errorf("WritePDB with ForceLossy: %v", err)
	}
}

func TestFormatDetection(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		content string
		want    Format
	}{
		{"pdb extension", "x.pdb", "", FormatPDB},
		{"cif extension", "x.cif", "", FormatCIF},
		{"gzipped cif", "x.cif.gz", "", FormatCIF},
		{"unknown extension", "x.dat", "", FormatUnknown},
		{"pdb content", "", samplePDB, FormatPDB},
		{"cif content", "", "# comment\ndata_1ABC\n_entry.id 1ABC\n", FormatCIF},
		{"empty content", "", "\n\n", FormatUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got Format
			if tc.path != "" {
				got = FormatFromPath(tc.path)
			} else {
				got = FormatFromContent([]byte(tc.content))
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHybrid36(t *testing.T) {
	// Boundary values from the wwPDB hybrid-36 specification.
	cases := []struct {
		value int
		width int
		text  string
	}{
		{1, 5, "    1"},
		{99999, 5, "99999"},
		{100000, 5, "A0000"},
		{43770015, 5, "ZZZZZ"},
		{43770016, 5, "a0000"},
		{87440031, 5, "zzzzz"},
		{9999, 4, "9999"},
		{10000, 4, "A000"},
	}
	for _, tc := range cases {
		got, ok := encodeHybrid36(tc.value, tc.width)
		if !ok || got != tc.text {
			t.Errorf("encodeHybrid36(%d, %d) = %q/%v, want %q", tc.value, tc.width, got, ok, tc.text)
		}
		back, err := decodeHybrid36(tc.text, tc.width)
		if err != nil || back != tc.value {
			t.Errorf("decodeHybrid36(%q, %d) = %d/%v, want %d", tc.text, tc.width, back, err, tc.value)
		}
	}

	if n, err := decodeHybrid36("-10", 4); err != nil || n != -10 {
		t.Errorf("negative residue numbers must decode: got %d/%v", n, err)
	}
}

func TestCIFQuoting(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "."},
		{".", "."},
		{"ALA", "ALA"},
		{"has space", "'has space'"},
		{"it's", `"it's"`},
		{"data_x", "'data_x'"},
		{"_leading", "'_leading'"},
	}
	for _, tc := range cases {
		if got := cifQuote(tc.in); got != tc.want {
			t.Errorf("cifQuote(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCIFParsesQuotingAndTextFields(t *testing.T) {
	const src = `data_TEST
_entry.id TEST
_struct.title
;A title that spans
two lines
;
loop_
_atom_site.group_PDB
_atom_site.id
_atom_site.type_symbol
_atom_site.label_atom_id
_atom_site.label_comp_id
_atom_site.label_asym_id
_atom_site.Cartn_x
_atom_site.Cartn_y
_atom_site.Cartn_z
_atom_site.auth_seq_id
ATOM 1 C "CA'" ALA A 1.000 2.000 3.000 1
ATOM 2 N 'N  ' ALA A 4.000 5.000 6.000 1
#
`
	s, err := ReadCIF([]byte(src))
	if err != nil {
		t.Fatalf("ReadCIF: %v", err)
	}
	if len(s.Atoms) != 2 {
		t.Fatalf("parsed %d atoms, want 2", len(s.Atoms))
	}
	if s.Atoms[0].Name != "CA'" {
		t.Errorf("double-quoted name = %q, want CA'", s.Atoms[0].Name)
	}
	if s.Atoms[1].Name != "N  " {
		t.Errorf("single-quoted name = %q, want %q", s.Atoms[1].Name, "N  ")
	}
	if s.Atoms[1].Z != 6.0 {
		t.Errorf("Cartn_z = %v, want 6", s.Atoms[1].Z)
	}
}

func TestCIFRejectsRaggedLoop(t *testing.T) {
	const src = `data_TEST
loop_
_atom_site.group_PDB
_atom_site.id
ATOM 1
ATOM
`
	if _, err := ReadCIF([]byte(src)); err == nil {
		t.Error("expected an error for a loop_ ending mid-row")
	}
}

const multiModelPDB = `HEADER                                                        2NMR
MODEL        1
ATOM      1  N   ALA A   1      10.000  10.000  10.000  1.00 11.18           N
ATOM      2  CA  ALA A   1      11.000  10.000  10.000  1.00 10.53           C
ENDMDL
MODEL        2
ATOM      1  N   ALA A   1      10.500  10.500  10.500  1.00 11.18           N
ATOM      2  CA  ALA A   1      11.500  10.500  10.500  1.00 10.53           C
ENDMDL
MODEL        3
ATOM      1  N   ALA A   1      10.900  10.900  10.900  1.00 11.18           N
ATOM      2  CA  ALA A   1      11.900  10.900  10.900  1.00 10.53           C
ENDMDL
END
`

// NMR ensembles carry the model number on every atom; mmCIF keeps it in a
// column while PDB brackets each model in MODEL/ENDMDL.
func TestMultiModelSurvivesBothFormats(t *testing.T) {
	s := mustReadPDB(t, multiModelPDB)
	if len(s.Atoms) != 6 || !s.HasMultipleModels() {
		t.Fatalf("parsed %d atoms across models=%v, want 6 in 3 models", len(s.Atoms), s.HasMultipleModels())
	}
	if s.Atoms[0].Model != 1 || s.Atoms[2].Model != 2 || s.Atoms[4].Model != 3 {
		t.Fatalf("model numbers = %d/%d/%d, want 1/2/3",
			s.Atoms[0].Model, s.Atoms[2].Model, s.Atoms[4].Model)
	}

	var cif bytes.Buffer
	if err := WriteCIF(&cif, s, WriteOptions{Version: "test", CommandLine: "test"}); err != nil {
		t.Fatalf("WriteCIF: %v", err)
	}
	viaCIF, err := ReadCIF(cif.Bytes())
	if err != nil {
		t.Fatalf("ReadCIF: %v", err)
	}
	for i := range s.Atoms {
		if a, b := s.Atoms[i], viaCIF.Atoms[i]; a.Model != b.Model || a.X != b.X {
			t.Fatalf("atom %d changed through mmCIF: model %d->%d, x %v->%v",
				i, a.Model, b.Model, a.X, b.X)
		}
	}

	var pdb bytes.Buffer
	if err := WritePDB(&pdb, viaCIF, WriteOptions{Version: "test", CommandLine: "test"}); err != nil {
		t.Fatalf("WritePDB: %v", err)
	}
	got := recordOrder(pdb.String(), "MODEL", "ENDMDL")
	want := "MODEL,ENDMDL,MODEL,ENDMDL,MODEL,ENDMDL"
	if got != want {
		t.Errorf("model bracketing = %s, want %s", got, want)
	}
	if !strings.Contains(pdb.String(), "MODEL        2") {
		t.Errorf("model numbers not preserved:\n%s", pdb.String())
	}
}

// Insertion codes and out-of-range residue numbers are where fixed-column PDB
// and free-form mmCIF disagree most.
func TestInsertionCodesAndResidueNumbering(t *testing.T) {
	const src = `HEADER                                                        1ABC
ATOM      1  N   ALA A 100      10.000  10.000  10.000  1.00 11.18           N
ATOM      2  N   GLY A 100A     11.000  10.000  10.000  1.00 11.18           N
ATOM      3  N   SER A 100B     12.000  10.000  10.000  1.00 11.18           N
ATOM      4 HG11 VAL A  -5      13.000  10.000  10.000  1.00 11.18           H
END
`
	s := mustReadPDB(t, src)
	if s.Atoms[1].InsCode != "A" || s.Atoms[2].InsCode != "B" {
		t.Fatalf("insertion codes = %q/%q, want A/B", s.Atoms[1].InsCode, s.Atoms[2].InsCode)
	}
	if s.Atoms[0].ResSeq != 100 || s.Atoms[1].ResSeq != 100 {
		t.Fatalf("residue numbers = %d/%d, want both 100", s.Atoms[0].ResSeq, s.Atoms[1].ResSeq)
	}
	if s.Atoms[3].ResSeq != -5 {
		t.Errorf("negative residue number = %d, want -5", s.Atoms[3].ResSeq)
	}
	// Residues differing only by insertion code must not be merged.
	if got := len(Residues(s.Atoms)); got != 4 {
		t.Errorf("grouped into %d residues, want 4", got)
	}
	// A four-character atom name fills columns 13-16 with no leading space.
	if s.Atoms[3].Name != "HG11" {
		t.Errorf("four-character atom name = %q, want HG11", s.Atoms[3].Name)
	}

	var cif bytes.Buffer
	if err := WriteCIF(&cif, s, WriteOptions{Version: "test", CommandLine: "test"}); err != nil {
		t.Fatalf("WriteCIF: %v", err)
	}
	back, err := ReadCIF(cif.Bytes())
	if err != nil {
		t.Fatalf("ReadCIF: %v", err)
	}
	for i := range s.Atoms {
		a, b := s.Atoms[i], back.Atoms[i]
		if a.InsCode != b.InsCode || a.ResSeq != b.ResSeq || a.Name != b.Name {
			t.Errorf("atom %d changed through mmCIF:\n in  %+v\n out %+v", i, *a, *b)
		}
	}
}

// A LINK to an atom that has been filtered away would name a residue the file
// no longer contains, and SEQRES for a dropped chain is equally dangling.
func TestFilterAtomsDropsOrphanedRecords(t *testing.T) {
	s := mustReadPDB(t, samplePDB)
	if len(s.Links) != 1 || len(s.SeqRes) != 1 {
		t.Fatalf("fixture should start with one LINK and one SEQRES chain")
	}

	// The LINK joins CYS A 3 to ZN A 99; dropping the zinc orphans it.
	noHetero := s.FilterAtoms(func(a *Atom) bool { return !a.Hetatm })
	if len(noHetero.Links) != 0 {
		t.Errorf("kept a LINK whose partner was filtered out: %+v", noHetero.Links)
	}
	if len(noHetero.SeqRes) != 1 {
		t.Errorf("dropped SEQRES for a chain that still has atoms: %+v", noHetero.SeqRes)
	}

	nothing := s.FilterAtoms(func(a *Atom) bool { return false })
	if len(nothing.SeqRes) != 0 || len(nothing.Links) != 0 {
		t.Errorf("SEQRES/LINK survived removal of every atom: %+v / %+v",
			nothing.SeqRes, nothing.Links)
	}

	// Filtering must not disturb the structure it was called on.
	if len(s.Atoms) != 7 || len(s.Links) != 1 {
		t.Errorf("FilterAtoms mutated its receiver: %d atoms, %d links", len(s.Atoms), len(s.Links))
	}
}

func TestReadFileDecompressesGzip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdb.gz")

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(samplePDB)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	s, err := ReadFile(path, FormatUnknown)
	if err != nil {
		t.Fatalf("ReadFile on gzipped input: %v", err)
	}
	if len(s.Atoms) != 7 || s.ID != "1ABC" {
		t.Errorf("gzipped read gave %d atoms, ID %q", len(s.Atoms), s.ID)
	}
}

func recordOrder(output string, prefixes ...string) string {
	var out []string
	for _, l := range strings.Split(output, "\n") {
		for _, p := range prefixes {
			if strings.HasPrefix(l, p) {
				out = append(out, p)
				break
			}
		}
	}
	return strings.Join(out, ",")
}

func coordinateLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.HasPrefix(l, "ATOM") || strings.HasPrefix(l, "HETATM") {
			for len(l) < 80 {
				l += " "
			}
			out = append(out, l)
		}
	}
	return out
}
