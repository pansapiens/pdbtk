package structure

import (
	"fmt"
	"strconv"
	"strings"
)

// ReadCIF parses a PDBx/mmCIF file. Exactly one data block is expected, which
// is what coordinate files from the PDB contain.
func ReadCIF(content []byte) (*Structure, error) {
	blocks, err := cifParse(string(content))
	if err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no data block found; this does not look like an mmCIF file")
	}
	if len(blocks) > 1 {
		return nil, fmt.Errorf("expected one data block, found %d", len(blocks))
	}
	b := blocks[0]

	s := &Structure{Source: FormatCIF, ModRes: make(map[string]string)}
	s.ID = strings.ToUpper(b.Category("entry").First("id"))
	if s.ID == "" {
		s.ID = strings.ToUpper(b.Name)
	}

	if err := readCIFAtomSites(s, b.Category("atom_site")); err != nil {
		return nil, err
	}
	readCIFStructConn(s, b.Category("struct_conn"))
	readCIFModRes(s, b.Category("pdbx_struct_mod_residue"))
	readCIFSeqRes(s, b)
	return s, nil
}

// cifCol pulls a column and reports whether it was present, so callers can
// fall back from auth_* to label_* naming.
func cifCol(c *cifCategory, names ...string) []string {
	for _, n := range names {
		if v := c.Col(n); v != nil {
			return v
		}
	}
	return nil
}

func readCIFAtomSites(s *Structure, c *cifCategory) error {
	if c == nil || len(c.Rows) == 0 {
		return fmt.Errorf("no _atom_site records found")
	}

	n := len(c.Rows)
	group := c.Col("group_PDB")
	serial := c.Col("id")
	element := c.Col("type_symbol")
	charge := c.Col("pdbx_formal_charge")
	altLoc := c.Col("label_alt_id")
	insCode := c.Col("pdbx_PDB_ins_code")
	modelNum := c.Col("pdbx_PDB_model_num")

	labelAtom := c.Col("label_atom_id")
	labelComp := c.Col("label_comp_id")
	labelAsym := c.Col("label_asym_id")
	labelEntity := c.Col("label_entity_id")
	labelSeq := c.Col("label_seq_id")

	// auth_* is the numbering biologists expect and matches the legacy PDB
	// file for the same entry; fall back to label_* when it is absent.
	name := cifCol(c, "auth_atom_id", "label_atom_id")
	comp := cifCol(c, "auth_comp_id", "label_comp_id")
	asym := cifCol(c, "auth_asym_id", "label_asym_id")
	seqID := cifCol(c, "auth_seq_id", "label_seq_id")

	x, y, z := c.Col("Cartn_x"), c.Col("Cartn_y"), c.Col("Cartn_z")
	if x == nil || y == nil || z == nil {
		return fmt.Errorf("_atom_site is missing Cartn_x/Cartn_y/Cartn_z")
	}
	occ, bfac := c.Col("occupancy"), c.Col("B_iso_or_equiv")

	at := func(col []string, i int) string {
		if i < len(col) {
			return cifValue(col[i])
		}
		return ""
	}

	s.Atoms = make([]*Atom, 0, n)
	for i := 0; i < n; i++ {
		a := &Atom{
			Model:         1,
			Hetatm:        strings.EqualFold(at(group, i), "HETATM"),
			Name:          at(name, i),
			AltLoc:        at(altLoc, i),
			ResName:       at(comp, i),
			ChainID:       at(asym, i),
			InsCode:       at(insCode, i),
			Element:       at(element, i),
			Charge:        at(charge, i),
			LabelAtomID:   at(labelAtom, i),
			LabelCompID:   at(labelComp, i),
			LabelAsymID:   at(labelAsym, i),
			LabelEntityID: at(labelEntity, i),
		}

		var err error
		if a.X, err = cifFloat(at(x, i), "Cartn_x", i); err != nil {
			return err
		}
		if a.Y, err = cifFloat(at(y, i), "Cartn_y", i); err != nil {
			return err
		}
		if a.Z, err = cifFloat(at(z, i), "Cartn_z", i); err != nil {
			return err
		}

		if v := at(serial, i); v != "" {
			a.Serial, _ = strconv.Atoi(v)
		}
		if v := at(seqID, i); v != "" {
			if a.ResSeq, err = strconv.Atoi(v); err != nil {
				return fmt.Errorf("_atom_site row %d: bad residue number %q", i+1, v)
			}
		}
		if v := at(labelSeq, i); v != "" {
			if ls, err := strconv.Atoi(v); err == nil {
				a.LabelSeqID, a.HasLabelSeq = ls, true
			}
		}
		if v := at(modelNum, i); v != "" {
			if m, err := strconv.Atoi(v); err == nil && m > 0 {
				a.Model = m
			}
		}
		if v := at(occ, i); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				a.Occupancy, a.HasOcc = f, true
			}
		}
		if v := at(bfac, i); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				a.BFactor, a.HasB = f, true
			}
		}
		s.Atoms = append(s.Atoms, a)
	}
	return nil
}

func cifFloat(v, field string, row int) (float64, error) {
	if v == "" {
		return 0, fmt.Errorf("_atom_site row %d: missing %s", row+1, field)
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("_atom_site row %d: bad %s value %q", row+1, field, v)
	}
	return f, nil
}

func readCIFStructConn(s *Structure, c *cifCategory) {
	if c == nil || len(c.Rows) == 0 {
		return
	}
	connType := c.Col("conn_type_id")
	dist := c.Col("pdbx_dist_value")
	sym1, sym2 := c.Col("ptnr1_symmetry"), c.Col("ptnr2_symmetry")

	side := func(n string) (atom, alt, comp, asym, seq, ins []string) {
		p := "ptnr" + n + "_"
		return c.Col(p + "label_atom_id"),
			c.Col(p + "label_alt_id"),
			cifCol(c, p+"auth_comp_id", p+"label_comp_id"),
			cifCol(c, p+"auth_asym_id", p+"label_asym_id"),
			cifCol(c, p+"auth_seq_id", p+"label_seq_id"),
			c.Col("pdbx_ptnr" + n + "_PDB_ins_code")
	}
	a1, al1, c1, as1, sq1, in1 := side("1")
	a2, al2, c2, as2, sq2, in2 := side("2")

	at := func(col []string, i int) string {
		if i < len(col) {
			return cifValue(col[i])
		}
		return ""
	}
	ref := func(atom, alt, comp, asym, seq, ins []string, i int) (AtomRef, bool) {
		r := AtomRef{
			AtomName: at(atom, i),
			AltLoc:   at(alt, i),
			ResName:  at(comp, i),
			ChainID:  at(asym, i),
			InsCode:  at(ins, i),
		}
		v := at(seq, i)
		if r.AtomName == "" || r.ChainID == "" || v == "" {
			return AtomRef{}, false
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return AtomRef{}, false
		}
		r.ResSeq = n
		return r, true
	}

	for i := range c.Rows {
		ra, ok1 := ref(a1, al1, c1, as1, sq1, in1, i)
		rb, ok2 := ref(a2, al2, c2, as2, sq2, in2, i)
		if !ok1 || !ok2 {
			continue
		}
		t := at(connType, i)
		if t == "" {
			t = "covale"
		}
		// hydrog bonds are inferred annotation rather than connectivity pdbtk
		// should carry into a LINK record.
		if strings.EqualFold(t, "hydrog") {
			continue
		}
		ra.SymOp, rb.SymOp = at(sym1, i), at(sym2, i)
		l := Link{Type: strings.ToLower(t), A: ra, B: rb}
		if v := at(dist, i); v != "" {
			if d, err := strconv.ParseFloat(v, 64); err == nil {
				l.Distance, l.HasDist = d, true
			}
		}
		s.Links = append(s.Links, l)
	}
}

func readCIFModRes(s *Structure, c *cifCategory) {
	if c == nil {
		return
	}
	mod := cifCol(c, "auth_comp_id", "label_comp_id")
	parent := c.Col("parent_comp_id")
	for i := range c.Rows {
		if i >= len(mod) || i >= len(parent) {
			break
		}
		m, p := cifValue(mod[i]), cifValue(parent[i])
		if m != "" && p != "" {
			s.ModRes[strings.ToUpper(m)] = strings.ToUpper(p)
		}
	}
}

// readCIFSeqRes prefers pdbx_poly_seq_scheme, which already carries the author
// chain ID alongside each component, and falls back to joining entity_poly_seq
// with struct_asym when it is absent.
func readCIFSeqRes(s *Structure, b *cifBlock) {
	if c := b.Category("pdbx_poly_seq_scheme"); c != nil && len(c.Rows) > 0 {
		chain := cifCol(c, "pdb_strand_id", "asym_id")
		mon := c.Col("mon_id")
		if chain != nil && mon != nil {
			appendSeq(s, chain, mon)
			return
		}
	}

	entityPoly := b.Category("entity_poly_seq")
	structAsym := b.Category("struct_asym")
	if entityPoly == nil || structAsym == nil {
		return
	}
	byEntity := make(map[string][]string)
	ents, mons := entityPoly.Col("entity_id"), entityPoly.Col("mon_id")
	for i := range entityPoly.Rows {
		if i >= len(ents) || i >= len(mons) {
			break
		}
		byEntity[cifValue(ents[i])] = append(byEntity[cifValue(ents[i])], cifValue(mons[i]))
	}

	asymIDs, asymEnts := structAsym.Col("id"), structAsym.Col("entity_id")
	for i := range structAsym.Rows {
		if i >= len(asymIDs) || i >= len(asymEnts) {
			break
		}
		if seq, ok := byEntity[cifValue(asymEnts[i])]; ok {
			s.SeqRes = append(s.SeqRes, ChainSeq{ChainID: cifValue(asymIDs[i]), Residues: seq})
		}
	}
}

func appendSeq(s *Structure, chain, mon []string) {
	index := make(map[string]int)
	for i := range chain {
		if i >= len(mon) {
			break
		}
		ch, m := cifValue(chain[i]), cifValue(mon[i])
		if ch == "" || m == "" {
			continue
		}
		pos, ok := index[ch]
		if !ok {
			index[ch] = len(s.SeqRes)
			pos = len(s.SeqRes)
			s.SeqRes = append(s.SeqRes, ChainSeq{ChainID: ch})
		}
		s.SeqRes[pos].Residues = append(s.SeqRes[pos].Residues, m)
	}
}
