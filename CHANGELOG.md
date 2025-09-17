# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `get` command for downloading PDB files from the RCSB PDB database
- Output to stdout option with `--output -` flag
- Stdin support for `extract` and `extract-seq` commands when no input file is specified

### Removed
- Removed CIF/PDBx support - who really needs more than 99,999 atoms, really?

### Fixed
- Improved error handling for CIF files containing nucleic acid sequences - now provides informative error message instead of panic
- REMARK 1 COMMAND line in extracted PDB files now includes all command-line arguments instead of just the command name

## [0.1] - 2025-09-17

### Added
- `extract` and `extract-seq` commands for extracting specific chains or a FASTA sequence from PDB files
