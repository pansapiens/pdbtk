package structure

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// WriteCIF serialises a structure as PDBx/mmCIF. It emits coordinates plus the
// categories pdbtk carries across formats — entry, atom_site, struct_conn and
// pdbx_poly_seq_scheme — along with a pdbtk provenance category. Categories
// present in an input file but not modelled here are not reproduced.
func WriteCIF(w io.Writer, s *Structure, opt WriteOptions) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	id := s.ID
	if id == "" {
		id = "XXXX"
	}

	fmt.Fprintf(bw, "data_%s\n#\n", id)
	fmt.Fprintf(bw, "_entry.id   %s\n#\n", cifQuote(id))
	fmt.Fprintf(bw, "_pdbtk.version   %s\n", cifQuote(opt.Version))
	fmt.Fprintf(bw, "_pdbtk.command   %s\n#\n", cifQuote(opt.CommandLine))

	writeCIFAtomSites(bw, s)
	writeCIFStructConn(bw, s)
	writeCIFPolySeq(bw, s)

	return bw.Flush()
}

var atomSiteColumns = []string{
	"group_PDB", "id", "type_symbol", "label_atom_id", "label_alt_id",
	"label_comp_id", "label_asym_id", "label_entity_id", "label_seq_id",
	"pdbx_PDB_ins_code", "Cartn_x", "Cartn_y", "Cartn_z", "occupancy",
	"B_iso_or_equiv", "pdbx_formal_charge", "auth_seq_id", "auth_comp_id",
	"auth_asym_id", "auth_atom_id", "pdbx_PDB_model_num",
}

func writeCIFAtomSites(bw *bufio.Writer, s *Structure) {
	if len(s.Atoms) == 0 {
		return
	}
	rows := make([][]string, 0, len(s.Atoms))
	for i, a := range s.Atoms {
		group := "ATOM"
		if a.Hetatm {
			group = "HETATM"
		}
		labelSeq := "."
		if a.HasLabelSeq {
			labelSeq = strconv.Itoa(a.LabelSeqID)
		} else if !a.Hetatm {
			labelSeq = strconv.Itoa(a.ResSeq)
		}
		rows = append(rows, []string{
			group,
			strconv.Itoa(i + 1),
			cifOrNull(strings.ToUpper(a.Element)),
			cifOrNull(orDefault(a.LabelAtomID, a.Name)),
			cifOrNull(a.AltLoc),
			cifOrNull(orDefault(a.LabelCompID, a.ResName)),
			cifOrNull(orDefault(a.LabelAsymID, a.ChainID)),
			cifOrNull(orDefault(a.LabelEntityID, "1")),
			labelSeq,
			cifOrNull(a.InsCode),
			formatCoord(a.X),
			formatCoord(a.Y),
			formatCoord(a.Z),
			formatOptFloat(a.Occupancy, a.HasOcc, 1.00),
			formatOptFloat(a.BFactor, a.HasB, 0.00),
			cifOrNull(a.Charge),
			strconv.Itoa(a.ResSeq),
			cifOrNull(a.ResName),
			cifOrNull(a.ChainID),
			cifOrNull(a.Name),
			strconv.Itoa(a.Model),
		})
	}
	writeCIFLoop(bw, "atom_site", atomSiteColumns, rows)
}

var structConnColumns = []string{
	"id", "conn_type_id",
	"ptnr1_label_atom_id", "ptnr1_label_alt_id", "ptnr1_auth_comp_id",
	"ptnr1_auth_asym_id", "ptnr1_auth_seq_id", "pdbx_ptnr1_PDB_ins_code",
	"ptnr2_label_atom_id", "ptnr2_label_alt_id", "ptnr2_auth_comp_id",
	"ptnr2_auth_asym_id", "ptnr2_auth_seq_id", "pdbx_ptnr2_PDB_ins_code",
	"ptnr1_symmetry", "ptnr2_symmetry", "pdbx_dist_value",
}

func writeCIFStructConn(bw *bufio.Writer, s *Structure) {
	if len(s.Links) == 0 {
		return
	}
	rows := make([][]string, 0, len(s.Links))
	for i, l := range s.Links {
		t := l.Type
		if t == "" {
			t = "covale"
		}
		dist := "."
		if l.HasDist {
			dist = fmt.Sprintf("%.3f", l.Distance)
		}
		rows = append(rows, []string{
			fmt.Sprintf("%s%d", t, i+1),
			t,
			cifOrNull(l.A.AtomName), cifOrNull(l.A.AltLoc), cifOrNull(l.A.ResName),
			cifOrNull(l.A.ChainID), strconv.Itoa(l.A.ResSeq), cifOrNull(l.A.InsCode),
			cifOrNull(l.B.AtomName), cifOrNull(l.B.AltLoc), cifOrNull(l.B.ResName),
			cifOrNull(l.B.ChainID), strconv.Itoa(l.B.ResSeq), cifOrNull(l.B.InsCode),
			cifOrNull(l.A.SymOp), cifOrNull(l.B.SymOp),
			dist,
		})
	}
	writeCIFLoop(bw, "struct_conn", structConnColumns, rows)
}

func writeCIFPolySeq(bw *bufio.Writer, s *Structure) {
	if len(s.SeqRes) == 0 {
		return
	}
	var rows [][]string
	for _, cs := range s.SeqRes {
		for i, mon := range cs.Residues {
			rows = append(rows, []string{
				cifOrNull(cs.ChainID),
				strconv.Itoa(i + 1),
				cifOrNull(mon),
				cifOrNull(cs.ChainID),
				strconv.Itoa(i + 1),
			})
		}
	}
	writeCIFLoop(bw, "pdbx_poly_seq_scheme",
		[]string{"asym_id", "seq_id", "mon_id", "pdb_strand_id", "pdb_seq_num"}, rows)
}

// writeCIFLoop emits a loop_ table, padding each column to a common width so
// the output stays readable in a terminal.
func writeCIFLoop(bw *bufio.Writer, category string, columns []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(bw, "loop_\n")
	for _, c := range columns {
		fmt.Fprintf(bw, "_%s.%s\n", category, c)
	}

	widths := make([]int, len(columns))
	quoted := make([][]string, len(rows))
	for r, row := range rows {
		quoted[r] = make([]string, len(row))
		for c, v := range row {
			q := cifQuote(v)
			quoted[r][c] = q
			if c < len(widths) && len(q) > widths[c] {
				widths[c] = len(q)
			}
		}
	}

	for _, row := range quoted {
		var b strings.Builder
		for c, v := range row {
			if c > 0 {
				b.WriteByte(' ')
			}
			if c == len(row)-1 {
				b.WriteString(v)
			} else {
				fmt.Fprintf(&b, "%-*s", widths[c], v)
			}
		}
		fmt.Fprintf(bw, "%s\n", strings.TrimRight(b.String(), " "))
	}
	fmt.Fprintf(bw, "#\n")
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func cifOrNull(v string) string {
	if strings.TrimSpace(v) == "" {
		return "."
	}
	return v
}

func formatCoord(v float64) string {
	return strconv.FormatFloat(v, 'f', 3, 64)
}

func formatOptFloat(v float64, has bool, fallback float64) string {
	if !has {
		v = fallback
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// cifQuote renders a value as a CIF token, quoting when the bare form would be
// ambiguous. Values containing both quote styles, a newline, or a leading
// semicolon must use the multi-line text field form.
func cifQuote(v string) string {
	// "." and "?" are the null markers and must stay bare; cifOrNull has
	// already mapped genuinely empty values onto them.
	if v == "" || v == "." || v == "?" {
		return "."
	}
	needsQuote := strings.ContainsAny(v, " \t'\"") ||
		strings.HasPrefix(v, "_") || strings.HasPrefix(v, "#") ||
		strings.HasPrefix(v, "$") || strings.HasPrefix(v, "[")
	for _, kw := range []string{"data_", "loop_", "save_", "stop_", "global_"} {
		if strings.HasPrefix(strings.ToLower(v), kw) {
			needsQuote = true
		}
	}
	if strings.Contains(v, "\n") || strings.HasPrefix(v, ";") {
		return "\n;" + v + "\n;\n"
	}
	if !needsQuote {
		return v
	}
	if !strings.Contains(v, "'") {
		return "'" + v + "'"
	}
	if !strings.Contains(v, "\"") {
		return "\"" + v + "\""
	}
	return "\n;" + v + "\n;\n"
}
