package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/perry/pdbtk/pdbtk/structure"
)

var (
	chains     string
	output     string
	altloc     string
	keepHetatm bool
	keepWaters bool
	noHetatm   bool
)

var extractCmd = &cobra.Command{
	Use:   "extract [flags] [input_file]",
	Short: "Extract chains from a PDB or mmCIF file",
	Long: `Extract specific chains from a PDB or PDBx/mmCIF structure file.
The output can be written to a file or stdout (if no output file is specified).
If no input file is specified, reads from stdin.

By default hetero atoms (ligands, ions, modified residues) belonging to the
selected chains are kept and waters are dropped. Use --keep-waters to retain
waters, or --no-hetatm to drop hetero atoms as well.

Examples:
  # Extract chains A, B, and C to a file
  pdbtk extract --chains A,B,C --output 1a02_chainABC.pdb 1a02.pdb

  # Extract chains A, B, and C to stdout
  pdbtk extract --chains A,B,C 1a02.pdb > 1a02_chainABC.pdb

  # Extract from an mmCIF file, writing mmCIF
  pdbtk extract --chains A,B --output 1a02_chainAB.cif 1a02.cif

  # Convert mmCIF to PDB
  pdbtk extract --chains A --out-format pdb 1a02.cif > 1a02_chainA.pdb

  # Extract from stdin
  cat 1a02.pdb | pdbtk extract --chains A,B,C

  # Extract only ALTLOC A atoms
  pdbtk extract --chains A --altloc A 1a02.pdb

  # Extract first ALTLOC when duplicates exist
  pdbtk extract --chains A --altloc first 1a02.pdb

  # Keep only the polymer, dropping ligands and waters
  pdbtk extract --chains A --no-hetatm 1a02.pdb

  # Include waters alongside hetero atoms
  pdbtk extract --chains A --keep-waters 1a02.pdb`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExtract,
}

func init() {
	extractCmd.Flags().StringVarP(&chains, "chains", "c", "", "Comma-separated list of chain IDs to extract")
	extractCmd.Flags().StringVar(&chains, "chain", "", "Alias for --chains")
	extractCmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: stdout)")
	extractCmd.Flags().StringVar(&altloc, "altloc", "", "Filter by ALTLOC identifier (e.g., A, B) or 'first' to take first ALTLOC when duplicates exist")
	extractCmd.Flags().BoolVar(&keepHetatm, "keep-hetatm", false, "Retain hetero atoms (the default; accepted for backwards compatibility)")
	extractCmd.Flags().BoolVar(&keepWaters, "keep-waters", false, "Retain waters matching the extraction selection")
	extractCmd.Flags().BoolVar(&noHetatm, "no-hetatm", false, "Drop all hetero atoms, keeping only ATOM records")
	addInFormatFlag(extractCmd)
	addOutFormatFlag(extractCmd)
}

func runExtract(cmd *cobra.Command, args []string) error {
	inputFile, err := resolveInputPath(args)
	if err != nil {
		return err
	}
	if keepHetatm && noHetatm {
		return fmt.Errorf("--keep-hetatm and --no-hetatm are mutually exclusive")
	}

	s, err := readStructure(inputFile)
	if err != nil {
		return err
	}

	chainList := splitChainList(chains)
	if len(chainList) > 0 {
		if err := checkChainsExist(s, chainList); err != nil {
			return err
		}
	}

	s = filterAtoms(s, chainList)
	if altloc != "" {
		s = filterByAltLoc(s, altloc)
	}
	if len(s.Atoms) == 0 {
		return fmt.Errorf("no atoms left after filtering")
	}
	s.Renumber()

	outFormat, err := resolveOutputFormat(s, output)
	if err != nil {
		return err
	}
	return writeStructure(s, output, outFormat, buildCommandLine(cmd, args))
}

func checkChainsExist(s *structure.Structure, chainList []string) error {
	present := make(map[string]bool)
	for _, id := range s.ChainIDs() {
		present[id] = true
	}
	var missing []string
	for _, id := range chainList {
		if !present[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("chain(s) not found in input: %s (available: %s)",
			strings.Join(missing, ", "), strings.Join(s.ChainIDs(), ", "))
	}
	return nil
}

// filterAtoms applies the chain selection and the hetero/water policy.
func filterAtoms(s *structure.Structure, chainList []string) *structure.Structure {
	wanted := make(map[string]bool, len(chainList))
	for _, id := range chainList {
		wanted[id] = true
	}

	return s.FilterAtoms(func(a *structure.Atom) bool {
		if len(wanted) > 0 && !wanted[a.ChainID] {
			return false
		}
		if a.IsWater() {
			return keepWaters
		}
		if a.Hetatm && noHetatm {
			return false
		}
		return true
	})
}

// filterByAltLoc keeps one alternate conformation per atom position. Atoms are
// grouped by residue and atom name; a blank ALTLOC means the atom has no
// alternates and is always kept.
func filterByAltLoc(s *structure.Structure, filter string) *structure.Structure {
	type groupKey struct {
		structure.ResidueKey
		atomName string
	}

	groups := make(map[groupKey][]*structure.Atom)
	for _, a := range s.Atoms {
		k := groupKey{ResidueKey: structure.KeyOf(a), atomName: strings.TrimSpace(a.Name)}
		groups[k] = append(groups[k], a)
	}

	selected := make(map[*structure.Atom]bool)
	for _, group := range groups {
		if filter == "first" {
			pick := group[0]
			for _, a := range group {
				if a.AltLoc != "" {
					pick = a
					break
				}
			}
			selected[pick] = true
			continue
		}
		for _, a := range group {
			if a.AltLoc == "" || strings.EqualFold(a.AltLoc, filter) {
				selected[a] = true
			}
		}
	}

	return s.FilterAtoms(func(a *structure.Atom) bool { return selected[a] })
}
