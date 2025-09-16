package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TuftsBCB/io/pdb"
	"github.com/TuftsBCB/io/pdbx"
	"github.com/spf13/cobra"
)

var (
	chains string
	output string
	format string
)

var extractCmd = &cobra.Command{
	Use:   "extract [flags] <input_file>",
	Short: "Extract chains from a PDB or PDBx/mmCIF file",
	Long: `Extract specific chains from a PDB or PDBx/mmCIF structure file.
The output can be written to a file or stdout (if no output file is specified).

Examples:
  # Extract chains A, B, and C to a file
  pdbtk extract --chains A,B,C --output 1a02_chainABC.pdb 1a02.pdb

  # Extract chains A, B, and C to stdout
  pdbtk extract --chains A,B,C 1a02.pdb > 1a02_chainABC.pdb

  # Extract from PDBx/mmCIF file
  pdbtk extract --chains A,B --output 1a02_chainAB.cif 1a02.cif`,
	Args: cobra.ExactArgs(1),
	RunE: runExtract,
}

func init() {
	extractCmd.Flags().StringVarP(&chains, "chains", "c", "", "Comma-separated list of chain IDs to extract (required)")
	extractCmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: stdout)")
	extractCmd.Flags().StringVarP(&format, "format", "f", "auto", "Output format: auto, pdb, or cif (default: auto)")

	extractCmd.MarkFlagRequired("chains")
}

func runExtract(cmd *cobra.Command, args []string) error {
	inputFile := args[0]

	// Check if input file exists
	if err := checkFileExists(inputFile); err != nil {
		return err
	}

	// Parse chain IDs
	chainList := strings.Split(chains, ",")
	for i, chain := range chainList {
		chainList[i] = strings.TrimSpace(chain)
		if len(chainList[i]) != 1 {
			return fmt.Errorf("invalid chain ID: %s (must be single character)", chainList[i])
		}
	}

	// Determine input format
	inputExt := strings.ToLower(filepath.Ext(inputFile))
	var isPDBx bool
	switch inputExt {
	case ".cif", ".mmcif":
		isPDBx = true
	case ".pdb":
		isPDBx = false
	default:
		// Try to detect format by reading the file
		var err error
		isPDBx, err = detectFormat(inputFile)
		if err != nil {
			return fmt.Errorf("could not detect file format: %v", err)
		}
	}

	// Read the structure file
	var entry interface{}
	var err error

	if isPDBx {
		entry, err = readPDBx(inputFile)
	} else {
		entry, err = readPDB(inputFile)
	}

	if err != nil {
		return fmt.Errorf("failed to read structure file: %v", err)
	}

	// Extract chains
	extractedChains, err := extractChains(entry, chainList, isPDBx)
	if err != nil {
		return fmt.Errorf("failed to extract chains: %v", err)
	}

	// Determine output format
	outputFormat := format
	if outputFormat == "auto" {
		if isPDBx {
			outputFormat = "cif"
		} else {
			outputFormat = "pdb"
		}
	}

	// Write output
	if output == "" {
		// Write to stdout
		return writeToStdout(extractedChains, outputFormat)
	} else {
		// Write to file
		return writeToFile(extractedChains, output, outputFormat)
	}
}

func detectFormat(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Read first few lines to detect format
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return false, err
	}

	content := string(buffer[:n])

	// Check for PDBx/mmCIF indicators
	if strings.Contains(content, "data_") || strings.Contains(content, "loop_") {
		return true, nil
	}

	// Check for PDB indicators
	if strings.Contains(content, "HEADER") || strings.Contains(content, "ATOM") {
		return false, nil
	}

	return false, fmt.Errorf("unable to detect file format")
}

func readPDB(filename string) (*pdb.Entry, error) {
	return pdb.ReadPDB(filename)
}

func readPDBx(filename string) (*pdbx.Entry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return pdbx.Read(file)
}

func extractChains(entry interface{}, chainList []string, isPDBx bool) (interface{}, error) {
	if isPDBx {
		return extractChainsPDBx(entry.(*pdbx.Entry), chainList)
	} else {
		return extractChainsPDB(entry.(*pdb.Entry), chainList)
	}
}

func extractChainsPDB(entry *pdb.Entry, chainList []string) (*pdb.Entry, error) {
	// Create a new entry with only the specified chains
	newEntry := &pdb.Entry{
		Path:   entry.Path,
		IdCode: entry.IdCode,
		Chains: make([]*pdb.Chain, 0),
	}

	// Create a map for quick lookup
	chainMap := make(map[string]bool)
	for _, chainID := range chainList {
		chainMap[chainID] = true
	}

	// Filter chains
	for _, chain := range entry.Chains {
		if chainMap[string(chain.Ident)] {
			newEntry.Chains = append(newEntry.Chains, chain)
		}
	}

	if len(newEntry.Chains) == 0 {
		return nil, fmt.Errorf("no matching chains found")
	}

	return newEntry, nil
}

func extractChainsPDBx(entry *pdbx.Entry, chainList []string) (*pdbx.Entry, error) {
	// Create a new entry with only the specified chains
	newEntry := &pdbx.Entry{
		Id:         entry.Id,
		Title:      entry.Title,
		Descriptor: entry.Descriptor,
		Entities:   make(map[byte]*pdbx.Entity),
		CIF:        entry.CIF,
	}

	// Create a map for quick lookup
	chainMap := make(map[string]bool)
	for _, chainID := range chainList {
		chainMap[chainID] = true
	}

	// Filter entities and their chains
	for entityID, entity := range entry.Entities {
		newEntity := &pdbx.Entity{
			Entry:  newEntry,
			Id:     entity.Id,
			Type:   entity.Type,
			Seq:    entity.Seq,
			Chains: make(map[byte]*pdbx.Chain),
		}

		// Copy chains that match our filter
		for chainID, chain := range entity.Chains {
			if chainMap[string(chainID)] {
				newEntity.Chains[chainID] = chain
			}
		}

		// Only add entity if it has matching chains
		if len(newEntity.Chains) > 0 {
			newEntry.Entities[entityID] = newEntity
		}
	}

	if len(newEntry.Entities) == 0 {
		return nil, fmt.Errorf("no matching chains found")
	}

	return newEntry, nil
}

func writeToStdout(entry interface{}, format string) error {
	// For now, we'll implement a simple text output
	// In a full implementation, you'd want to use proper PDB/PDBx writers
	fmt.Fprintf(os.Stderr, "Writing %s format to stdout\n", format)

	if format == "pdb" {
		pdbEntry := entry.(*pdb.Entry)
		return writePDBToWriter(pdbEntry, os.Stdout)
	} else {
		pdbxEntry := entry.(*pdbx.Entry)
		return writePDBxToWriter(pdbxEntry, os.Stdout)
	}
}

func writeToFile(entry interface{}, filename, format string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if format == "pdb" {
		pdbEntry := entry.(*pdb.Entry)
		return writePDBToWriter(pdbEntry, file)
	} else {
		pdbxEntry := entry.(*pdbx.Entry)
		return writePDBxToWriter(pdbxEntry, file)
	}
}

// extractElementSymbol extracts the element symbol from an atom name
func extractElementSymbol(atomName string) string {
	// Remove leading digits and spaces, then take the first letter
	atomName = strings.TrimSpace(atomName)
	if len(atomName) == 0 {
		return ""
	}

	// Find the first alphabetic character
	for i, char := range atomName {
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') {
			// Take the first letter and optionally the second if it's lowercase
			if i+1 < len(atomName) && atomName[i+1] >= 'a' && atomName[i+1] <= 'z' {
				return strings.ToUpper(atomName[i : i+2])
			}
			return strings.ToUpper(string(char))
		}
	}

	// Fallback: return the first character if no alphabetic character found
	if len(atomName) > 0 {
		return strings.ToUpper(string(atomName[0]))
	}
	return ""
}

// singleLetterToResidue converts single-letter amino acid codes to three-letter codes
func singleLetterToResidue(singleLetter string) string {
	// Convert to uppercase for consistency
	singleLetter = strings.ToUpper(singleLetter)

	// Reverse mapping from single letter to three letter codes
	reverseMap := map[string]string{
		"A": "ALA", "R": "ARG", "N": "ASN", "D": "ASP", "C": "CYS",
		"Q": "GLN", "E": "GLU", "G": "GLY", "H": "HIS", "I": "ILE",
		"L": "LEU", "K": "LYS", "M": "MET", "F": "PHE", "P": "PRO",
		"S": "SER", "T": "THR", "W": "TRP", "Y": "TYR", "V": "VAL",
		// Modified amino acids and common variants
		"U": "SEC", "O": "PYL", // Selenocysteine and Pyrrolysine
		"X": "UNK", "J": "XLE", // Unknown amino acids
	}

	if threeLetter, exists := reverseMap[singleLetter]; exists {
		return threeLetter
	}

	// If not found, return UNK for unknown
	return "UNK"
}

// formatAtomName formats the atom name for PDB output according to spec.
// Columns 13-16: Atom name.
// Details: Element symbol right-justified in 13-14.
//
//	Trailing characters left-justified in 15-16.
//	Single-char element symbol should be in column 14, unless atom name is 4 chars.
func formatAtomName(atomName string) string {
	name := strings.TrimSpace(atomName)
	element := extractElementSymbol(name)

	// Rule: If an atom name has four characters, it must start in column 13
	if len(name) >= 4 {
		return fmt.Sprintf("%-4s", name)
	}

	// Rule: single-character element symbol should not appear in column 13
	if len(element) == 1 {
		// Place element in column 14.
		trailing := strings.TrimPrefix(name, element)
		return fmt.Sprintf(" %-1s%-2s", element, trailing)
	}

	// Rule: element symbols right-justified in columns 13-14.
	if len(element) == 2 {
		trailing := strings.TrimPrefix(name, element)
		return fmt.Sprintf("%-2s%-2s", element, trailing)
	}

	// Fallback for weird cases, just left-justify.
	return fmt.Sprintf("%-4s", name)
}

func writePDBToWriter(entry *pdb.Entry, writer *os.File) error {
	// Simple PDB output - in a full implementation, you'd want to use proper PDB writers
	fmt.Fprintf(writer, "HEADER    EXTRACTED CHAINS FROM %s\n", entry.IdCode)
	fmt.Fprintf(writer, "REMARK    EXTRACTED BY PDBTK\n")

	// Check if any chain has multiple models (ensemble) to determine if we need MODEL/ENDMDL records
	hasMultipleModels := false
	for _, chain := range entry.Chains {
		if len(chain.Models) > 1 {
			hasMultipleModels = true
			break
		}
	}

	atomSerial := 1
	for _, chain := range entry.Chains {
		for _, model := range chain.Models {
			// Only output MODEL record if we have multiple models (ensemble)
			if hasMultipleModels {
				fmt.Fprintf(writer, "MODEL        %d\n", model.Num)
			}

			for _, residue := range model.Residues {
				for _, atom := range residue.Atoms {
					recordType := "ATOM  "
					if atom.Het {
						recordType = "HETATM"
					}
					// Handle insertion code - use space if it's null byte
					insertionCode := residue.InsertionCode
					if insertionCode == 0 {
						insertionCode = ' '
					}

					formattedAtomName := formatAtomName(atom.Name)

					fmt.Fprintf(writer, "%-6s%5d %s%c%3s %c%4d%c   %8.3f%8.3f%8.3f%6.2f%6.2f          %2s\n",
						recordType,        // 1-6: "ATOM  " or "HETATM"
						atomSerial,        // 7-11: atom serial number
						formattedAtomName, // 13-16: atom name
						' ',               // 17: alternate location indicator
						singleLetterToResidue(string(residue.Name)), // 18-20: residue name
						chain.Ident,                                 // 22: chain identifier
						residue.SequenceNum,                         // 23-26: residue sequence number
						insertionCode,                               // 27: insertion code
						atom.Coords.X, atom.Coords.Y, atom.Coords.Z, // 31-38, 39-46, 47-54: coordinates
						1.00, 20.00, // 55-60, 61-66: occupancy and temperature factor
						extractElementSymbol(atom.Name), // 77-78: element symbol
					)
					atomSerial++
				}
			}

			// Only output ENDMDL record if we have multiple models (ensemble)
			if hasMultipleModels {
				fmt.Fprintf(writer, "ENDMDL\n")
			}
		}
	}

	fmt.Fprintf(writer, "END\n")
	return nil
}

func writePDBxToWriter(entry *pdbx.Entry, writer *os.File) error {
	// Simple PDBx output - in a full implementation, you'd want to use proper PDBx writers
	fmt.Fprintf(writer, "data_%s\n", entry.Id)
	fmt.Fprintf(writer, "#\n")
	fmt.Fprintf(writer, "_entry.id %s\n", entry.Id)
	if entry.Title != "" {
		fmt.Fprintf(writer, "_struct.title %s\n", entry.Title)
	}
	fmt.Fprintf(writer, "#\n")
	fmt.Fprintf(writer, "loop_\n")
	fmt.Fprintf(writer, "_atom_site.group_PDB\n")
	fmt.Fprintf(writer, "_atom_site.id\n")
	fmt.Fprintf(writer, "_atom_site.type_symbol\n")
	fmt.Fprintf(writer, "_atom_site.label_atom_id\n")
	fmt.Fprintf(writer, "_atom_site.label_alt_id\n")
	fmt.Fprintf(writer, "_atom_site.label_comp_id\n")
	fmt.Fprintf(writer, "_atom_site.label_asym_id\n")
	fmt.Fprintf(writer, "_atom_site.label_entity_id\n")
	fmt.Fprintf(writer, "_atom_site.label_seq_id\n")
	fmt.Fprintf(writer, "_atom_site.pdbx_PDB_ins_code\n")
	fmt.Fprintf(writer, "_atom_site.Cartn_x\n")
	fmt.Fprintf(writer, "_atom_site.Cartn_y\n")
	fmt.Fprintf(writer, "_atom_site.Cartn_z\n")
	fmt.Fprintf(writer, "_atom_site.occupancy\n")
	fmt.Fprintf(writer, "_atom_site.B_iso_or_equiv\n")
	fmt.Fprintf(writer, "_atom_site.pdbx_formal_charge\n")
	fmt.Fprintf(writer, "_atom_site.auth_seq_id\n")
	fmt.Fprintf(writer, "_atom_site.auth_comp_id\n")
	fmt.Fprintf(writer, "_atom_site.auth_asym_id\n")
	fmt.Fprintf(writer, "_atom_site.auth_atom_id\n")
	fmt.Fprintf(writer, "_atom_site.pdbx_PDB_model_num\n")

	atomID := 1
	for _, entity := range entry.Entities {
		for _, chain := range entity.Chains {
			for _, model := range chain.Models {
				for _, site := range model.Sites {
					for _, atom := range site.Atoms {
						group := "ATOM"
						if atom.Het {
							group = "HETATM"
						}
						fmt.Fprintf(writer, "%s %d %s %s . %s %c %c %d . %.3f %.3f %.3f 1.00 20.00 ? %d %s %c %s %d\n",
							group,
							atomID,
							"", // type_symbol
							atom.Name,
							site.Comp,
							chain.Id,
							entity.Id,
							site.SeqIndex+1,
							atom.Coords.X, atom.Coords.Y, atom.Coords.Z,
							site.SeqIndex+1,
							site.Comp,
							chain.Id,
							atom.Name,
							model.Num,
						)
						atomID++
					}
				}
			}
		}
	}

	return nil
}
