package structure

import "testing"

// Files produced by some refinement software carry only the label_* namespace.
// The reader has to fall back to it for the fields it would normally take from
// auth_*, or every atom lands in an empty chain with residue number zero.
func TestCIFFallsBackToLabelColumns(t *testing.T) {
	const src = `data_TEST
loop_
_atom_site.group_PDB
_atom_site.id
_atom_site.type_symbol
_atom_site.label_atom_id
_atom_site.label_comp_id
_atom_site.label_asym_id
_atom_site.label_seq_id
_atom_site.Cartn_x
_atom_site.Cartn_y
_atom_site.Cartn_z
ATOM 1 N N  ALA A 1 1.000 2.000 3.000
ATOM 2 C CA ALA A 1 4.000 5.000 6.000
#
`
	s, err := ReadCIF([]byte(src))
	if err != nil {
		t.Fatalf("ReadCIF: %v", err)
	}
	a := s.Atoms[0]
	if a.ChainID != "A" || a.ResSeq != 1 || a.ResName != "ALA" || a.Name != "N" {
		t.Errorf("label_* fallback failed: %+v", *a)
	}
	if !a.HasLabelSeq || a.LabelSeqID != 1 {
		t.Errorf("label_seq_id not retained: %+v", *a)
	}
	// Absent occupancy and B-factor must stay absent rather than becoming 0.0,
	// so the writer can substitute the format's defaults.
	if a.HasOcc || a.HasB {
		t.Errorf("missing occupancy/B recorded as present: %+v", *a)
	}
}

// The . and ? null markers are not values. Reading them as literals would put
// "?" in a chain ID or an insertion code.
func TestCIFNullMarkersAreNotValues(t *testing.T) {
	const src = `data_TEST
loop_
_atom_site.group_PDB
_atom_site.id
_atom_site.type_symbol
_atom_site.label_atom_id
_atom_site.label_alt_id
_atom_site.label_comp_id
_atom_site.auth_asym_id
_atom_site.auth_seq_id
_atom_site.pdbx_PDB_ins_code
_atom_site.occupancy
_atom_site.B_iso_or_equiv
_atom_site.Cartn_x
_atom_site.Cartn_y
_atom_site.Cartn_z
ATOM 1 N N . ALA A 1 ? . ? 1.000 2.000 3.000
#
`
	s, err := ReadCIF([]byte(src))
	if err != nil {
		t.Fatalf("ReadCIF: %v", err)
	}
	a := s.Atoms[0]
	if a.AltLoc != "" || a.InsCode != "" {
		t.Errorf("null markers read as values: altLoc=%q insCode=%q", a.AltLoc, a.InsCode)
	}
	if a.HasOcc || a.HasB {
		t.Errorf("null occupancy/B recorded as present: %+v", *a)
	}
}

// When pdbx_poly_seq_scheme is absent, SEQRES has to be recovered by joining
// entity_poly_seq to struct_asym through the entity ID.
func TestCIFSeqResFromEntityPolySeq(t *testing.T) {
	const src = `data_TEST
loop_
_atom_site.group_PDB
_atom_site.id
_atom_site.type_symbol
_atom_site.label_atom_id
_atom_site.label_comp_id
_atom_site.auth_asym_id
_atom_site.auth_seq_id
_atom_site.Cartn_x
_atom_site.Cartn_y
_atom_site.Cartn_z
ATOM 1 C CA ALA A 1 1.000 2.000 3.000
#
loop_
_struct_asym.id
_struct_asym.entity_id
A 1
B 2
#
loop_
_entity_poly_seq.entity_id
_entity_poly_seq.num
_entity_poly_seq.mon_id
1 1 ALA
1 2 GLY
2 1 VAL
#
`
	s, err := ReadCIF([]byte(src))
	if err != nil {
		t.Fatalf("ReadCIF: %v", err)
	}
	if len(s.SeqRes) != 2 {
		t.Fatalf("recovered %d chains of SEQRES, want 2: %+v", len(s.SeqRes), s.SeqRes)
	}
	if s.SeqRes[0].ChainID != "A" || len(s.SeqRes[0].Residues) != 2 ||
		s.SeqRes[0].Residues[1] != "GLY" {
		t.Errorf("chain A sequence = %+v, want [ALA GLY]", s.SeqRes[0])
	}
	if s.SeqRes[1].ChainID != "B" || len(s.SeqRes[1].Residues) != 1 {
		t.Errorf("chain B sequence = %+v, want [VAL]", s.SeqRes[1])
	}
}

// struct_conn also records hydrogen bonds, which are inferred annotation rather
// than the connectivity a LINK record describes.
func TestCIFStructConnSkipsHydrogenBondsAndReadsModRes(t *testing.T) {
	const src = `data_TEST
loop_
_atom_site.group_PDB
_atom_site.id
_atom_site.type_symbol
_atom_site.label_atom_id
_atom_site.label_comp_id
_atom_site.auth_asym_id
_atom_site.auth_seq_id
_atom_site.Cartn_x
_atom_site.Cartn_y
_atom_site.Cartn_z
ATOM   1 S SG CYS A 3  1.000 2.000 3.000
HETATM 2 ZN ZN ZN  A 99 4.000 5.000 6.000
#
loop_
_struct_conn.id
_struct_conn.conn_type_id
_struct_conn.ptnr1_label_atom_id
_struct_conn.ptnr1_auth_comp_id
_struct_conn.ptnr1_auth_asym_id
_struct_conn.ptnr1_auth_seq_id
_struct_conn.ptnr2_label_atom_id
_struct_conn.ptnr2_auth_comp_id
_struct_conn.ptnr2_auth_asym_id
_struct_conn.ptnr2_auth_seq_id
_struct_conn.pdbx_dist_value
metalc1 metalc SG CYS A 3 ZN ZN A 99 2.310
hydrog1 hydrog N  ALA A 1 O  ALA A 2 2.900
#
loop_
_pdbx_struct_mod_residue.id
_pdbx_struct_mod_residue.auth_comp_id
_pdbx_struct_mod_residue.parent_comp_id
1 MSE MET
#
`
	s, err := ReadCIF([]byte(src))
	if err != nil {
		t.Fatalf("ReadCIF: %v", err)
	}
	if len(s.Links) != 1 {
		t.Fatalf("kept %d links, want only the metal coordination: %+v", len(s.Links), s.Links)
	}
	l := s.Links[0]
	if l.Type != "metalc" || l.A.AtomName != "SG" || l.B.ResName != "ZN" {
		t.Errorf("link = %+v, want the CYS SG - ZN metal bond", l)
	}
	if !l.HasDist || l.Distance != 2.310 {
		t.Errorf("link distance = %v/%v, want 2.31", l.Distance, l.HasDist)
	}
	if s.ModRes["MSE"] != "MET" {
		t.Errorf("pdbx_struct_mod_residue mapping = %v, want MSE->MET", s.ModRes)
	}
}

func TestCIFRejectsMultipleDataBlocks(t *testing.T) {
	const src = `data_ONE
_entry.id ONE
data_TWO
_entry.id TWO
`
	if _, err := ReadCIF([]byte(src)); err == nil {
		t.Error("expected an error for a file with two data blocks")
	}
}
