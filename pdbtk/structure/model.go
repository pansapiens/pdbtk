// Package structure provides a format-neutral representation of macromolecular
// structures, along with codecs for legacy PDB and PDBx/mmCIF files.
package structure

import (
	"fmt"
	"strings"
)

// Format identifies a structure file format.
type Format int

const (
	FormatUnknown Format = iota
	FormatPDB
	FormatCIF
)

func (f Format) String() string {
	switch f {
	case FormatPDB:
		return "pdb"
	case FormatCIF:
		return "cif"
	}
	return "unknown"
}

// ParseFormat converts a user-supplied format name to a Format.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pdb":
		return FormatPDB, nil
	case "cif", "mmcif", "pdbx":
		return FormatCIF, nil
	}
	return FormatUnknown, fmt.Errorf("unknown format %q (supported: pdb, cif)", s)
}

// Structure is a whole entry: a flat, file-ordered list of atoms plus the
// connectivity and sequence records pdbtk knows how to carry across formats.
//
// Atoms are deliberately kept flat rather than nested in a chain/model/residue
// tree: it preserves input order exactly, and every pdbtk operation is then
// either a filter over Atoms or a rewrite of one of their fields.
type Structure struct {
	ID     string
	Atoms  []*Atom
	Links  []Link
	SeqRes []ChainSeq
	Source Format

	// ModRes maps a modified component code to its parent, from PDB MODRES or
	// mmCIF pdbx_struct_mod_residue. Used when deriving one-letter sequences.
	ModRes map[string]string
}

// Atom is one ATOM/HETATM record (PDB) or one _atom_site row (mmCIF).
type Atom struct {
	Serial  int
	Model   int
	Hetatm  bool
	Name    string
	AltLoc  string
	ResName string
	ChainID string
	ResSeq  int
	InsCode string
	X, Y, Z float64

	// HasOcc/HasB distinguish an absent value from a legitimate 0.0.
	Occupancy float64
	HasOcc    bool
	BFactor   float64
	HasB      bool

	Element string
	Charge  string

	// mmCIF label_* namespace, preserved verbatim on CIF read and synthesised
	// on PDB->CIF conversion.
	LabelAsymID   string
	LabelCompID   string
	LabelAtomID   string
	LabelEntityID string
	LabelSeqID    int
	HasLabelSeq   bool
}

// IsWater reports whether the atom belongs to a solvent water residue.
func (a *Atom) IsWater() bool {
	switch strings.ToUpper(strings.TrimSpace(a.ResName)) {
	case "HOH", "DOD", "WAT":
		return true
	}
	return false
}

// AtomRef identifies one end of a bond, independent of the atom list.
type AtomRef struct {
	AtomName string
	AltLoc   string
	ResName  string
	ChainID  string
	ResSeq   int
	InsCode  string
	SymOp    string
}

// Link is a LINK record (PDB) or a struct_conn row (mmCIF).
type Link struct {
	Type     string // covale, metalc, disulf, ...
	A, B     AtomRef
	Distance float64
	HasDist  bool
}

// ChainSeq is the full polymer sequence for one chain, as 3-letter component
// codes, from SEQRES (PDB) or pdbx_poly_seq_scheme (mmCIF).
type ChainSeq struct {
	ChainID  string
	Residues []string
}

// ResidueKey groups atoms belonging to a single residue.
type ResidueKey struct {
	Model   int
	ChainID string
	ResSeq  int
	InsCode string
}

// Residue is a contiguous run of atoms sharing a ResidueKey.
type Residue struct {
	ResidueKey
	ResName string
	Atoms   []*Atom
}

// KeyOf returns the grouping key for an atom.
func KeyOf(a *Atom) ResidueKey {
	return ResidueKey{Model: a.Model, ChainID: a.ChainID, ResSeq: a.ResSeq, InsCode: a.InsCode}
}

// Residues groups atoms into residues, preserving first-appearance order.
// A residue's atoms need not be contiguous in the input; they are collected
// under the same key wherever they appear.
func Residues(atoms []*Atom) []*Residue {
	index := make(map[ResidueKey]*Residue)
	var out []*Residue
	for _, a := range atoms {
		k := KeyOf(a)
		r, ok := index[k]
		if !ok {
			r = &Residue{ResidueKey: k, ResName: a.ResName}
			index[k] = r
			out = append(out, r)
		}
		r.Atoms = append(r.Atoms, a)
	}
	return out
}

// ChainIDs returns the distinct chain identifiers in first-appearance order.
func (s *Structure) ChainIDs() []string {
	seen := make(map[string]bool)
	var out []string
	for _, a := range s.Atoms {
		if !seen[a.ChainID] {
			seen[a.ChainID] = true
			out = append(out, a.ChainID)
		}
	}
	return out
}

// HasMultipleModels reports whether the structure spans more than one model.
func (s *Structure) HasMultipleModels() bool {
	first := 0
	for _, a := range s.Atoms {
		if first == 0 {
			first = a.Model
			continue
		}
		if a.Model != first {
			return true
		}
	}
	return false
}

// Renumber assigns sequential serial numbers starting at 1, in atom order.
func (s *Structure) Renumber() {
	for i, a := range s.Atoms {
		a.Serial = i + 1
	}
}

// FilterAtoms returns a new Structure keeping only atoms satisfying keep.
// Links whose endpoints no longer both exist are dropped, and SeqRes entries
// for chains with no surviving atoms are dropped.
func (s *Structure) FilterAtoms(keep func(*Atom) bool) *Structure {
	out := &Structure{ID: s.ID, Source: s.Source, ModRes: s.ModRes}
	for _, a := range s.Atoms {
		if keep(a) {
			out.Atoms = append(out.Atoms, a)
		}
	}
	out.Links = filterLinks(s.Links, out.Atoms)
	out.SeqRes = filterSeqRes(s.SeqRes, out.Atoms)
	return out
}

func refKey(chainID, resName, atomName string, resSeq int, insCode string) string {
	return fmt.Sprintf("%s|%d|%s|%s|%s",
		chainID,
		resSeq,
		normalizeInsCode(insCode),
		strings.ToUpper(strings.TrimSpace(resName)),
		strings.ToUpper(strings.TrimSpace(atomName)),
	)
}

func normalizeInsCode(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "." || s == "?" {
		return ""
	}
	return s
}

func filterLinks(links []Link, atoms []*Atom) []Link {
	if len(links) == 0 {
		return nil
	}
	present := make(map[string]bool, len(atoms))
	for _, a := range atoms {
		present[refKey(a.ChainID, a.ResName, a.Name, a.ResSeq, a.InsCode)] = true
	}
	var out []Link
	for _, l := range links {
		ka := refKey(l.A.ChainID, l.A.ResName, l.A.AtomName, l.A.ResSeq, l.A.InsCode)
		kb := refKey(l.B.ChainID, l.B.ResName, l.B.AtomName, l.B.ResSeq, l.B.InsCode)
		if present[ka] && present[kb] {
			out = append(out, l)
		}
	}
	return out
}

func filterSeqRes(seqres []ChainSeq, atoms []*Atom) []ChainSeq {
	if len(seqres) == 0 {
		return nil
	}
	present := make(map[string]bool)
	for _, a := range atoms {
		present[a.ChainID] = true
	}
	var out []ChainSeq
	for _, cs := range seqres {
		if present[cs.ChainID] {
			out = append(out, cs)
		}
	}
	return out
}

// Clone returns a deep copy, so transforms can rewrite atom fields without
// mutating the caller's structure.
func (s *Structure) Clone() *Structure {
	out := &Structure{
		ID:     s.ID,
		Source: s.Source,
		ModRes: s.ModRes,
		Atoms:  make([]*Atom, len(s.Atoms)),
		Links:  append([]Link(nil), s.Links...),
	}
	for i, a := range s.Atoms {
		c := *a
		out.Atoms[i] = &c
	}
	for _, cs := range s.SeqRes {
		out.SeqRes = append(out.SeqRes, ChainSeq{
			ChainID:  cs.ChainID,
			Residues: append([]string(nil), cs.Residues...),
		})
	}
	return out
}
