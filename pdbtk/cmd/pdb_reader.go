package cmd

import (
	"bufio"
	"os"
	"strings"

	"github.com/TuftsBCB/io/pdb"
)

// PDBEntryWithAltLoc extends the PDB entry with ALTLOC information
type PDBEntryWithAltLoc struct {
	*pdb.Entry
	AltLocList []byte // ALTLOC values in the order they appear in the file
}

// ReadPDBWithAltLoc reads a PDB file and preserves ALTLOC information
func ReadPDBWithAltLoc(filename string) (*PDBEntryWithAltLoc, error) {
	// First, read the PDB file normally
	entry, err := pdb.ReadPDB(filename)
	if err != nil {
		return nil, err
	}

	// Create the extended entry
	extendedEntry := &PDBEntryWithAltLoc{
		Entry:      entry,
		AltLocList: make([]byte, 0),
	}

	// Now read the file again to extract ALTLOC information
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ATOM") || strings.HasPrefix(line, "HETATM") {
			if len(line) >= 17 {
				// Extract ALTLOC (column 17)
				altLoc := line[16]
				extendedEntry.AltLocList = append(extendedEntry.AltLocList, altLoc)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return extendedEntry, nil
}

// ReadPDBWithAltLocFromContent reads PDB content and preserves ALTLOC information
func ReadPDBWithAltLocFromContent(content []byte, filename string) (*PDBEntryWithAltLoc, error) {
	// First, read the PDB content normally
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

	// Create the extended entry
	extendedEntry := &PDBEntryWithAltLoc{
		Entry:      entry,
		AltLocList: make([]byte, 0),
	}

	// Now parse the content to extract ALTLOC information
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "ATOM") || strings.HasPrefix(line, "HETATM") {
			if len(line) >= 17 {
				// Extract ALTLOC (column 17)
				altLoc := line[16]
				extendedEntry.AltLocList = append(extendedEntry.AltLocList, altLoc)
			}
		}
	}

	return extendedEntry, nil
}
