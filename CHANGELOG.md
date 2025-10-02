# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
