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
	seqChains string
	seqOutput string
)

var extractSeqCmd = &cobra.Command{
	Use:   "extract-seq [flags] [input_file]",
	Short: "Extract sequences from chains in a PDB file",
	Long: `Extract sequences from chains in a PDB structure file.
The output is in FASTA format with sequence IDs in the format: >{pdbfilename_no_dotpdb}_{chain}

If no chains are specified, all chains will be extracted.
If no input file is specified, reads from stdin.

Examples:
  # Extract sequences from all chains
  pdbtk extract-seq 1a02.pdb > 1a02.fasta

  # Extract sequences from specific chains
  pdbtk extract-seq --chains A,B 1a02.pdb > 1a02_chainAB.fasta

  # Extract all chains to a file
  pdbtk extract-seq --output 1a02_all.fasta 1a02.pdb

  # Extract from stdin
  cat 1a02.pdb | pdbtk extract-seq --chains B,C`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExtractSeq,
}

func init() {
	extractSeqCmd.Flags().StringVarP(&seqChains, "chains", "c", "", "Comma-separated list of chain IDs to extract (default: all chains)")
	extractSeqCmd.Flags().StringVarP(&seqOutput, "output", "o", "", "Output file (default: stdout)")
}

func runExtractSeq(cmd *cobra.Command, args []string) error {
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

	// Parse chain IDs if specified
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

	// Read the PDB file
	var entry *pdb.Entry
	var err error
	if isStdin {
		content, err := readAllFromStdin()
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %v", err)
		}
		entry, err = readPDBFromContent(content)
	} else {
		entry, err = readPDB(inputFile)
	}
	if err != nil {
		return fmt.Errorf("failed to read PDB file: %v", err)
	}

	// Extract sequences
	sequences, err := extractSequencesPDB(entry, chainList)
	if err != nil {
		return fmt.Errorf("failed to extract sequences: %v", err)
	}

	// Write the output
	if seqOutput == "" || seqOutput == "-" {
		// Write to stdout
		return writeFASTAToWriter(sequences, os.Stdout, inputFile)
	} else {
		// Write to file
		file, err := os.Create(seqOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		return writeFASTAToWriter(sequences, file, inputFile)
	}
}

func extractSequencesPDB(entry *pdb.Entry, chainList []string) (map[string]string, error) {
	sequences := make(map[string]string)

	// If no chains specified, extract all chains
	if len(chainList) == 0 {
		for _, chain := range entry.Chains {
			sequence := extractChainSequence(chain)
			if sequence != "" {
				sequences[string(chain.Ident)] = sequence
			}
		}
	} else {
		// Extract specified chains
		validChains := make(map[byte]bool)
		for _, chainID := range chainList {
			if len(chainID) == 1 {
				validChains[chainID[0]] = true
			}
		}

		for _, chain := range entry.Chains {
			if validChains[chain.Ident] {
				sequence := extractChainSequence(chain)
				if sequence != "" {
					sequences[string(chain.Ident)] = sequence
				}
			}
		}
	}

	return sequences, nil
}

func extractChainSequence(chain *pdb.Chain) string {
	// Use the sequence from the chain directly
	var sequence strings.Builder
	for _, residue := range chain.Sequence {
		sequence.WriteByte(byte(residue))
	}
	return sequence.String()
}

func aminoAcidToLetter(residueName string) byte {
	// Map 3-letter amino acid codes to 1-letter codes
	switch strings.ToUpper(residueName) {
	case "ALA":
		return 'A'
	case "ARG":
		return 'R'
	case "ASN":
		return 'N'
	case "ASP":
		return 'D'
	case "CYS":
		return 'C'
	case "GLU":
		return 'E'
	case "GLN":
		return 'Q'
	case "GLY":
		return 'G'
	case "HIS":
		return 'H'
	case "ILE":
		return 'I'
	case "LEU":
		return 'L'
	case "LYS":
		return 'K'
	case "MET":
		return 'M'
	case "PHE":
		return 'F'
	case "PRO":
		return 'P'
	case "SER":
		return 'S'
	case "THR":
		return 'T'
	case "TRP":
		return 'W'
	case "TYR":
		return 'Y'
	case "VAL":
		return 'V'
	case "UNK":
		return 'X'
	default:
		return 'X' // Unknown amino acid
	}
}

func writeFASTAToWriter(sequences map[string]string, writer io.Writer, inputFile string) error {
	// Get base filename without extension
	var baseName string
	if inputFile == "" {
		baseName = "stdin"
	} else {
		baseName = strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	}

	// Write sequences in FASTA format
	for chainID, sequence := range sequences {
		fmt.Fprintf(writer, ">%s_%s\n", baseName, chainID)
		// Write sequence in lines of 80 characters
		for i := 0; i < len(sequence); i += 80 {
			end := i + 80
			if end > len(sequence) {
				end = len(sequence)
			}
			fmt.Fprintf(writer, "%s\n", sequence[i:end])
		}
	}

	return nil
}
