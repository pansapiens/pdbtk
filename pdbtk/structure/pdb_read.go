package structure

import (
	"fmt"
	"strconv"
	"strings"
)

// ReadPDB parses a legacy PDB file. Every ATOM/HETATM record is retained, in
// file order; nothing is filtered or reinterpreted at this layer.
func ReadPDB(content []byte) (*Structure, error) {
	s := &Structure{Source: FormatPDB}
	seqres := make(map[string][]string)
	var seqresOrder []string
	modres := make(map[string]string)
	model := 1

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimRight(line, "\r")
		if len(line) < 6 {
			continue
		}
		switch strings.TrimRight(line[:6], " ") {
		case "HEADER":
			s.ID = field(line, 62, 66)
			if s.ID == "" {
				// pdbtk 0.1.x wrote the ID into the classification field.
				if rest := field(line, 10, len(line)); len(rest) == 4 {
					s.ID = rest
				}
			}
		case "MODEL":
			if n, err := strconv.Atoi(strings.TrimSpace(field(line, 10, 14))); err == nil {
				model = n
			}
		case "ENDMDL":
			model++
		case "SEQRES":
			chain := field(line, 11, 12)
			if chain == "" {
				chain = " "
			}
			if _, seen := seqres[chain]; !seen {
				seqresOrder = append(seqresOrder, chain)
			}
			seqres[chain] = append(seqres[chain], strings.Fields(field(line, 19, 70))...)
		case "MODRES":
			// MODRES stdRes gives the parent component for a modified residue.
			res := field(line, 12, 15)
			std := field(line, 24, 27)
			if res != "" && std != "" {
				modres[strings.ToUpper(res)] = strings.ToUpper(std)
			}
		case "LINK":
			if l, ok := parsePDBLink(line); ok {
				s.Links = append(s.Links, l)
			}
		case "ATOM", "HETATM":
			a, err := parsePDBAtom(line, model)
			if err != nil {
				return nil, err
			}
			s.Atoms = append(s.Atoms, a)
		}
	}

	if len(s.Atoms) == 0 {
		return nil, fmt.Errorf("no ATOM or HETATM records found; this does not look like a PDB file")
	}

	// A file with a single unnumbered MODEL still leaves every atom at model 1.
	for _, chain := range seqresOrder {
		s.SeqRes = append(s.SeqRes, ChainSeq{ChainID: strings.TrimSpace(chain), Residues: seqres[chain]})
	}
	s.ModRes = modres
	return s, nil
}

// field extracts columns [start, end) using 0-based indices, tolerating lines
// truncated before end (common in files trimmed of trailing whitespace).
func field(line string, start, end int) string {
	if start >= len(line) {
		return ""
	}
	if end > len(line) {
		end = len(line)
	}
	return strings.TrimSpace(line[start:end])
}

// charAt returns the single character at a 0-based column, or "" when the
// column is absent or blank.
func charAt(line string, col int) string {
	if col >= len(line) {
		return ""
	}
	c := line[col]
	if c == ' ' {
		return ""
	}
	return string(c)
}

func parsePDBAtom(line string, model int) (*Atom, error) {
	a := &Atom{
		Model:   model,
		Hetatm:  strings.HasPrefix(line, "HETATM"),
		Name:    field(line, 12, 16),
		AltLoc:  charAt(line, 16),
		ResName: field(line, 17, 20),
		ChainID: charAt(line, 21),
		InsCode: charAt(line, 26),
		Element: field(line, 76, 78),
		Charge:  field(line, 78, 80),
	}

	if v := field(line, 6, 11); v != "" {
		n, err := decodeHybrid36(v, 5)
		if err != nil {
			return nil, fmt.Errorf("bad atom serial %q: %v", v, err)
		}
		a.Serial = n
	}
	if v := field(line, 22, 26); v != "" {
		n, err := decodeHybrid36(v, 4)
		if err != nil {
			return nil, fmt.Errorf("bad residue sequence number %q: %v", v, err)
		}
		a.ResSeq = n
	}

	var err error
	if a.X, err = parseCoord(line, 30, 38); err != nil {
		return nil, err
	}
	if a.Y, err = parseCoord(line, 38, 46); err != nil {
		return nil, err
	}
	if a.Z, err = parseCoord(line, 46, 54); err != nil {
		return nil, err
	}

	if v := field(line, 54, 60); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			a.Occupancy, a.HasOcc = f, true
		}
	}
	if v := field(line, 60, 66); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			a.BFactor, a.HasB = f, true
		}
	}

	if a.Element == "" {
		a.Element = guessElement(a.Name, a.ResName)
	}
	return a, nil
}

func parseCoord(line string, start, end int) (float64, error) {
	v := field(line, start, end)
	if v == "" {
		return 0, fmt.Errorf("missing coordinate in columns %d-%d of: %s", start+1, end, strings.TrimRight(line, " "))
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("bad coordinate %q: %v", v, err)
	}
	return f, nil
}

// guessElement recovers an element symbol for files that omit columns 77-78.
// Standard PDB atom names are right-justified so that the element occupies
// columns 13-14, which is what makes two-character elements distinguishable
// from names like "CA" (alpha carbon) in a four-character field.
func guessElement(name, resName string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	// A name starting in column 13 with two letters is a two-character element
	// (FE, ZN, MG). Inside a polymer residue the same columns hold names like
	// "CA"/"CD", which are carbons rather than calcium or cadmium, so only
	// symbols that genuinely occur in polymer residues are honoured there.
	if len(name) >= 2 && name[0] != ' ' {
		cand := strings.ToUpper(strings.TrimSpace(name[:2]))
		if len(cand) == 2 && isAlpha(cand[0]) && isAlpha(cand[1]) {
			if !IsPolymerComponent(resName) || polymerTwoLetterElements[cand] {
				return cand
			}
		}
	}
	for i := 0; i < len(trimmed); i++ {
		if isAlpha(trimmed[i]) {
			return strings.ToUpper(string(trimmed[i]))
		}
	}
	return ""
}

// polymerTwoLetterElements are two-letter element symbols that appear as atoms
// of a polymer residue itself: selenium in selenomethionine/selenocysteine.
var polymerTwoLetterElements = map[string]bool{"SE": true}

func isAlpha(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func parsePDBLink(line string) (Link, bool) {
	if len(line) < 54 {
		return Link{}, false
	}
	a := AtomRef{
		AtomName: field(line, 12, 16),
		AltLoc:   charAt(line, 16),
		ResName:  field(line, 17, 20),
		ChainID:  charAt(line, 21),
		InsCode:  charAt(line, 26),
	}
	b := AtomRef{
		AtomName: field(line, 42, 46),
		AltLoc:   charAt(line, 46),
		ResName:  field(line, 47, 50),
		ChainID:  charAt(line, 51),
		InsCode:  charAt(line, 56),
	}
	sa, err1 := strconv.Atoi(field(line, 22, 26))
	sb, err2 := strconv.Atoi(field(line, 52, 56))
	if err1 != nil || err2 != nil || a.AtomName == "" || b.AtomName == "" {
		return Link{}, false
	}
	a.ResSeq, b.ResSeq = sa, sb
	a.SymOp, b.SymOp = field(line, 59, 65), field(line, 66, 72)

	l := Link{Type: "covale", A: a, B: b}
	if v := field(line, 73, 78); v != "" {
		if d, err := strconv.ParseFloat(v, 64); err == nil {
			l.Distance, l.HasDist = d, true
		}
	}
	return l, true
}
