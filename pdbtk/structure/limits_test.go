package structure

import (
	"bytes"
	"strings"
	"testing"
)

// syntheticLarge builds a structure that breaks several legacy PDB limits at
// once: more atoms than the five-digit serial field holds, multi-character
// chain IDs, and a five-character CCD component code. Entries like this exist
// only as mmCIF and run to tens of megabytes, so synthesising one is cheaper
// than carrying a fixture.
func syntheticLarge(residuesPerChain int, chains []string) *Structure {
	s := &Structure{ID: "8BIG", Source: FormatCIF, ModRes: map[string]string{}}
	names := [...]string{"N", "CA", "C", "O"}
	for _, ch := range chains {
		for r := 1; r <= residuesPerChain; r++ {
			for i, n := range names {
				s.Atoms = append(s.Atoms, &Atom{
					Serial:  len(s.Atoms) + 1,
					Model:   1,
					Name:    n,
					ResName: "ALA",
					ChainID: ch,
					ResSeq:  r,
					Element: string(n[0]),
					// Kept inside the PDB fixed-point range so that the field
					// width checks do not fire alongside the ones under test.
					X:         float64(r%900) + float64(i)/10,
					Y:         float64(i),
					Z:         float64(r % 900),
					Occupancy: 1, HasOcc: true,
					BFactor: 25.5, HasB: true,
				})
			}
		}
		s.SeqRes = append(s.SeqRes, ChainSeq{ChainID: ch, Residues: []string{"ALA", "ALA"}})
	}
	// A five-character component code, legal in mmCIF and unrepresentable in PDB.
	s.Atoms[len(s.Atoms)-1].ResName = "A1B2C"
	return s
}

func TestStructureBeyondPDBLimits(t *testing.T) {
	const perChain = 12501 // 4 atoms each => 100,008 atoms over two chains
	s := syntheticLarge(perChain, []string{"AAA", "BBB"})
	if len(s.Atoms) <= 99999 {
		t.Fatalf("fixture has %d atoms, want more than the PDB serial field holds", len(s.Atoms))
	}

	t.Run("round trips through mmCIF", func(t *testing.T) {
		var cif bytes.Buffer
		if err := WriteCIF(&cif, s, WriteOptions{Version: "test", CommandLine: "test"}); err != nil {
			t.Fatalf("WriteCIF: %v", err)
		}
		back, err := ReadCIF(cif.Bytes())
		if err != nil {
			t.Fatalf("ReadCIF: %v", err)
		}
		if len(back.Atoms) != len(s.Atoms) {
			t.Fatalf("atom count %d -> %d", len(s.Atoms), len(back.Atoms))
		}
		if got := back.ChainIDs(); len(got) != 2 || got[0] != "AAA" || got[1] != "BBB" {
			t.Errorf("chain IDs = %v, want [AAA BBB]", got)
		}
		last, wantLast := back.Atoms[len(back.Atoms)-1], s.Atoms[len(s.Atoms)-1]
		if last.ResName != "A1B2C" {
			t.Errorf("five-character component code = %q, want A1B2C", last.ResName)
		}
		if last.X != wantLast.X || last.ResSeq != wantLast.ResSeq || last.BFactor != wantLast.BFactor {
			t.Errorf("last atom changed:\n in  %+v\n out %+v", *wantLast, *last)
		}
	})

	t.Run("is refused by the PDB writer", func(t *testing.T) {
		problems := FitPDB(s)
		joined := strings.Join(problems, "; ")
		for _, want := range []string{`chain ID "AAA"`, `residue name "A1B2C"`} {
			if !strings.Contains(joined, want) {
				t.Errorf("FitPDB did not report %s: %v", want, problems)
			}
		}

		err := WritePDB(&bytes.Buffer{}, s, WriteOptions{Version: "test"})
		if err == nil {
			t.Fatal("WritePDB accepted a structure that does not fit the PDB format")
		}
		if !strings.Contains(err.Error(), "force-lossy-pdb") {
			t.Errorf("error should point at --force-lossy-pdb, got: %v", err)
		}
	})

	t.Run("uses hybrid-36 serials under force-lossy", func(t *testing.T) {
		var buf bytes.Buffer
		var warnings []string
		err := WritePDB(&buf, s, WriteOptions{
			Version: "test", ForceLossy: true,
			Warn: func(m string) { warnings = append(warnings, m) },
		})
		if err != nil {
			t.Fatalf("WritePDB with ForceLossy: %v", err)
		}
		if len(warnings) == 0 {
			t.Error("expected warnings about the lossy conversion")
		}

		// Serials past 99999 must be hybrid-36 encoded rather than overflowing
		// into the neighbouring column. TER records take serials too, so the
		// encoded run starts a little before the 100,000th atom.
		if !strings.Contains(buf.String(), "\nATOM  A0000 ") {
			t.Error("expected serial 100000 to be written as hybrid-36 A0000")
		}

		back, err := ReadPDB(buf.Bytes())
		if err != nil {
			t.Fatalf("re-reading force-lossy output: %v", err)
		}
		if len(back.Atoms) != len(s.Atoms) {
			t.Errorf("atom count %d -> %d through PDB", len(s.Atoms), len(back.Atoms))
		}
		prev := 0
		for i, a := range back.Atoms {
			if a.Serial <= prev {
				t.Fatalf("serial went backwards at atom %d: %d after %d", i, a.Serial, prev)
			}
			prev = a.Serial
		}
		if prev <= 99999 {
			t.Errorf("highest serial read back = %d, want more than 99999", prev)
		}
		// Multi-character chain IDs and long component codes are the lossy part.
		if back.Atoms[0].ChainID != "A" {
			t.Errorf("chain ID = %q, want it truncated to A", back.Atoms[0].ChainID)
		}
	})
}

// Coordinates and the fixed-point columns have no hybrid-36 escape hatch, so an
// out-of-range value would otherwise shift every later column and make the
// record unreadable — including by pdbtk itself.
func TestFitPDBRejectsOutOfRangeFixedPointFields(t *testing.T) {
	base := func() *Structure {
		return &Structure{ID: "1ABC", ModRes: map[string]string{}, Atoms: []*Atom{{
			Model: 1, Name: "CA", ResName: "ALA", ChainID: "A", ResSeq: 1, Element: "C",
			Occupancy: 1, HasOcc: true, BFactor: 10, HasB: true,
		}}}
	}

	cases := []struct {
		name    string
		mutate  func(*Atom)
		wantSub string
	}{
		{"x too large", func(a *Atom) { a.X = 12345.678 }, "x coordinate"},
		{"y too negative", func(a *Atom) { a.Y = -1234.567 }, "y coordinate"},
		{"z in range", func(a *Atom) { a.Z = 9999.999 }, ""},
		{"B-factor too large", func(a *Atom) { a.BFactor = 1234.56 }, "B-factor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := base()
			tc.mutate(s.Atoms[0])
			problems := strings.Join(FitPDB(s), "; ")

			if tc.wantSub == "" {
				if problems != "" {
					t.Fatalf("FitPDB rejected a representable value: %s", problems)
				}
				return
			}
			if !strings.Contains(problems, tc.wantSub) {
				t.Fatalf("FitPDB did not report %s: %q", tc.wantSub, problems)
			}

			// Under --force-lossy-pdb the field becomes asterisks rather than
			// overflowing, so the remaining columns stay where they belong.
			var buf bytes.Buffer
			if err := WritePDB(&buf, s, WriteOptions{ForceLossy: true}); err != nil {
				t.Fatalf("WritePDB with ForceLossy: %v", err)
			}
			for _, l := range strings.Split(buf.String(), "\n") {
				if !strings.HasPrefix(l, "ATOM") {
					continue
				}
				if !strings.Contains(l, "*") {
					t.Errorf("expected an asterisk-filled field, got %q", l)
				}
				if got := strings.TrimSpace(l[76:78]); got != "C" {
					t.Errorf("element column shifted: %q in %q", got, l)
				}
			}
		})
	}
}
