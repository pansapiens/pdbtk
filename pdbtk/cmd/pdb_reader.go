package cmd

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/TuftsBCB/io/pdb"
)

// HetRecord is a preserved HETATM line from the input file plus parsed fields used for filtering.
type HetRecord struct {
	Raw      string
	Chain    byte
	ResName  string
	AltLoc   byte
	Serial   int
	IsWater  bool
	AtomName string
	SeqNum   int
	InsCode  byte
}

// PDBEntryWithAltLoc extends the PDB entry with ALTLOC information
type PDBEntryWithAltLoc struct {
	*pdb.Entry
	AltLocList  []byte // ALTLOC values aligned with atoms TuftsBCB accepts from the coordinate section
	HetRecords  []HetRecord
	LinkRecords []string // verbatim LINK records from the file (coordinate-related connectivity)
}

type parserSeenPosition struct {
	chain byte
	model int
}

// ParseHetRecordLine parses a PDB HETATM line into HetRecord. Returns ok false if the line is malformed.
func ParseHetRecordLine(line string) (HetRecord, bool) {
	if len(line) < 26 {
		return HetRecord{}, false
	}
	if !strings.HasPrefix(line, "HETATM") {
		return HetRecord{}, false
	}
	serialStr := strings.TrimSpace(line[6:11])
	sn, err := strconv.Atoi(serialStr)
	if err != nil || sn < 0 {
		return HetRecord{}, false
	}
	seqStr := strings.TrimSpace(line[22:26])
	seqNum, err := strconv.Atoi(seqStr)
	if err != nil {
		return HetRecord{}, false
	}
	res := strings.TrimSpace(line[17:20])
	rec := HetRecord{
		Raw:      strings.TrimRight(line, "\r\n"),
		Serial:   sn,
		ResName:  res,
		IsWater:  res == "HOH",
		AltLoc:   line[16],
		AtomName: strings.TrimSpace(line[12:16]),
		SeqNum:   seqNum,
	}
	if len(line) > 21 {
		rec.Chain = line[21]
	}
	if len(line) > 26 {
		rec.InsCode = line[26]
	}
	return rec, true
}

// LinkAtomSite is one end of a PDB LINK record (fixed-column layout).
type LinkAtomSite struct {
	AtomName string
	ResName  string
	Chain    byte
	SeqNum   int
	InsCode  byte
}

func padCols80(line string) string {
	line = strings.TrimRight(line, "\r\n")
	if len(line) >= 80 {
		return line[:80]
	}
	return line + strings.Repeat(" ", 80-len(line))
}

// ParseLINKRecord parses wwPDB v3 LINK bonding fields (coordinate-section layout).
func ParseLINKRecord(line string) (LinkAtomSite, LinkAtomSite, bool) {
	s := padCols80(line)
	if len(s) < 57 {
		return LinkAtomSite{}, LinkAtomSite{}, false
	}
	rec := strings.TrimSpace(s[:6])
	if len(rec) < 4 || rec[:4] != "LINK" {
		return LinkAtomSite{}, LinkAtomSite{}, false
	}

	a1, ok1 := parseLinkAtom1Side(s)
	a2, ok2 := parseLinkAtom2Side(s)
	if !(ok1 && ok2) {
		return LinkAtomSite{}, LinkAtomSite{}, false
	}
	return a1, a2, true
}

func normalizeLinkInsertion(code byte) byte {
	if code == 0 || code == ' ' {
		return ' '
	}
	return code
}

func parseLinkAtom1Side(s80 string) (LinkAtomSite, bool) {
	atom := strings.TrimSpace(s80[12:16])
	res := strings.TrimSpace(s80[17:20])
	chain := s80[21]
	seqStr := strings.TrimSpace(s80[22:26])
	seq, err := strconv.Atoi(seqStr)
	if err != nil || len(atom) == 0 || len(res) == 0 || chain == ' ' || chain == 0 {
		return LinkAtomSite{}, false
	}
	ins := normalizeLinkInsertion(s80[26])
	return LinkAtomSite{AtomName: atom, ResName: res, Chain: chain, SeqNum: seq, InsCode: ins}, true
}

func parseLinkAtom2Side(s80 string) (LinkAtomSite, bool) {
	atom := strings.TrimSpace(s80[42:46])
	res := strings.TrimSpace(s80[47:50])
	chain := s80[51]
	seqStr := strings.TrimSpace(s80[52:56])
	seq, err := strconv.Atoi(seqStr)
	if err != nil || len(atom) == 0 || len(res) == 0 || chain == ' ' || chain == 0 {
		return LinkAtomSite{}, false
	}
	ins := normalizeLinkInsertion(s80[56])
	return LinkAtomSite{AtomName: atom, ResName: res, Chain: chain, SeqNum: seq, InsCode: ins}, true
}

func tuftsWouldIncludeAtomOrHetatm(processed map[parserSeenPosition]bool, curModel int, line string) bool {
	if !(strings.HasPrefix(line, "ATOM") || strings.HasPrefix(line, "HETATM")) {
		return false
	}
	if len(line) < 26 {
		return false
	}
	ident := line[21]
	if processed[parserSeenPosition{chain: ident, model: curModel}] {
		return false
	}
	res := strings.TrimSpace(line[17:20])
	if res == "HOH" {
		return false
	}
	return true
}

func parseModelSerial(line string) (int, bool) {
	fs := strings.Fields(strings.TrimSpace(line))
	if len(fs) < 2 || strings.ToUpper(fs[0]) != "MODEL" {
		return 0, false
	}
	n, err := strconv.Atoi(fs[1])
	return n, err == nil && n >= 1
}

// scanPDBRawLines parses ATOM/HETATM lines while mirroring MODEL/TER bookkeeping from Tufts parser.
func scanPDBRawLines(extendedEntry *PDBEntryWithAltLoc, visitLines func(cb func(line string))) {
	processed := make(map[parserSeenPosition]bool)
	lastSeen := parserSeenPosition{}
	curModel := 1

	visitor := func(line string) {
		fs := strings.Fields(line)
		if len(fs) == 0 {
			return
		}
		kw := strings.ToUpper(fs[0])

		switch kw {
		case "MODEL":
			if n, ok := parseModelSerial(line); ok {
				curModel = n
				processed = make(map[parserSeenPosition]bool)
				lastSeen = parserSeenPosition{}
			}
			return
		case "TER":
			if lastSeen.chain != 0 {
				processed[parserSeenPosition{chain: lastSeen.chain, model: curModel}] = true
			}
			return
		case "LINK":
			extendedEntry.LinkRecords = append(extendedEntry.LinkRecords, strings.TrimRight(line, "\r"))
			return
		default:
		}

		if !(strings.HasPrefix(line, "ATOM") || strings.HasPrefix(line, "HETATM")) {
			return
		}

		include := tuftsWouldIncludeAtomOrHetatm(processed, curModel, line)

		if include && len(line) >= 17 {
			extendedEntry.AltLocList = append(extendedEntry.AltLocList, line[16])
		}

		if strings.HasPrefix(line, "HETATM") && !include {
			if hr, ok := ParseHetRecordLine(line); ok {
				extendedEntry.HetRecords = append(extendedEntry.HetRecords, hr)
			}
		}

		if include && len(line) >= 22 {
			lastSeen = parserSeenPosition{chain: line[21], model: curModel}
		}
	}

	visitLines(visitor)
}

// ReadPDBWithAltLoc reads a PDB file and preserves ALTLOC information
func ReadPDBWithAltLoc(filename string) (*PDBEntryWithAltLoc, error) {
	entry, err := pdb.ReadPDB(filename)
	if err != nil {
		return nil, err
	}

	extendedEntry := &PDBEntryWithAltLoc{
		Entry:       entry,
		AltLocList:  make([]byte, 0),
		HetRecords:  make([]HetRecord, 0),
		LinkRecords: make([]string, 0),
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanPDBRawLines(extendedEntry, func(cb func(line string)) {
		for scanner.Scan() {
			cb(scanner.Text())
		}
	})
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return extendedEntry, nil
}

// ReadPDBWithAltLocFromContent reads PDB content and preserves ALTLOC information
func ReadPDBWithAltLocFromContent(content []byte, filename string) (*PDBEntryWithAltLoc, error) {
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

	entry, err := pdb.ReadPDB(tmpfile.Name())
	if err != nil {
		return nil, err
	}

	extendedEntry := &PDBEntryWithAltLoc{
		Entry:       entry,
		AltLocList:  make([]byte, 0),
		HetRecords:  make([]HetRecord, 0),
		LinkRecords: make([]string, 0),
	}

	lines := strings.Split(string(content), "\n")
	scanPDBRawLines(extendedEntry, func(cb func(line string)) {
		for _, line := range lines {
			cb(line)
		}
	})

	return extendedEntry, nil
}
