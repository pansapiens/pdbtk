package cmd

import (
	"github.com/spf13/cobra"

	"github.com/perry/pdbtk/pdbtk/structure"
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
	Short: "Renumber residues in a PDB or mmCIF file",
	Long: `Renumber residues in a PDB or PDBx/mmCIF structure file starting from a specified number.
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
	addInFormatFlag(renumberResiduesCmd)
	addOutFormatFlag(renumberResiduesCmd)
}

func runRenumberResidues(cmd *cobra.Command, args []string) error {
	inputFile, err := resolveInputPath(args)
	if err != nil {
		return err
	}

	s, err := readStructure(inputFile)
	if err != nil {
		return err
	}

	if renumberChain != "" {
		if err := checkChainsExist(s, []string{renumberChain}); err != nil {
			return err
		}
	}

	renumberResidues(s, renumberStart, renumberChain, renumberForceSequential, renumberExcludeZero)
	s.Renumber()

	outFormat, err := resolveOutputFormat(s, renumberOutput)
	if err != nil {
		return err
	}
	return writeStructure(s, renumberOutput, outFormat, buildCommandLine(cmd, args))
}

// renumberResidues rewrites residue numbers in place. Renumbering is applied
// per chain and per model, so every model of a chain ends up with matching
// numbering.
func renumberResidues(s *structure.Structure, start int, chainID string, sequential, excludeZero bool) {
	type chainModel struct {
		chain string
		model int
	}

	groups := make(map[chainModel][]*structure.Residue)
	var order []chainModel
	for _, r := range structure.Residues(s.Atoms) {
		if chainID != "" && r.ChainID != chainID {
			continue
		}
		k := chainModel{chain: r.ChainID, model: r.Model}
		if _, seen := groups[k]; !seen {
			order = append(order, k)
		}
		groups[k] = append(groups[k], r)
	}

	// Residue numbers feed the LINK/struct_conn endpoints too, so record the
	// mapping and apply it to connectivity afterwards.
	remap := make(map[structure.ResidueKey]int)

	for _, k := range order {
		residues := groups[k]
		if sequential {
			n := start
			for _, r := range residues {
				if excludeZero && n == 0 {
					n = 1
				}
				remap[r.ResidueKey] = n
				n++
			}
			continue
		}

		min := residues[0].ResSeq
		for _, r := range residues {
			if r.ResSeq < min {
				min = r.ResSeq
			}
		}
		offset := start - min
		for _, r := range residues {
			n := r.ResSeq + offset
			if excludeZero && n == 0 {
				n = 1
			}
			remap[r.ResidueKey] = n
		}
	}

	// Link endpoints carry no model number, so index the mapping without one.
	type linkKey struct {
		chain   string
		resSeq  int
		insCode string
	}
	linkRemap := make(map[linkKey]int, len(remap))
	for k, n := range remap {
		linkRemap[linkKey{chain: k.ChainID, resSeq: k.ResSeq, insCode: k.InsCode}] = n
	}
	applyLinkRemap := func(ref *structure.AtomRef) {
		if n, ok := linkRemap[linkKey{chain: ref.ChainID, resSeq: ref.ResSeq, insCode: ref.InsCode}]; ok {
			ref.ResSeq = n
		}
	}
	for i := range s.Links {
		applyLinkRemap(&s.Links[i].A)
		applyLinkRemap(&s.Links[i].B)
	}

	for _, a := range s.Atoms {
		if n, ok := remap[structure.KeyOf(a)]; ok {
			a.ResSeq = n
		}
	}
}
