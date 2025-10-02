package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/TuftsBCB/io/pdb"
	"github.com/spf13/cobra"
)

var (
	chains string
	output string
	altloc string
)

var extractCmd = &cobra.Command{
	Use:   "extract [flags] [input_file]",
	Short: "Extract chains from a PDB file",
	Long: `Extract specific chains from a PDB structure file.
The output can be written to a file or stdout (if no output file is specified).
If no input file is specified, reads from stdin.

Examples:
  # Extract chains A, B, and C to a file
  pdbtk extract --chains A,B,C --output 1a02_chainABC.pdb 1a02.pdb

  # Extract chains A, B, and C to stdout
  pdbtk extract --chains A,B,C 1a02.pdb > 1a02_chainABC.pdb

  # Extract from stdin
  cat 1a02.pdb | pdbtk extract --chains A,B,C

  # Extract only ALTLOC A atoms
  pdbtk extract --chains A --altloc A 1a02.pdb

  # Extract first ALTLOC when duplicates exist
  pdbtk extract --chains A --altloc first 1a02.pdb`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExtract,
}

func init() {
	extractCmd.Flags().StringVarP(&chains, "chains", "c", "", "Comma-separated list of chain IDs to extract")
	extractCmd.Flags().StringVar(&chains, "chain", "", "Alias for --chains")
	extractCmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: stdout)")
	extractCmd.Flags().StringVar(&altloc, "altloc", "", "Filter by ALTLOC identifier (e.g., A, B) or 'first' to take first ALTLOC when duplicates exist")
}

func runExtract(cmd *cobra.Command, args []string) error {
	var inputFile string
	var isStdin bool

	if len(args) > 0 {
		inputFile = args[0]
		isStdin = false
		// Check if input file exists
		if err := CheckFileExists(inputFile); err != nil {
			return err
		}
		// Check if it's a PDB file
		inputExt := strings.ToLower(filepath.Ext(inputFile))
		if inputExt != ".pdb" {
			return fmt.Errorf("only PDB files are supported, got: %s", inputExt)
		}
	} else {
		// Check if stdin is available
		stat, err := os.Stdin.Stat()
		if err != nil {
			return fmt.Errorf("failed to check stdin: %v", err)
		}
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return fmt.Errorf("no input file specified and stdin is not available")
		}
		inputFile = ""
		isStdin = true
	}

	// Validate that at least one of --chains or --altloc is specified
	if chains == "" && altloc == "" {
		return fmt.Errorf("at least one of --chains or --altloc must be specified")
	}

	// Parse chain IDs
	var chainList []string
	if chains != "" {
		chainList = strings.Split(chains, ",")
		for i, chain := range chainList {
			chainList[i] = strings.TrimSpace(chain)
			if len(chainList[i]) != 1 {
				return fmt.Errorf("invalid chain ID: %s (must be single character)", chainList[i])
			}
		}
	}

	// Read the PDB file with ALTLOC support
	var entry *pdb.Entry
	var altLocList []byte
	var err error
	if isStdin {
		content, err := readAllFromStdin()
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %v", err)
		}
		extendedEntry, err := ReadPDBWithAltLocFromContent(content, "")
		if err != nil {
			return fmt.Errorf("failed to read PDB file: %v", err)
		}
		entry = extendedEntry.Entry
		altLocList = extendedEntry.AltLocList
	} else {
		extendedEntry, err := ReadPDBWithAltLoc(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read PDB file: %v", err)
		}
		entry = extendedEntry.Entry
		altLocList = extendedEntry.AltLocList
	}

	// Extract the specified chains (if specified)
	var extractedChains *pdb.Entry
	if len(chainList) > 0 {
		extractedChains, altLocList, err = ExtractChainsPDB(entry, chainList, altLocList)
		if err != nil {
			return fmt.Errorf("failed to extract chains: %v", err)
		}
	} else {
		// No chain filtering, use all chains
		extractedChains = entry
	}

	// Apply ALTLOC filtering if specified
	if altloc != "" {
		extractedChains, altLocList, err = filterByAltLoc(extractedChains, altLocList, altloc)
		if err != nil {
			return fmt.Errorf("failed to filter by ALTLOC: %v", err)
		}
	}

	// Build the full command line
	commandLine := buildCommandLine(cmd, args, inputFile)

	// Write the output
	if output == "" || output == "-" {
		// Write to stdout
		return writePDBToWriterWithAltLoc(extractedChains, altLocList, os.Stdout, commandLine)
	} else {
		// Write to file
		file, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		return writePDBToWriterWithAltLoc(extractedChains, altLocList, file, commandLine)
	}
}

func readPDB(filename string) (*pdb.Entry, error) {
	return pdb.ReadPDB(filename)
}

func readPDBFromContent(content []byte) (*pdb.Entry, error) {
	// Create a temporary file to read from content
	tmpfile, err := os.CreateTemp("", "pdbtk_*.pdb")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		tmpfile.Close()
		return nil, err
	}
	tmpfile.Close()

	return pdb.ReadPDB(tmpfile.Name())
}

func ExtractChainsPDB(entry *pdb.Entry, chainList []string, altLocList []byte) (*pdb.Entry, []byte, error) {
	// Create a new entry with only the specified chains
	newEntry := &pdb.Entry{
		Path:   entry.Path,
		IdCode: entry.IdCode,
		Chains: make([]*pdb.Chain, 0),
		Scop:   entry.Scop,
		Cath:   entry.Cath,
	}

	// Create a set of valid chain IDs for quick lookup
	validChains := make(map[byte]bool)
	for _, chainID := range chainList {
		if len(chainID) == 1 {
			validChains[chainID[0]] = true
		}
	}

	// Filter chains and corresponding ALTLOC information
	newAltLocList := make([]byte, 0)
	atomIndex := 0

	for _, chain := range entry.Chains {
		// Count atoms in this chain
		atomCount := 0
		for _, model := range chain.Models {
			for _, residue := range model.Residues {
				atomCount += len(residue.Atoms)
			}
		}

		if validChains[chain.Ident] {
			// Include this chain
			newEntry.Chains = append(newEntry.Chains, chain)
			// Copy the corresponding ALTLOC entries
			if altLocList != nil && atomIndex+atomCount <= len(altLocList) {
				newAltLocList = append(newAltLocList, altLocList[atomIndex:atomIndex+atomCount]...)
			}
		}
		atomIndex += atomCount
	}

	return newEntry, newAltLocList, nil
}

// filterByAltLoc filters atoms based on ALTLOC criteria
func filterByAltLoc(entry *pdb.Entry, altLocList []byte, altlocFilter string) (*pdb.Entry, []byte, error) {
	// Create a new entry with filtered atoms
	filteredEntry := &pdb.Entry{
		Path:   entry.Path,
		IdCode: entry.IdCode,
		Chains: make([]*pdb.Chain, 0),
		Scop:   entry.Scop,
		Cath:   entry.Cath,
	}

	newAltLocList := make([]byte, 0)
	atomIndex := 0

	for _, chain := range entry.Chains {
		newChain := &pdb.Chain{
			Entry:    filteredEntry,
			Ident:    chain.Ident,
			SeqType:  chain.SeqType,
			Sequence: chain.Sequence,
			Models:   make([]*pdb.Model, 0),
			Missing:  chain.Missing,
		}

		for _, model := range chain.Models {
			newModel := &pdb.Model{
				Entry:    filteredEntry,
				Chain:    newChain,
				Num:      model.Num,
				Residues: make([]*pdb.Residue, 0),
			}

			for _, residue := range model.Residues {
				newResidue := &pdb.Residue{
					Name:          residue.Name,
					SequenceNum:   residue.SequenceNum,
					InsertionCode: residue.InsertionCode,
					Atoms:         make([]pdb.Atom, 0),
				}

				// Group atoms by name to detect duplicates
				type atomWithIndex struct {
					atom   pdb.Atom
					index  int
					altLoc byte
				}
				atomGroups := make(map[string][]atomWithIndex)

				// Group atoms by name
				for _, atom := range residue.Atoms {
					var altLoc byte = ' '
					if atomIndex < len(altLocList) {
						altLoc = altLocList[atomIndex]
					}
					atomGroups[atom.Name] = append(atomGroups[atom.Name], atomWithIndex{
						atom:   atom,
						index:  atomIndex,
						altLoc: altLoc,
					})
					atomIndex++
				}

				// Apply ALTLOC filtering
				for _, group := range atomGroups {
					if altlocFilter == "first" {
						// Take the first ALTLOC when duplicates exist
						if len(group) > 1 {
							// Find the first atom with a non-space ALTLOC, or take the first atom
							selectedIdx := 0
							for i, atomInfo := range group {
								if atomInfo.altLoc != ' ' {
									selectedIdx = i
									break
								}
							}
							newResidue.Atoms = append(newResidue.Atoms, group[selectedIdx].atom)
							newAltLocList = append(newAltLocList, group[selectedIdx].altLoc)
						} else {
							// Only one atom, keep it
							newResidue.Atoms = append(newResidue.Atoms, group[0].atom)
							newAltLocList = append(newAltLocList, group[0].altLoc)
						}
					} else {
						// Filter by specific ALTLOC identifier
						targetAltLoc := altlocFilter[0]
						for _, atomInfo := range group {
							if atomInfo.altLoc == targetAltLoc || atomInfo.altLoc == ' ' {
								newResidue.Atoms = append(newResidue.Atoms, atomInfo.atom)
								newAltLocList = append(newAltLocList, atomInfo.altLoc)
							}
						}
					}
				}

				// Only add residue if it has atoms
				if len(newResidue.Atoms) > 0 {
					newModel.Residues = append(newModel.Residues, newResidue)
				}
			}

			// Only add model if it has residues
			if len(newModel.Residues) > 0 {
				newChain.Models = append(newChain.Models, newModel)
			}
		}

		// Only add chain if it has models
		if len(newChain.Models) > 0 {
			filteredEntry.Chains = append(filteredEntry.Chains, newChain)
		}
	}

	return filteredEntry, newAltLocList, nil
}

func readAllFromStdin() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}

func buildCommandLine(cmd *cobra.Command, args []string, inputFile string) string {
	var parts []string

	// Add the command name
	parts = append(parts, "pdbtk", "extract")

	// Add flags
	if chains != "" {
		parts = append(parts, "--chain", chains)
	}
	if output != "" {
		parts = append(parts, "--output", output)
	}
	if altloc != "" {
		parts = append(parts, "--altloc", altloc)
	}

	// Add input file if not from stdin
	if inputFile != "" {
		parts = append(parts, inputFile)
	}

	return strings.Join(parts, " ")
}
