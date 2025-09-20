package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TuftsBCB/io/pdb"
	"github.com/spf13/cobra"
)

var (
	renumberStart           int
	renumberChain           string
	renumberForceSequential bool
	renumberExcludeZero     bool
	renumberOutput          string
)

var renumberResiduesCmd = &cobra.Command{
	Use:   "renumber-residues [flags] [input_file]",
	Short: "Renumber residues in a PDB file",
	Long: `Renumber residues in a PDB structure file starting from a specified number.
By default, this preserves gaps in the residue sequence but offsets the numbering.
Use --force-sequential to make all residues sequential without gaps.
Use --exclude-zero to skip residue number zero when using negative start values.

Examples:
  # Renumber all residues starting from 1
  pdbtk renumber-residues --start 1 1a02.pdb

  # Renumber residues in chain A starting from 1
  pdbtk renumber-residues --start 1 --chain A 1a02.pdb

  # Force sequential numbering starting from 1
  pdbtk renumber-residues --start 1 --force-sequential 1a02.pdb

  # Renumber starting from negative number
  pdbtk renumber-residues --start -10 1a02.pdb

  # Renumber starting from -1, skipping zero (goes -1, 1, 2, 3...)
  pdbtk renumber-residues --start -1 --exclude-zero 1a02.pdb

  # Renumber and output to a file
  pdbtk renumber-residues --start 1 --output 1a02_renumbered.pdb 1a02.pdb`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRenumberResidues,
}

func init() {
	renumberResiduesCmd.Flags().IntVarP(&renumberStart, "start", "s", 1, "Starting residue number (can be negative)")
	renumberResiduesCmd.Flags().StringVarP(&renumberChain, "chain", "c", "", "Chain ID to renumber (default: all chains)")
	renumberResiduesCmd.Flags().BoolVarP(&renumberForceSequential, "force-sequential", "f", false, "Force sequential numbering without gaps")
	renumberResiduesCmd.Flags().BoolVarP(&renumberExcludeZero, "exclude-zero", "z", false, "Skip residue number zero when using negative start values")
	renumberResiduesCmd.Flags().StringVarP(&renumberOutput, "output", "o", "", "Output file (default: stdout)")
}

func runRenumberResidues(cmd *cobra.Command, args []string) error {
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

	// Validate chain ID if specified
	if renumberChain != "" && len(renumberChain) != 1 {
		return fmt.Errorf("chain ID must be a single character, got: %s", renumberChain)
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

	// Renumber residues
	renumberedEntry, err := renumberResiduesPDB(entry, renumberStart, renumberChain, renumberForceSequential, renumberExcludeZero)
	if err != nil {
		return fmt.Errorf("failed to renumber residues: %v", err)
	}

	// Build the full command line
	commandLine := buildRenumberResiduesCommandLine(cmd, args, inputFile)

	// Write the output
	if renumberOutput == "" || renumberOutput == "-" {
		// Write to stdout
		return writePDBToWriter(renumberedEntry, os.Stdout, commandLine)
	} else {
		// Write to file
		file, err := os.Create(renumberOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		return writePDBToWriter(renumberedEntry, file, commandLine)
	}
}

func renumberResiduesPDB(entry *pdb.Entry, startNum int, chainID string, forceSequential bool, excludeZero bool) (*pdb.Entry, error) {
	// Create a new entry
	newEntry := &pdb.Entry{
		Path:   entry.Path,
		IdCode: entry.IdCode,
		Chains: make([]*pdb.Chain, 0, len(entry.Chains)),
		Scop:   entry.Scop,
		Cath:   entry.Cath,
	}

	// Determine which chains to process
	var targetChain byte
	if chainID != "" {
		targetChain = chainID[0]
	}

	// Process each chain
	for _, chain := range entry.Chains {
		// Skip chains that don't match the target chain (if specified)
		if chainID != "" && chain.Ident != targetChain {
			// Copy the chain unchanged
			newChain := copyChain(chain)
			newEntry.Chains = append(newEntry.Chains, newChain)
			continue
		}

		// Renumber this chain
		renumberedChain, err := renumberChainResidues(chain, startNum, forceSequential, excludeZero)
		if err != nil {
			return nil, fmt.Errorf("failed to renumber chain %c: %v", chain.Ident, err)
		}

		newEntry.Chains = append(newEntry.Chains, renumberedChain)
	}

	return newEntry, nil
}

func renumberChainResidues(chain *pdb.Chain, startNum int, forceSequential bool, excludeZero bool) (*pdb.Chain, error) {
	// Create a new chain
	newChain := &pdb.Chain{
		Ident:    chain.Ident,
		Sequence: chain.Sequence,
		Models:   make([]*pdb.Model, len(chain.Models)),
	}

	// Process each model
	for i, model := range chain.Models {
		newModel := &pdb.Model{
			Num:      model.Num,
			Residues: make([]*pdb.Residue, len(model.Residues)),
		}

		if forceSequential {
			// Force sequential numbering
			currentNum := startNum
			for j, residue := range model.Residues {
				// Skip zero if excludeZero is true and we would assign zero
				if excludeZero && currentNum == 0 {
					currentNum = 1
				}

				newResidue := &pdb.Residue{
					Name:        residue.Name,
					SequenceNum: currentNum,
				}
				newModel.Residues[j] = newResidue
				currentNum++
			}
		} else {
			// Preserve gaps but offset numbering
			if len(model.Residues) == 0 {
				newChain.Models[i] = newModel
				continue
			}

			// Find the minimum residue number to calculate offset
			minResNum := model.Residues[0].SequenceNum
			for _, residue := range model.Residues {
				if residue.SequenceNum < minResNum {
					minResNum = residue.SequenceNum
				}
			}

			// Calculate offset
			offset := startNum - minResNum

			// Apply offset to all residues
			for j, residue := range model.Residues {
				newResNum := residue.SequenceNum + offset

				// Skip zero if excludeZero is true and we would assign zero
				if excludeZero && newResNum == 0 {
					newResNum = 1
				}

				newResidue := &pdb.Residue{
					Name:        residue.Name,
					SequenceNum: newResNum,
				}
				newModel.Residues[j] = newResidue
			}
		}

		newChain.Models[i] = newModel
	}

	return newChain, nil
}

func copyChain(chain *pdb.Chain) *pdb.Chain {
	newChain := &pdb.Chain{
		Ident:    chain.Ident,
		Sequence: chain.Sequence,
		Models:   make([]*pdb.Model, len(chain.Models)),
	}

	for i, model := range chain.Models {
		newModel := &pdb.Model{
			Num:      model.Num,
			Residues: make([]*pdb.Residue, len(model.Residues)),
		}

		for j, residue := range model.Residues {
			newResidue := &pdb.Residue{
				Name:        residue.Name,
				SequenceNum: residue.SequenceNum,
			}
			newModel.Residues[j] = newResidue
		}

		newChain.Models[i] = newModel
	}

	return newChain
}

func buildRenumberResiduesCommandLine(cmd *cobra.Command, args []string, inputFile string) string {
	var parts []string

	// Add the command name
	parts = append(parts, "pdbtk", "renumber-residues")

	// Add flags
	parts = append(parts, "--start", strconv.Itoa(renumberStart))
	if renumberChain != "" {
		parts = append(parts, "--chain", renumberChain)
	}
	if renumberForceSequential {
		parts = append(parts, "--force-sequential")
	}
	if renumberExcludeZero {
		parts = append(parts, "--exclude-zero")
	}
	if renumberOutput != "" {
		parts = append(parts, "--output", renumberOutput)
	}

	// Add input file if not from stdin
	if inputFile != "" {
		parts = append(parts, inputFile)
	}

	return strings.Join(parts, " ")
}
