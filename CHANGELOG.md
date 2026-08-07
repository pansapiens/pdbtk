# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-08-07

### Added
- PDBx/mmCIF support: every command reads and writes both PDB and mmCIF, so `pdbtk` also works as a format converter
- Input format is detected from the file extension or contents (including gzipped input); `--in-format` overrides it
- Output format follows the `--output` extension, else the input format; `--out-format` overrides it
- `--force-lossy-pdb` to permit writing PDB output that cannot faithfully represent the structure
- `--no-hetatm` flag for `extract` to drop hetero atoms
- `cif` and `cif.gz` formats restored for the `get` command
- `version` command to print the current version number
- Version information displayed in help text
- `--chain` as alias for `--chains` flag in `extract` and `extract-seq` commands
- PDB output now includes `SEQRES` records, and `LINK` records whenever both bond sites survive filtering

### Changed
- **Breaking:** `extract` now keeps hetero atoms belonging to the selected chains by default and drops waters. Previously hetero records before a chain's `TER` were kept and those after it were dropped unless `--keep-hetatm` was given. `--keep-hetatm` is still accepted but has no effect.
- **Breaking:** writing mmCIF-only features (chain IDs longer than one character, more than 99,999 atoms, residue names longer than three characters) to a PDB file is now an error instead of silent corruption; pass `--force-lossy-pdb` to convert anyway
- Chain IDs are no longer restricted to a single character, since mmCIF permits longer identifiers
- `renumber-residues` now errors when `--chain` names a chain that does not exist
- `HEADER` records are written with the entry ID in columns 63-66 as the format specifies, rather than in the classification field
- Replaced the internal structure model and the `github.com/TuftsBCB/io` dependency with a format-neutral model and native PDB/mmCIF codecs

### Fixed
- Residue names are no longer corrupted on output. Nucleic acid residues were rewritten to amino acids (`DT` became `THR`) and ligands became `UNK`, because the internal model stored only single-letter residue codes.
- Occupancy and B-factor values are preserved instead of being replaced with `1.00` and `20.00`
- Element symbols are preserved instead of being re-derived from the atom name (`ZN` was written as `Z`)
- Waters and hetero records following a `TER` are no longer silently dropped while parsing

### Removed
- `github.com/TuftsBCB/io`, `github.com/TuftsBCB/seq` and `github.com/TuftsBCB/structure` dependencies

## [0.1.1] - 2025-01-27

### Added
- `get` command for downloading PDB files from the RCSB PDB database
- `rename-chain` command for renaming chains in PDB files
- `renumber-residues` command for renumbering residues with gap preservation or sequential numbering
- Output to stdout option with `--output -` flag
- Stdin support for `extract` and `extract-seq` commands when no input file is specified
- Run tests on CI
- `--altloc` flag for extract command to filter by ALTLOC identifier or take first ALTLOC when duplicates exist (can be used with or without --chains)
- `--seqres` flag for extract-seq command to strictly use SEQRES records only
- Tests for extract-seq functionality including SEQRES, ATOM-based extraction, and gap handling

### Removed
- Removed CIF/PDBx support - who really needs more than 99,999 atoms, really?

### Changed
- `extract-seq` now defaults to extracting sequences from ATOM records instead of SEQRES records (use `--seqres` flag to use SEQRES records)
- Refactored PDB reader and writer functions

### Fixed
- Improved error handling for CIF files containing nucleic acid sequences - now provides informative error message instead of panic
- REMARK 1 COMMAND line in extracted PDB files now includes all command-line arguments instead of just the command name
- Added comprehensive test coverage for extract command to prevent regression
- Fixed extract-seq failing to extract sequences from piped input without SEQRES records - now extracts sequences from ATOM records with gap characters for missing residues

## [0.1] - 2025-09-17

### Added
- `extract` and `extract-seq` commands for extracting specific chains or a FASTA sequence from PDB files
