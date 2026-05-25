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
	chains     string
	output     string
	altloc     string
	keepHetatm bool
	keepWaters bool
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
  pdbtk extract --chains A --altloc first 1a02.pdb

  # Retain hetero atoms (excluding waters) matching extracted chains
  pdbtk extract --chains A --keep-hetatm 1a02.pdb

  # Include waters alongside hetero atoms
  pdbtk extract --chains A --keep-hetatm --keep-waters 1a02.pdb`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExtract,
}

func init() {
	extractCmd.Flags().StringVarP(&chains, "chains", "c", "", "Comma-separated list of chain IDs to extract")
	extractCmd.Flags().StringVar(&chains, "chain", "", "Alias for --chains")
	extractCmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: stdout)")
	extractCmd.Flags().StringVar(&altloc, "altloc", "", "Filter by ALTLOC identifier (e.g., A, B) or 'first' to take first ALTLOC when duplicates exist")
	extractCmd.Flags().BoolVar(&keepHetatm, "keep-hetatm", false, "Retain HETATM records (excluding waters) matching the extraction selection; also emits LINK records when both bond sites remain in the output")
	extractCmd.Flags().BoolVar(&keepWaters, "keep-waters", false, "Retain HOH waters matching the extraction selection")
}

func runExtract(cmd *cobra.Command, args []string) error {
	var inputFile string
	var isStdin bool

	if len(args) > 0 {
		inputFile = args[0]
		isStdin = false
		if err := CheckFileExists(inputFile); err != nil {
			return err
		}
		inputExt := strings.ToLower(filepath.Ext(inputFile))
		if inputExt != ".pdb" {
			return fmt.Errorf("only PDB files are supported, got: %s", inputExt)
		}
	} else {
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

	if chains == "" && altloc == "" && !keepHetatm && !keepWaters {
		return fmt.Errorf("at least one of --chains, --altloc, --keep-hetatm, or --keep-waters must be specified")
	}

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

	var entry *pdb.Entry
	var altLocList []byte
	var hetRaw []HetRecord
	var linkRaw []string
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
		hetRaw = extendedEntry.HetRecords
		linkRaw = extendedEntry.LinkRecords
	} else {
		extendedEntry, err := ReadPDBWithAltLoc(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read PDB file: %v", err)
		}
		entry = extendedEntry.Entry
		altLocList = extendedEntry.AltLocList
		hetRaw = extendedEntry.HetRecords
		linkRaw = extendedEntry.LinkRecords
	}

	var extractedChains *pdb.Entry
	if len(chainList) > 0 {
		extractedChains, altLocList, err = ExtractChainsPDB(entry, chainList, altLocList)
		if err != nil {
			return fmt.Errorf("failed to extract chains: %v", err)
		}
	} else {
		extractedChains = entry
	}

	if altloc != "" {
		extractedChains, altLocList, err = filterByAltLoc(extractedChains, altLocList, altloc)
		if err != nil {
			return fmt.Errorf("failed to filter by ALTLOC: %v", err)
		}
	}

	filteredHet := filterHetRecords(hetRaw, chainList, keepHetatm, keepWaters, altloc)

	var filteredLinks []string
	if keepHetatm {
		filteredLinks = filterLINKRecords(linkRaw, extractedChains, filteredHet)
	}

	commandLine := buildCommandLine(cmd, args, inputFile)

	if output == "" || output == "-" {
		return writePDBToWriterWithAltLoc(extractedChains, altLocList, filteredHet, filteredLinks, os.Stdout, commandLine)
	}
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()
	return writePDBToWriterWithAltLoc(extractedChains, altLocList, filteredHet, filteredLinks, file, commandLine)
}

func linkAtomSiteKey(s LinkAtomSite) string {
	return fmt.Sprintf("%s|%s|%c|%d|%c",
		strings.ToUpper(strings.TrimSpace(s.AtomName)),
		strings.ToUpper(strings.TrimSpace(s.ResName)),
		s.Chain,
		s.SeqNum,
		normalizeLinkInsertion(s.InsCode),
	)
}

func buildLinkAtomSiteKeys(entry *pdb.Entry, keptHet []HetRecord) map[string]struct{} {
	keys := make(map[string]struct{})
	for _, ch := range entry.Chains {
		for _, md := range ch.Models {
			for _, res := range md.Residues {
				res3 := strings.ToUpper(singleLetterToResidue(strings.ToUpper(string(byte(res.Name)))))
				ins := normalizeLinkInsertion(res.InsertionCode)
				for _, atom := range res.Atoms {
					an := strings.ToUpper(strings.TrimSpace(RemoveAltLocFromAtomName(atom.Name)))
					keys[linkAtomSiteKey(LinkAtomSite{
						AtomName: an,
						ResName:  res3,
						Chain:    ch.Ident,
						SeqNum:   res.SequenceNum,
						InsCode:  ins,
					})] = struct{}{}
				}
			}
		}
	}
	for _, h := range keptHet {
		ins := normalizeLinkInsertion(h.InsCode)
		keys[linkAtomSiteKey(LinkAtomSite{
			AtomName: strings.ToUpper(strings.TrimSpace(h.AtomName)),
			ResName:  strings.ToUpper(strings.TrimSpace(h.ResName)),
			Chain:    h.Chain,
			SeqNum:   h.SeqNum,
			InsCode:  ins,
		})] = struct{}{}
	}
	return keys
}

func filterLINKRecords(lines []string, entry *pdb.Entry, keptHet []HetRecord) []string {
	if len(lines) == 0 {
		return nil
	}
	keySet := buildLinkAtomSiteKeys(entry, keptHet)
	var out []string
	for _, line := range lines {
		a1, a2, ok := ParseLINKRecord(line)
		if !ok {
			continue
		}
		if _, ok1 := keySet[linkAtomSiteKey(a1)]; !ok1 {
			continue
		}
		if _, ok2 := keySet[linkAtomSiteKey(a2)]; !ok2 {
			continue
		}
		out = append(out, line)
	}
	return out
}

func filterHetRecords(hetRaw []HetRecord, chainList []string, keepHetatm bool, keepWaters bool, altlocFilter string) []HetRecord {
	if !keepHetatm && !keepWaters {
		return nil
	}
	validChains := make(map[byte]bool)
	for _, id := range chainList {
		if len(id) == 1 {
			validChains[id[0]] = true
		}
	}
	chainFilter := len(chainList) > 0

	var sel []HetRecord
	for _, h := range hetRaw {
		if chainFilter && !validChains[h.Chain] {
			continue
		}
		if h.IsWater {
			if !keepWaters {
				continue
			}
		} else if !keepHetatm {
			continue
		}
		sel = append(sel, h)
	}
	if altlocFilter == "" {
		return sel
	}
	return filterHetByAltLoc(sel, altlocFilter)
}

func hetAltLocGroupKey(h HetRecord) string {
	ic := h.InsCode
	if ic == 0 || ic == ' ' {
		return fmt.Sprintf("%c|%d|%c|%s", h.Chain, h.SeqNum, ' ', strings.TrimSpace(h.AtomName))
	}
	return fmt.Sprintf("%c|%d|%c|%s", h.Chain, h.SeqNum, ic, strings.TrimSpace(h.AtomName))
}

func filterHetByAltLoc(hets []HetRecord, altlocFilter string) []HetRecord {
	groups := make(map[string][]HetRecord)
	var order []string
	seen := make(map[string]bool)
	for _, h := range hets {
		k := hetAltLocGroupKey(h)
		groups[k] = append(groups[k], h)
		if !seen[k] {
			seen[k] = true
			order = append(order, k)
		}
	}
	var result []HetRecord
	for _, k := range order {
		result = append(result, pickHetAltLoc(groups[k], altlocFilter)...)
	}
	return result
}

func pickHetAltLoc(group []HetRecord, altlocFilter string) []HetRecord {
	if len(group) == 0 {
		return nil
	}
	if altlocFilter == "first" {
		if len(group) == 1 {
			return []HetRecord{group[0]}
		}
		selectedIdx := 0
		for i, rec := range group {
			al := rec.AltLoc
			if al != ' ' && al != 0 {
				selectedIdx = i
				break
			}
		}
		return []HetRecord{group[selectedIdx]}
	}
	target := altlocFilter[0]
	var out []HetRecord
	for _, rec := range group {
		al := rec.AltLoc
		if al == target || al == ' ' || al == 0 {
			out = append(out, rec)
		}
	}
	return out
}

func readPDB(filename string) (*pdb.Entry, error) {
	return pdb.ReadPDB(filename)
}

func readPDBFromContent(content []byte) (*pdb.Entry, error) {
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
	newEntry := &pdb.Entry{
		Path:   entry.Path,
		IdCode: entry.IdCode,
		Chains: make([]*pdb.Chain, 0),
		Scop:   entry.Scop,
		Cath:   entry.Cath,
	}

	validChains := make(map[byte]bool)
	for _, chainID := range chainList {
		if len(chainID) == 1 {
			validChains[chainID[0]] = true
		}
	}

	newAltLocList := make([]byte, 0)
	atomIndex := 0

	for _, chain := range entry.Chains {
		atomCount := 0
		for _, model := range chain.Models {
			for _, residue := range model.Residues {
				atomCount += len(residue.Atoms)
			}
		}

		if validChains[chain.Ident] {
			newEntry.Chains = append(newEntry.Chains, chain)
			if altLocList != nil && atomIndex+atomCount <= len(altLocList) {
				newAltLocList = append(newAltLocList, altLocList[atomIndex:atomIndex+atomCount]...)
			}
		}
		atomIndex += atomCount
	}

	return newEntry, newAltLocList, nil
}

func filterByAltLoc(entry *pdb.Entry, altLocList []byte, altlocFilter string) (*pdb.Entry, []byte, error) {
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

				type atomWithIndex struct {
					atom   pdb.Atom
					index  int
					altLoc byte
				}
				atomGroups := make(map[string][]atomWithIndex)

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

				for _, group := range atomGroups {
					if altlocFilter == "first" {
						if len(group) > 1 {
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
							newResidue.Atoms = append(newResidue.Atoms, group[0].atom)
							newAltLocList = append(newAltLocList, group[0].altLoc)
						}
					} else {
						targetAltLoc := altlocFilter[0]
						for _, atomInfo := range group {
							if atomInfo.altLoc == targetAltLoc || atomInfo.altLoc == ' ' {
								newResidue.Atoms = append(newResidue.Atoms, atomInfo.atom)
								newAltLocList = append(newAltLocList, atomInfo.altLoc)
							}
						}
					}
				}

				if len(newResidue.Atoms) > 0 {
					newModel.Residues = append(newModel.Residues, newResidue)
				}
			}

			if len(newModel.Residues) > 0 {
				newChain.Models = append(newChain.Models, newModel)
			}
		}

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

	parts = append(parts, "pdbtk", "extract")

	if chains != "" {
		parts = append(parts, "--chain", chains)
	}
	if output != "" {
		parts = append(parts, "--output", output)
	}
	if altloc != "" {
		parts = append(parts, "--altloc", altloc)
	}
	if keepHetatm {
		parts = append(parts, "--keep-hetatm")
	}
	if keepWaters {
		parts = append(parts, "--keep-waters")
	}

	if inputFile != "" {
		parts = append(parts, inputFile)
	}

	return strings.Join(parts, " ")
}
