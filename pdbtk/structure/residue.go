package structure

import "strings"

// Component code tables. Ported from the vendored TuftsBCB abbrev.go, which
// dispatches purely on code length: 3 characters is protein, 2 is
// deoxyribonucleotide, 1 is ribonucleotide.
var aminoMap = map[string]byte{
	"UNK": 'X',
	"ALA": 'A', "ARG": 'R', "ASN": 'N', "ASP": 'D', "CYS": 'C',
	"GLU": 'E', "GLN": 'Q', "GLY": 'G', "HIS": 'H', "ILE": 'I',
	"LEU": 'L', "LYS": 'K', "MET": 'M', "PHE": 'F', "PRO": 'P',
	"SER": 'S', "THR": 'T', "TRP": 'W', "TYR": 'Y', "VAL": 'V',
	"SEC": 'U', "PYL": 'O',

	// Frequently seen modified residues, resolved to their parent so that
	// selenomethionine structures do not come out as a run of X.
	"MSE": 'M', "HYP": 'P', "PCA": 'E', "CSO": 'C', "SEP": 'S',
	"TPO": 'T', "PTR": 'Y', "MLY": 'K', "KCX": 'K', "LLP": 'K',
	"CME": 'C', "CSD": 'C', "OCS": 'C', "M3L": 'K', "FME": 'M',
	"ABA": 'A', "DAL": 'A', "AIB": 'A', "NLE": 'L', "ORN": 'K',

	// Ambiguous or non-residue codes.
	"ASX": 'X', "GLX": 'X', "DLE": 'X', "DOP": 'X', "8OG": 'X', "NH2": 'X',
}

var deoxyMap = map[string]byte{
	"DA": 'A', "DC": 'C', "DG": 'G', "DT": 'T', "DI": 'I', "DU": 'U',
}

var riboMap = map[string]byte{
	"A": 'A', "C": 'C', "G": 'G', "U": 'U', "I": 'I', "T": 'T',
	"N": 'X',
}

// OneLetter converts a component code to its single-letter abbreviation,
// returning 'X' for anything unrecognised. modres supplies MODRES/struct_conf
// parent mappings discovered while reading the file, and takes precedence over
// the built-in tables.
func OneLetter(code string, modres map[string]string) byte {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return 'X'
	}
	if parent, ok := modres[code]; ok {
		if r := lookupCode(strings.ToUpper(strings.TrimSpace(parent))); r != 0 {
			return r
		}
	}
	if r := lookupCode(code); r != 0 {
		return r
	}
	return 'X'
}

func lookupCode(code string) byte {
	switch len(code) {
	case 1:
		if r, ok := riboMap[code]; ok {
			return r
		}
	case 2:
		if r, ok := deoxyMap[code]; ok {
			return r
		}
	case 3:
		if r, ok := aminoMap[code]; ok {
			return r
		}
	}
	return 0
}

// IsPolymerComponent reports whether a component code names a standard or
// commonly modified polymer residue, as opposed to a ligand, ion or solvent.
func IsPolymerComponent(code string) bool {
	return lookupCode(strings.ToUpper(strings.TrimSpace(code))) != 0
}
