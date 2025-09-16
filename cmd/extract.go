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

func writePDBToWriter(entry *pdb.Entry, writer *os.File) error {
	// Simple PDB output - in a full implementation, you'd want to use proper PDB writers
	fmt.Fprintf(writer, "HEADER    EXTRACTED CHAINS FROM %s\n", entry.IdCode)
	fmt.Fprintf(writer, "REMARK    EXTRACTED BY PDBTK\n")

	atomSerial := 1
	for _, chain := range entry.Chains {
		for _, model := range chain.Models {
			fmt.Fprintf(writer, "MODEL        %d\n", model.Num)
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

					fmt.Fprintf(writer, "%s  %5d  %-4s%c%3s %c%4d%c   %8.3f%8.3f%8.3f  1.00 20.00           %c\n",
						recordType,
						atomSerial,
						atom.Name,
						' ',                  // alt loc
						string(residue.Name), // Convert seq.Residue to string
						chain.Ident,
						residue.SequenceNum,
						insertionCode,
						atom.Coords.X, atom.Coords.Y, atom.Coords.Z,
						' ', // element
					)
					atomSerial++
				}
			}
			fmt.Fprintf(writer, "ENDMDL\n")
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
