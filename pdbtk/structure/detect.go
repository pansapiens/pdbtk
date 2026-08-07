package structure

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FormatFromPath infers a format from a filename, transparently looking past a
// .gz suffix. Returns FormatUnknown when the extension is not recognised.
func FormatFromPath(path string) Format {
	name := strings.ToLower(filepath.Base(path))
	name = strings.TrimSuffix(name, ".gz")
	switch filepath.Ext(name) {
	case ".pdb", ".ent":
		return FormatPDB
	case ".cif", ".mmcif":
		return FormatCIF
	}
	return FormatUnknown
}

// FormatFromContent sniffs a format from the leading bytes of a file. mmCIF is
// identified by its mandatory data_ block header; PDB by any of the coordinate
// or header record names appearing at the start of a line.
func FormatFromContent(content []byte) Format {
	limit := len(content)
	if limit > 8192 {
		limit = 8192
	}
	for _, line := range strings.Split(string(content[:limit]), "\n") {
		line = strings.TrimLeft(line, " \t")
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(strings.ToLower(line), "data_"),
			strings.HasPrefix(strings.ToLower(line), "loop_"),
			strings.HasPrefix(line, "_"):
			return FormatCIF
		case strings.HasPrefix(line, "ATOM  "), strings.HasPrefix(line, "HETATM"),
			strings.HasPrefix(line, "HEADER"), strings.HasPrefix(line, "MODEL "),
			strings.HasPrefix(line, "REMARK"), strings.HasPrefix(line, "SEQRES"),
			strings.HasPrefix(line, "CRYST1"), strings.HasPrefix(line, "TITLE "):
			return FormatPDB
		}
	}
	return FormatUnknown
}

// ReadFile reads a structure from disk, decompressing .gz transparently.
// A non-zero want overrides format detection.
func ReadFile(path string, want Format) (*Structure, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content, err := maybeGunzip(raw)
	if err != nil {
		return nil, err
	}
	f := want
	if f == FormatUnknown {
		f = FormatFromPath(path)
	}
	if f == FormatUnknown {
		f = FormatFromContent(content)
	}
	if f == FormatUnknown {
		return nil, fmt.Errorf("could not determine the format of %s (use --in-format)", path)
	}
	return Decode(content, f)
}

// ReadContent reads a structure from an in-memory buffer, decompressing gzip
// transparently. A non-zero want overrides content sniffing.
func ReadContent(raw []byte, want Format) (*Structure, error) {
	content, err := maybeGunzip(raw)
	if err != nil {
		return nil, err
	}
	f := want
	if f == FormatUnknown {
		f = FormatFromContent(content)
	}
	if f == FormatUnknown {
		return nil, fmt.Errorf("could not determine input format (use --in-format)")
	}
	return Decode(content, f)
}

// Decode parses content known to be in the given format.
func Decode(content []byte, f Format) (*Structure, error) {
	switch f {
	case FormatPDB:
		return ReadPDB(content)
	case FormatCIF:
		return ReadCIF(content)
	}
	return nil, fmt.Errorf("unsupported format %q", f)
}

// Write serialises a structure in the given format.
func Write(w io.Writer, s *Structure, f Format, opt WriteOptions) error {
	switch f {
	case FormatPDB:
		return WritePDB(w, s, opt)
	case FormatCIF:
		return WriteCIF(w, s, opt)
	}
	return fmt.Errorf("unsupported output format %q", f)
}

func maybeGunzip(raw []byte) ([]byte, error) {
	if len(raw) < 2 || raw[0] != 0x1f || raw[1] != 0x8b {
		return raw, nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to decompress gzip input: %v", err)
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress gzip input: %v", err)
	}
	return out, nil
}
