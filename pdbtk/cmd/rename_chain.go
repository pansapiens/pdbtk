package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TuftsBCB/io/pdb"
	"github.com/spf13/cobra"
)

var (
	renameChainID   string
	renameToChainID string
	renameOutput    string
)

var renameChainCmd = &cobra.Command{
	Use:   "rename-chain [flags] <chain_id> [input_file]",
	Short: "Rename a chain in a PDB file",
	Long: `Rename a chain in a PDB structure file.
The chain ID must be a single character. The new chain ID must also be a single character.
If the specified chain does not exist, the command will exit with an error.
If the new chain ID already exists, a warning will be logged but the operation will continue.

Examples:
  # Rename chain A to B
  pdbtk rename-chain A --to B 1a02.pdb

  # Rename chain A to B and output to a file
  pdbtk rename-chain A --to B --output 1a02_renamed.pdb 1a02.pdb

  # Rename chain A to B from stdin
  cat 1a02.pdb | pdbtk rename-chain A --to B`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runRenameChain,
}

func init() {
	renameChainCmd.Flags().StringVarP(&renameToChainID, "to", "t", "", "New chain ID (required)")
	renameChainCmd.Flags().StringVarP(&renameOutput, "output", "o", "", "Output file (default: stdout)")

	renameChainCmd.MarkFlagRequired("to")
}

func runRenameChain(cmd *cobra.Command, args []string) error {
	// Get the chain ID to rename
	chainID := args[0]
	if len(chainID) != 1 {
		return fmt.Errorf("chain ID must be a single character, got: %s", chainID)
	}

	// Validate new chain ID
	if len(renameToChainID) != 1 {
		return fmt.Errorf("new chain ID must be a single character, got: %s", renameToChainID)
	}

	var inputFile string
	var isStdin bool

	if len(args) > 1 {
		inputFile = args[1]
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

	// Rename the chain
	renamedEntry, err := renameChainPDB(entry, chainID[0], renameToChainID[0])
	if err != nil {
		return fmt.Errorf("failed to rename chain: %v", err)
	}

	// Build the full command line
	commandLine := buildRenameChainCommandLine(cmd, args, inputFile)

	// Write the output
	if renameOutput == "" || renameOutput == "-" {
		// Write to stdout
		return writePDBToWriter(renamedEntry, os.Stdout, commandLine)
	} else {
		// Write to file
		file, err := os.Create(renameOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		return writePDBToWriter(renamedEntry, file, commandLine)
	}
}

func renameChainPDB(entry *pdb.Entry, oldChainID, newChainID byte) (*pdb.Entry, error) {
	// Create a new entry with the renamed chain
	newEntry := &pdb.Entry{
		Path:   entry.Path,
		IdCode: entry.IdCode,
		Chains: make([]*pdb.Chain, 0, len(entry.Chains)),
		Scop:   entry.Scop,
		Cath:   entry.Cath,
	}

	// Check if the old chain exists and if the new chain already exists
	oldChainExists := false
	newChainExists := false

	for _, chain := range entry.Chains {
		if chain.Ident == oldChainID {
			oldChainExists = true
		}
		if chain.Ident == newChainID {
			newChainExists = true
		}
	}

	// Error if old chain doesn't exist
	if !oldChainExists {
		return nil, fmt.Errorf("chain %c does not exist", oldChainID)
	}

	// Warning if new chain already exists
	if newChainExists {
		fmt.Fprintf(os.Stderr, "Warning: chain %c already exists, continuing anyway\n", newChainID)
	}

	// Copy chains with renamed chain
	for _, chain := range entry.Chains {
		newChain := &pdb.Chain{
			Ident:    chain.Ident,
			Sequence: chain.Sequence,
			Models:   make([]*pdb.Model, len(chain.Models)),
		}

		// Rename the chain if it matches the old chain ID
		if chain.Ident == oldChainID {
			newChain.Ident = newChainID
		}

		// Copy models
		for i, model := range chain.Models {
			newModel := &pdb.Model{
				Num:      model.Num,
				Residues: make([]*pdb.Residue, len(model.Residues)),
			}

			// Copy residues
			for j, residue := range model.Residues {
				newResidue := &pdb.Residue{
					Name:        residue.Name,
					SequenceNum: residue.SequenceNum,
				}
				newModel.Residues[j] = newResidue
			}

			newChain.Models[i] = newModel
		}

		newEntry.Chains = append(newEntry.Chains, newChain)
	}

	return newEntry, nil
}

func buildRenameChainCommandLine(cmd *cobra.Command, args []string, inputFile string) string {
	var parts []string

	// Add the command name
	parts = append(parts, "pdbtk", "rename-chain")

	// Add the chain ID
	parts = append(parts, args[0])

	// Add flags
	if renameToChainID != "" {
		parts = append(parts, "--to", renameToChainID)
	}
	if renameOutput != "" {
		parts = append(parts, "--output", renameOutput)
	}

	// Add input file if not from stdin
	if inputFile != "" {
		parts = append(parts, inputFile)
	}

	return strings.Join(parts, " ")
}
