# Usage and Examples

## Quick Guide

- **Download PDB files**: [get](#get-usage)
- **Coordinate extraction**: [extract](#extract-usage)
- **Sequence extraction**: [extract-seq](#extract-seq-usage)
- **Chain manipulation**: [rename-chain](#rename-chain-usage), [renumber-residues](#renumber-residues-usage)
- **Version info**: [version](#version-usage)
- **Other**: [completion](#completion-usage)

## File formats

`pdbtk` reads and writes both the legacy PDB format and PDBx/mmCIF. The same
operations apply to either, so `pdbtk` doubles as a format converter.

**Input format** is detected from the file extension (`.pdb`, `.ent`, `.cif`,
`.mmcif`, optionally `.gz`), falling back to sniffing the file contents — which
is also how input on stdin is handled. Override it with `--in-format pdb|cif`.

**Output format** follows the `--output` extension when there is one, otherwise
it matches the input format. Override it with `--out-format pdb|cif`.

```bash
# Convert mmCIF to PDB
$ pdbtk extract --out-format pdb 1a02.cif > 1a02.pdb

# Convert PDB to mmCIF (the .cif output extension is enough)
$ pdbtk extract --output 1a02.cif 1a02.pdb

# Reading gzipped input directly
$ pdbtk extract-seq 1a02.cif.gz
```

### Converting mmCIF to PDB

mmCIF can represent structures the legacy PDB format cannot: chain IDs longer
than one character, residue names longer than three characters, and coordinates
outside the range `-999.999` to `9999.999`. When such a structure would be
written as PDB, `pdbtk` **fails with an error** rather than silently corrupting
it:

```text
Error: structure cannot be represented in PDB format:
  - chain ID "AAA" is longer than one character
write mmCIF instead (--out-format cif), or pass --force-lossy-pdb to convert anyway
```

Pass `--force-lossy-pdb` to convert anyway. Chain IDs, residue names and atom
names are truncated, values that overflow a fixed-point column become asterisks
so the remaining columns stay aligned, a warning is printed to stderr, and a
`REMARK   1 LOSSY CONVERSION` line records what was lost.

Atom serials above 99,999 and residue numbers above 9,999 are **not** treated as
loss: they are written using the wwPDB
[hybrid-36](http://cci.lbl.gov/hybrid_36/) convention, which `pdbtk` reads back
exactly. Be aware that not every other tool understands hybrid-36 — write mmCIF
if the file is destined for one that does not.

### What is preserved

Both writers emit coordinates, connectivity (`LINK` / `struct_conn`), sequence
records (`SEQRES` / `pdbx_poly_seq_scheme`) and a provenance record. Other
metadata in the input file (experimental details, citations, secondary
structure) is **not** carried through.

## pdbtk Usage

```text
pdbtk -- a cross-platform, efficient and practical PDB/PDBx-mmCIF structure file manipulation toolkit

Version: 0.2.0
Author: Perry
Source code: https://github.com/perry/pdbtk

pdbtk is a command-line toolkit for manipulating PDB and PDBx/mmCIF structure files.
It provides various operations for extracting, filtering, and transforming protein structure data.

Usage:
  pdbtk [command]

Available Commands:
  get               Download a structure file from the RCSB PDB database
  extract           Extract chains from a PDB or mmCIF file
  extract-seq       Extract sequences from chains in a PDB or mmCIF file
  rename-chain      Rename a chain in a PDB or mmCIF file
  renumber-residues Renumber residues in a PDB or mmCIF file
  version           Print the version number
  completion        Generate the autocompletion script for the specified shell
  help              Help about any command

Flags:
  -h, --help              help for pdbtk
      --force-lossy-pdb   Allow writing PDB output that cannot faithfully represent the structure (long chain IDs, long residue names, out-of-range coordinates)

Use "pdbtk [command] --help" for more information about a command.
```

## get Usage

```text
Download a structure file from the RCSB PDB database using the PDB code.
The file will be downloaded from https://files.rcsb.org/download/{pdb_code}.{format}

By default, the file is saved as {pdb_code}.{format} in the current directory.
Use --output to specify a different filename or "-" to output to stdout.
Use --format to specify the file format (pdb, pdb.gz, cif, cif.gz).

Usage:
  pdbtk get [flags] <pdb_code>

Flags:
  -f, --format string   File format: pdb, pdb.gz, cif, cif.gz (default "pdb")
  -h, --help            help for get
  -o, --output string   Output file (default: {pdb_code}.{format}, use '-' for stdout)
```

### Examples

Download 1A02 as PDB file
```bash
$ pdbtk get 1A02
```

Download as compressed PDB file
```bash
$ pdbtk get --format pdb.gz 1A02
```

Download as an mmCIF file
```bash
$ pdbtk get --format cif 1A02
```

Download to stdout and view the first 10 line with `head`
```bash
$ pdbtk get --output - 1A02 | head
```

Download to specific filename
```bash
$ pdbtk get --output my_structure.pdb 1A02
```

Download the gzipped PDB, uncompress it and extract chain B in a single command
```bash
$ pdbtk get --format pdb.gz -o - 1A02 | gunzip -c - | pdbtk extract --chains B
```

## extract Usage

```text
Extract specific chains from a PDB or PDBx/mmCIF structure file.
The output can be written to a file or stdout (if no output file is specified).
If no input file is specified, reads from stdin.

By default hetero atoms (ligands, ions, modified residues) belonging to the
selected chains are kept and waters are dropped. Use --keep-waters to retain
waters, or --no-hetatm to drop hetero atoms as well.

LINK / struct_conn records are written whenever both bond sites survive filtering.

Usage:
  pdbtk extract [flags] [input_file]

Flags:
      --altloc string       Filter by ALTLOC identifier (e.g., A, B) or 'first' to take first ALTLOC when duplicates exist
      --chain string        Alias for --chains
  -c, --chains string       Comma-separated list of chain IDs to extract (default: all chains)
  -h, --help                help for extract
      --in-format string    Input format: pdb or cif (default: inferred from the file extension or contents)
      --keep-hetatm         Retain hetero atoms (the default; accepted for backwards compatibility)
      --keep-waters         Retain waters matching the extraction selection
      --no-hetatm           Drop all hetero atoms, keeping only ATOM records
      --out-format string   Output format: pdb or cif (default: inferred from --output, else same as the input)
  -o, --output string       Output file (default: stdout)
```

### Examples

1. Extract chains A, B, and C to a file
```bash
$ pdbtk extract --chains A,B,C --output 1a02_chainABC.pdb 1a02.pdb
```

2. Extract chains A, B, and C to stdout
```bash
$ pdbtk extract --chains A,B,C 1a02.pdb > 1a02_chainABC.pdb
```

3. Extract from an mmCIF file, writing mmCIF
```bash
$ pdbtk extract --chains A,B --output 1a02_chainAB.cif 1a02.cif
```

4. Convert an mmCIF file to PDB
```bash
$ pdbtk extract --chains A --out-format pdb 1a02.cif > 1a02_chainA.pdb
```

5. Extract from stdin
```bash
$ cat 1a02.pdb | pdbtk extract --chains A,B,C
```

6. Extract only ALTLOC B atoms
```bash
$ pdbtk extract --chains A --altloc B 1a02.pdb
```

7. Extract first ALTLOC when duplicates exist
```bash
$ pdbtk extract --chains A --altloc first 1a02.pdb
```

8. Keep only the polymer, dropping ligands, ions and waters
```bash
$ pdbtk extract --chains A --no-hetatm 1a02.pdb
```

9. Keep waters alongside ligands and ions
```bash
$ pdbtk extract --chains A --keep-waters 1a02.pdb
```

10. Strip waters from a whole structure without selecting chains
```bash
$ pdbtk extract 1a02.pdb > 1a02_nowat.pdb
```

**Note on hetero atoms:** as of 0.2.0 hetero atoms belonging to the selected
chains are kept by default and waters are dropped. Earlier versions kept hetero
records appearing before a chain's `TER` but discarded those after it unless
`--keep-hetatm` was given. `--keep-hetatm` is still accepted and now has no
effect, since it describes the default.

## extract-seq Usage

```text
Extract sequences from chains in a PDB or PDBx/mmCIF structure file.
The output is in FASTA format with sequence IDs in the format: >{filename_no_ext}_{chain}

If no chains are specified, all chains will be extracted.
If no input file is specified, reads from stdin.

Usage:
  pdbtk extract-seq [flags] [input_file]

Flags:
      --chain string       Alias for --chains
  -c, --chains string      Comma-separated list of chain IDs to extract (default: all chains)
  -h, --help               help for extract-seq
      --in-format string   Input format: pdb or cif (default: inferred from the file extension or contents)
  -o, --output string      Output file (default: stdout)
      --seqres             Use SEQRES records instead of ATOM records
```

### Examples

1. Extract sequences from all chains
```bash
$ pdbtk extract-seq 1a02.pdb >1a02.fasta
```

2. Extract sequences from specific chains A, B, and C
```bash
$ pdbtk extract-seq --chains A,B,C 1a02.pdb >1a02_chainABC.fasta
```

3. Extract all chains to a file
```bash
$ pdbtk extract-seq --output 1a02_all.fasta 1a02.pdb
```

4. Extract from stdin
```bash
$ cat 1a02.pdb | pdbtk extract-seq --chains B,C
```

5. Extract sequences using SEQRES records
```bash
$ pdbtk extract-seq --seqres 1a02.pdb
```

6. Extract sequences using --chain alias
```bash
$ pdbtk extract-seq --chain A,B 1a02.pdb > 1a02_chainAB.fasta
```

7. Extract sequences from multiple PDB files in the current directory
```bash
$ find . -name "*.pdb" -exec pdbtk extract-seq {} \; > myseqs.fasta
```

8. Extract sequences from an mmCIF file
```bash
$ pdbtk extract-seq --chains A,B 1a02.cif > 1a02_chainAB.fasta
```

**Note on sequence extraction:**
- By default, `extract-seq` extracts sequences from ATOM records with gap characters (`-`) inserted for missing residue numbers.
- Use `--seqres` to extract from SEQRES records instead (which contain the full sequence including regions not present in ATOM records).
- If `--seqres` is specified but no SEQRES records are present, a warning is printed and no sequence is returned.
- For mmCIF input, `--seqres` reads `pdbx_poly_seq_scheme` (falling back to `entity_poly_seq`).
- Ligands, ions and waters are excluded; modified residues such as `MSE` are resolved to their parent residue.

## version Usage

```text
Print the version number of pdbtk.

Usage:
  pdbtk version [flags]

Flags:
  -h, --help   help for version
```

### Examples

Print the current version
```bash
$ pdbtk version
0.2.0
```

## completion Usage

```text
Generate the autocompletion script for the specified shell

Usage:
  pdbtk completion [command]

Available Commands:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh

Flags:
  -h, --help   help for completion

Use "pdbtk completion [command] --help" for more information about a command.
```

See [download.md](download.md#shell-completion) for more details.

## rename-chain Usage

```text
Rename a chain in a PDB or PDBx/mmCIF structure file.
If the specified chain does not exist, the command will exit with an error.
If the new chain ID already exists, a warning will be logged but the operation will continue.

Chain IDs longer than one character are allowed for mmCIF output; writing them
to a PDB file requires --force-lossy-pdb.

Usage:
  pdbtk rename-chain [flags] <chain_id> [input_file]

Flags:
  -h, --help                help for rename-chain
      --in-format string    Input format: pdb or cif (default: inferred from the file extension or contents)
      --out-format string   Output format: pdb or cif (default: inferred from --output, else same as the input)
  -o, --output string       Output file (default: stdout)
  -t, --to string           New chain ID (required)
```

### Examples

1. Rename chain A to B
```bash
$ pdbtk rename-chain A --to B 1a02.pdb
```

2. Rename chain A to B and output to a file
```bash
$ pdbtk rename-chain A --to B --output 1a02_renamed.pdb 1a02.pdb
```

3. Rename chain A to B from stdin
```bash
$ cat 1a02.pdb | pdbtk rename-chain A --to B
```

4. Rename to a multi-character chain ID, writing mmCIF
```bash
$ pdbtk rename-chain A --to HEAVY --output 1a02_renamed.cif 1a02.cif
```

## renumber-residues Usage

```text
Renumber residues in a PDB or PDBx/mmCIF structure file starting from a specified number.
By default, this preserves gaps in the residue sequence but offsets the numbering.
Use --force-sequential to make all residues sequential without gaps.
Use --exclude-zero to skip residue number zero when using negative start values.

Usage:
  pdbtk renumber-residues [flags] [input_file]

Flags:
  -c, --chain string        Chain ID to renumber (default: all chains)
  -z, --exclude-zero        Skip residue number zero when using negative start values
  -f, --force-sequential    Force sequential numbering without gaps
  -h, --help                help for renumber-residues
      --in-format string    Input format: pdb or cif (default: inferred from the file extension or contents)
      --out-format string   Output format: pdb or cif (default: inferred from --output, else same as the input)
  -o, --output string       Output file (default: stdout)
  -s, --start int           Starting residue number (can be negative) (default 1)
```

### Examples

1. Renumber all residues starting from 1
```bash
$ pdbtk renumber-residues --start 1 1a02.pdb
```

2. Renumber residues in chain A starting from 1
```bash
$ pdbtk renumber-residues --start 1 --chain A 1a02.pdb
```

3. Force sequential numbering starting from 1
```bash
$ pdbtk renumber-residues --start 1 --force-sequential 1a02.pdb
```

4. Renumber starting from negative number
```bash
$ pdbtk renumber-residues --start -10 1a02.pdb
```

5. Renumber starting from -1, skipping zero (goes -1, 1, 2, 3...)
```bash
$ pdbtk renumber-residues --start -1 --exclude-zero 1a02.pdb
```

6. Renumber and output to a file
```bash
$ pdbtk renumber-residues --start 1 --output 1a02_renumbered.pdb 1a02.pdb
```