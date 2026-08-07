package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/perry/pdbtk/pdbtk/structure"
)

var (
	renameToChainID string
	renameOutput    string
)

var renameChainCmd = &cobra.Command{
	Use:   "rename-chain [flags] <chain_id> [input_file]",
	Short: "Rename a chain in a PDB or mmCIF file",
	Long: `Rename a chain in a PDB or PDBx/mmCIF structure file.
If the specified chain does not exist, the command will exit with an error.
If the new chain ID already exists, a warning will be logged but the operation will continue.

Chain IDs longer than one character are allowed for mmCIF output; writing them
to a PDB file requires --force-lossy-pdb.

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
	addInFormatFlag(renameChainCmd)
	addOutFormatFlag(renameChainCmd)

	renameChainCmd.MarkFlagRequired("to")
}

func runRenameChain(cmd *cobra.Command, args []string) error {
	oldChainID := args[0]
	if renameToChainID == "" {
		return fmt.Errorf("new chain ID must not be empty")
	}

	inputFile, err := resolveInputPath(args[1:])
	if err != nil {
		return err
	}

	s, err := readStructure(inputFile)
	if err != nil {
		return err
	}

	if err := renameChain(s, oldChainID, renameToChainID); err != nil {
		return err
	}
	s.Renumber()

	outFormat, err := resolveOutputFormat(s, renameOutput)
	if err != nil {
		return err
	}
	return writeStructure(s, renameOutput, outFormat, buildCommandLine(cmd, args))
}

func renameChain(s *structure.Structure, from, to string) error {
	var found, collides bool
	for _, id := range s.ChainIDs() {
		if id == from {
			found = true
		}
		if id == to {
			collides = true
		}
	}
	if !found {
		return fmt.Errorf("chain %s does not exist (available: %v)", from, s.ChainIDs())
	}
	if collides {
		fmt.Fprintf(os.Stderr, "Warning: chain %s already exists, continuing anyway\n", to)
	}

	for _, a := range s.Atoms {
		if a.ChainID == from {
			a.ChainID = to
		}
	}
	for i := range s.Links {
		if s.Links[i].A.ChainID == from {
			s.Links[i].A.ChainID = to
		}
		if s.Links[i].B.ChainID == from {
			s.Links[i].B.ChainID = to
		}
	}
	for i := range s.SeqRes {
		if s.SeqRes[i].ChainID == from {
			s.SeqRes[i].ChainID = to
		}
	}
	return nil
}
