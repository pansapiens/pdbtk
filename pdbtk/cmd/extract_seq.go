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
	seqChains string
	seqOutput string
)

var extractSeqCmd = &cobra.Command{
	Use:   "extract-seq [flags] <input_file>",
	Short: "Extract sequences from chains in a PDB or PDBx/mmCIF file",
	Long: `Extract sequences from chains in a PDB or PDBx/mmCIF structure file.
The output is in FASTA format with sequence IDs in the format: >{pdbfilename_no_dotpdb}_{chain}

If no chains are specified, all chains will be extracted.

Examples:
  # Extract sequences from all chains
  pdbtk extract-seq 1a02.pdb > 1a02.fasta

  # Extract sequences from specific chains A, B, and C
  pdbtk extract-seq --chains A,B,C 1a02.pdb > 1a02_chainABC.fasta

  # Extract from PDBx/mmCIF file
  pdbtk extract-seq --chains A,B --output 1a02_chainAB.fasta 1a02.cif`,
	Args: cobra.ExactArgs(1),
	RunE: runExtractSeq,
}

func init() {
	extractSeqCmd.Flags().StringVarP(&seqChains, "chains", "c", "", "Comma-separated list of chain IDs to extract (default: all chains)")
	extractSeqCmd.Flags().StringVarP(&seqOutput, "output", "o", "", "Output file (default: stdout)")
}

func runExtractSeq(cmd *cobra.Command, args []string) error {
	inputFile := args[0]

	// Check if input file exists
	if err := CheckFileExists(inputFile); err != nil {
		return err
	}

	// Parse chain IDs (if provided)
	var chainList []string
	if seqChains != "" {
		chainList = strings.Split(seqChains, ",")
		for i, chain := range chainList {
			chainList[i] = strings.TrimSpace(chain)
			if len(chainList[i]) != 1 {
				return fmt.Errorf("invalid chain ID: %s (must be single character)", chainList[i])
			}
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

	// Generate base filename for sequence IDs (remove .pdb/.cif extension)
	baseFilename := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))

	// Extract sequences
	sequences, err := extractSequences(entry, chainList, baseFilename, isPDBx)
	if err != nil {
		return fmt.Errorf("failed to extract sequences: %v", err)
	}

	// Write output
	if seqOutput == "" {
		// Write to stdout
		return writeSequencesToStdout(sequences)
	} else {
		// Write to file
		return writeSequencesToFile(sequences, seqOutput)
	}
}

func extractSequences(entry interface{}, chainList []string, baseFilename string, isPDBx bool) ([]Sequence, error) {
	if isPDBx {
		return extractSequencesPDBx(entry.(*pdbx.Entry), chainList, baseFilename)
	} else {
		return extractSequencesPDB(entry.(*pdb.Entry), chainList, baseFilename)
	}
}

// Sequence represents a FASTA sequence with ID and sequence data
type Sequence struct {
	ID       string
	Sequence string
}

func extractSequencesPDB(entry *pdb.Entry, chainList []string, baseFilename string) ([]Sequence, error) {
	var sequences []Sequence

	// If no specific chains requested, extract all chains
	if len(chainList) == 0 {
		for _, chain := range entry.Chains {
			sequence := extractSequenceFromPDBChain(chain)
			seqID := fmt.Sprintf("%s_%c", baseFilename, chain.Ident)
			sequences = append(sequences, Sequence{
				ID:       seqID,
				Sequence: sequence,
			})
		}
	} else {
		// Create a map for quick lookup
		chainMap := make(map[string]bool)
		for _, chainID := range chainList {
			chainMap[chainID] = true
		}

		// Extract sequences from matching chains
		for _, chain := range entry.Chains {
			if chainMap[string(chain.Ident)] {
				sequence := extractSequenceFromPDBChain(chain)
				seqID := fmt.Sprintf("%s_%c", baseFilename, chain.Ident)
				sequences = append(sequences, Sequence{
					ID:       seqID,
					Sequence: sequence,
				})
			}
		}
	}

	if len(sequences) == 0 {
		return nil, fmt.Errorf("no chains found")
	}

	return sequences, nil
}

func extractSequencesPDBx(entry *pdbx.Entry, chainList []string, baseFilename string) ([]Sequence, error) {
	var sequences []Sequence

	// If no specific chains requested, extract all chains
	if len(chainList) == 0 {
		for _, entity := range entry.Entities {
			for chainID, chain := range entity.Chains {
				sequence := extractSequenceFromPDBxChain(chain)
				seqID := fmt.Sprintf("%s_%c", baseFilename, chainID)
				sequences = append(sequences, Sequence{
					ID:       seqID,
					Sequence: sequence,
				})
			}
		}
	} else {
		// Create a map for quick lookup
		chainMap := make(map[string]bool)
		for _, chainID := range chainList {
			chainMap[chainID] = true
		}

		// Extract sequences from matching chains
		for _, entity := range entry.Entities {
			for chainID, chain := range entity.Chains {
				if chainMap[string(chainID)] {
					sequence := extractSequenceFromPDBxChain(chain)
					seqID := fmt.Sprintf("%s_%c", baseFilename, chainID)
					sequences = append(sequences, Sequence{
						ID:       seqID,
						Sequence: sequence,
					})
				}
			}
		}
	}

	if len(sequences) == 0 {
		return nil, fmt.Errorf("no chains found")
	}

	return sequences, nil
}

func extractSequenceFromPDBChain(chain *pdb.Chain) string {
	var sequence strings.Builder

	// Get the first model (assuming single model for sequence extraction)
	if len(chain.Models) > 0 {
		model := chain.Models[0]
		for _, residue := range model.Residues {
			// residue.Name is already a single letter code
			sequence.WriteString(string(residue.Name))
		}
	}

	return sequence.String()
}

func extractSequenceFromPDBxChain(chain *pdbx.Chain) string {
	var sequence strings.Builder

	// Get the first model (assuming single model for sequence extraction)
	if len(chain.Models) > 0 {
		model := chain.Models[0]
		for _, site := range model.Sites {
			// site.Comp might be three-letter code, so convert it
			aa := residueToSingleLetter(site.Comp)
			sequence.WriteString(aa)
		}
	}

	return sequence.String()
}

// residueToSingleLetter converts three-letter amino acid codes to single letter codes
func residueToSingleLetter(residue string) string {
	// Convert to uppercase for consistency
	residue = strings.ToUpper(residue)

	// Standard amino acid mapping
	aaMap := map[string]string{
		"ALA": "A", "ARG": "R", "ASN": "N", "ASP": "D", "CYS": "C",
		"GLN": "Q", "GLU": "E", "GLY": "G", "HIS": "H", "ILE": "I",
		"LEU": "L", "LYS": "K", "MET": "M", "PHE": "F", "PRO": "P",
		"SER": "S", "THR": "T", "TRP": "W", "TYR": "Y", "VAL": "V",
		// Modified amino acids and common variants
		"SEC": "U", "PYL": "O", // Selenocysteine and Pyrrolysine
		"UNK": "X", "XAA": "X", "XLE": "J", // Unknown amino acids
		// DNA/RNA nucleotides
		"DA": "A", "DT": "T", "DC": "C", "DG": "G",
		"A": "A", "T": "T", "C": "C", "G": "G", "U": "U",
		// RNA nucleotides
		"RA": "A", "RU": "U", "RC": "C", "RG": "G",
		// Common non-standard amino acids
		"ASX": "B", "GLX": "Z", // Asparagine or Aspartic acid, Glutamine or Glutamic acid
		"CSO": "C", "HIP": "H", "HID": "H", "HIE": "H", // Modified cysteines and histidines
		// Water and ions
		"HOH": "X", "WAT": "X", // Water molecules
		"ZN": "X", "CA": "X", "MG": "X", "FE": "X", "MN": "X", // Metal ions
	}

	if singleLetter, exists := aaMap[residue]; exists {
		return singleLetter
	}

	// If not found, return X for unknown
	return "X"
}

func writeSequencesToStdout(sequences []Sequence) error {
	for _, seq := range sequences {
		fmt.Fprintf(os.Stdout, ">%s\n", seq.ID)
		// Write sequence on single line
		fmt.Fprintf(os.Stdout, "%s\n", seq.Sequence)
	}
	return nil
}

func writeSequencesToFile(sequences []Sequence, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, seq := range sequences {
		fmt.Fprintf(file, ">%s\n", seq.ID)
		// Write sequence on single line
		fmt.Fprintf(file, "%s\n", seq.Sequence)
	}
	return nil
}
