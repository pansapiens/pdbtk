package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/perry/pdbtk/pdbtk/structure"
)

var (
	seqChains string
	seqOutput string
	useSeqRes bool
)

var extractSeqCmd = &cobra.Command{
	Use:   "extract-seq [flags] [input_file]",
	Short: "Extract sequences from chains in a PDB or mmCIF file",
	Long: `Extract sequences from chains in a PDB or PDBx/mmCIF structure file.
The output is in FASTA format with sequence IDs in the format: >{filename_no_ext}_{chain}

If no chains are specified, all chains will be extracted.
If no input file is specified, reads from stdin.

Examples:
  # Extract sequences from all chains
  pdbtk extract-seq 1a02.pdb > 1a02.fasta

  # Extract sequences from specific chains
  pdbtk extract-seq --chains A,B 1a02.pdb > 1a02_chainAB.fasta

  # Extract from an mmCIF file
  pdbtk extract-seq --chains A,B --output 1a02_chainAB.fasta 1a02.cif

  # Extract all chains to a file
  pdbtk extract-seq --output 1a02_all.fasta 1a02.pdb

  # Extract from stdin
  cat 1a02.pdb | pdbtk extract-seq --chains B,C`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExtractSeq,
}

func init() {
	extractSeqCmd.Flags().StringVarP(&seqChains, "chains", "c", "", "Comma-separated list of chain IDs to extract (default: all chains)")
	extractSeqCmd.Flags().StringVar(&seqChains, "chain", "", "Alias for --chains")
	extractSeqCmd.Flags().StringVarP(&seqOutput, "output", "o", "", "Output file (default: stdout)")
	extractSeqCmd.Flags().BoolVar(&useSeqRes, "seqres", false, "Use SEQRES records instead of ATOM records")
	addInFormatFlag(extractSeqCmd)
}

func runExtractSeq(cmd *cobra.Command, args []string) error {
	inputFile, err := resolveInputPath(args)
	if err != nil {
		return err
	}

	s, err := readStructure(inputFile)
	if err != nil {
		return err
	}

	chainList := splitChainList(seqChains)
	order, sequences := extractSequences(s, chainList, useSeqRes)

	if seqOutput == "" || seqOutput == "-" {
		return writeFASTAToWriter(order, sequences, os.Stdout, inputFile)
	}
	file, err := os.Create(seqOutput)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()
	return writeFASTAToWriter(order, sequences, file, inputFile)
}

// extractSequences returns the chain IDs in file order alongside their
// sequences, so FASTA output is deterministic.
func extractSequences(s *structure.Structure, chainList []string, fromSeqRes bool) ([]string, map[string]string) {
	wanted := make(map[string]bool, len(chainList))
	for _, id := range chainList {
		wanted[id] = true
	}
	want := func(id string) bool { return len(wanted) == 0 || wanted[id] }

	sequences := make(map[string]string)
	var order []string

	if fromSeqRes {
		for _, cs := range s.SeqRes {
			if !want(cs.ChainID) {
				continue
			}
			var b strings.Builder
			for _, mon := range cs.Residues {
				b.WriteByte(structure.OneLetter(mon, s.ModRes))
			}
			if b.Len() > 0 {
				order = append(order, cs.ChainID)
				sequences[cs.ChainID] = b.String()
			}
		}
		for _, id := range s.ChainIDs() {
			if want(id) && sequences[id] == "" {
				fmt.Fprintf(os.Stderr, "Warning: --seqres specified but no SEQRES records found for chain %s\n", id)
			}
		}
		return order, sequences
	}

	// Build from coordinates, using the first model only and inserting gap
	// characters where residue numbering skips ahead.
	byChain := make(map[string][]*structure.Residue)
	firstModel := 0
	for _, a := range s.Atoms {
		if firstModel == 0 {
			firstModel = a.Model
		}
	}
	for _, r := range structure.Residues(s.Atoms) {
		if r.Model != firstModel || !want(r.ChainID) || !isPolymerResidue(r, s.ModRes) {
			continue
		}
		byChain[r.ChainID] = append(byChain[r.ChainID], r)
	}

	for _, id := range s.ChainIDs() {
		residues, ok := byChain[id]
		if !ok || len(residues) == 0 {
			continue
		}
		var b strings.Builder
		prev := residues[0].ResSeq - 1
		for _, r := range residues {
			for gap := r.ResSeq - prev - 1; gap > 0; gap-- {
				b.WriteByte('-')
			}
			b.WriteByte(structure.OneLetter(r.ResName, s.ModRes))
			prev = r.ResSeq
		}
		if b.Len() > 0 {
			order = append(order, id)
			sequences[id] = b.String()
		}
	}
	return order, sequences
}

// isPolymerResidue decides whether a residue contributes to a chain's
// sequence. ATOM records always do; hetero residues only when they name a
// known polymer component (MSE, SEP, ...), which keeps ligands, ions and
// waters out of the FASTA output.
func isPolymerResidue(r *structure.Residue, modres map[string]string) bool {
	for _, a := range r.Atoms {
		if !a.Hetatm {
			return true
		}
		if a.IsWater() {
			return false
		}
	}
	if _, ok := modres[strings.ToUpper(strings.TrimSpace(r.ResName))]; ok {
		return true
	}
	return structure.IsPolymerComponent(r.ResName)
}

func writeFASTAToWriter(order []string, sequences map[string]string, writer io.Writer, inputFile string) error {
	baseName := "stdin"
	if inputFile != "" {
		baseName = strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	}

	for _, chainID := range order {
		sequence := sequences[chainID]
		fmt.Fprintf(writer, ">%s_%s\n", baseName, chainID)
		for i := 0; i < len(sequence); i += 80 {
			end := i + 80
			if end > len(sequence) {
				end = len(sequence)
			}
			fmt.Fprintf(writer, "%s\n", sequence[i:end])
		}
	}

	if len(order) > 0 {
		fmt.Fprintf(writer, "\n")
	}
	return nil
}
