# Usage and Examples

## Quick Guide

- **Download PDB files**: [get](#get-usage)
- **Coordinate extraction**: [extract](#extract-usage)
- **Sequence extraction**: [extract-seq](#extract-seq-usage)
- **Chain manipulation**: [rename-chain](#rename-chain-usage), [renumber-residues](#renumber-residues-usage)
- **Version info**: [version](#version-usage)
- **Other**: [completion](#completion-usage)

## pdbtk Usage

```text
pdbtk -- a cross-platform, efficient and practical PDB structure file manipulation toolkit

Version: 0.1.1
Author: Perry
Source code: https://github.com/perry/pdbtk

pdbtk is a command-line toolkit for manipulating PDB structure files.
It provides various operations for extracting, filtering, and transforming protein structure data.

Usage:
  pdbtk [command]

Available Commands:
  get               Download a PDB file from the RCSB PDB database
  extract           Extract chains from a PDB file
  extract-seq       Extract sequences from chains in a PDB file
  rename-chain      Rename a chain in a PDB file
  renumber-residues Renumber residues in a PDB file
  version           Print the version number
  completion        Generate the autocompletion script for the specified shell
  help              Help about any command

Flags:
  -h, --help   help for pdbtk

Use "pdbtk [command] --help" for more information about a command.
```

## get Usage

```text
Download a PDB file from the RCSB PDB database using the PDB code.
The file will be downloaded from https://files.rcsb.org/download/{pdb_code}.{format}

By default, the file is saved as {pdb_code}.pdb in the current directory.
Use --output to specify a different filename or "-" to output to stdout.
Use --format to specify the file format (pdb, pdb.gz).

Usage:
  pdbtk get [flags] <pdb_code>

Flags:
  -f, --format string   File format: pdb, pdb.gz (default: pdb)
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
Extract specific chains from a PDB structure file.
The output can be written to a file or stdout (if no output file is specified).
If no input file is specified, reads from stdin.

Usage:
  pdbtk extract [flags] [input_file]

Flags:
  -c, --chains string   Comma-separated list of chain IDs to extract (required)
      --chain string    Alias for --chains
  -h, --help            help for extract
  -o, --output string   Output file (default: stdout)
      --altloc string   Filter by alternative location (ALTLOC) identifier (e.g., A, B) or 'first' to take first ALTLOC when duplicates exist
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

3. Extract from stdin
```bash
$ cat 1a02.pdb | pdbtk extract --chains A,B,C
```

4. Extract only ALTLOC B atoms
```bash
$ pdbtk extract --chains A --altloc B 1a02.pdb
```

5. Extract first ALTLOC when duplicates exist
```bash
$ pdbtk extract --chains A --altloc first 1a02.pdb
```

6. Extract using --chain alias
```bash
$ pdbtk extract --chain A,B,C --output 1a02_chainABC.pdb 1a02.pdb
```

## extract-seq Usage

```text
Extract sequences from chains in a PDB structure file.
The output is in FASTA format with sequence IDs in the format: >{pdbfilename_no_dotpdb}_{chain}

If no chains are specified, all chains will be extracted.
If no input file is specified, reads from stdin.

Usage:
  pdbtk extract-seq [flags] [input_file]

Flags:
  -c, --chains string   Comma-separated list of chain IDs to extract (default: all chains)
      --chain string    Alias for --chains
  -h, --help            help for extract-seq
  -o, --output string   Output file (default: stdout)
      --seqres          Use SEQRES records instead of ATOM records
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

**Note on sequence extraction:**
- By default, `extract-seq` extracts sequences from ATOM records with gap characters (`-`) inserted for missing residue numbers.
- Use `--seqres` to extract from SEQRES records instead (which contain the full sequence including regions not present in ATOM records).
- If `--seqres` is specified but no SEQRES records are present, a warning is printed and no sequence is returned.

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
0.1.1
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
Rename a chain in a PDB structure file.
The chain ID must be a single character. The new chain ID must also be a single character.
If the specified chain does not exist, the command will exit with an error.
If the new chain ID already exists, a warning will be logged but the operation will continue.

Usage:
  pdbtk rename-chain [flags] <chain_id> [input_file]

Flags:
  -h, --help            help for rename-chain
  -o, --output string   Output file (default: stdout)
  -t, --to string       New chain ID (required)
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

## renumber-residues Usage

```text
Renumber residues in a PDB structure file starting from a specified number.
By default, this preserves gaps in the residue sequence but offsets the numbering.
Use --force-sequential to make all residues sequential without gaps.
Use --exclude-zero to skip residue number zero when using negative start values.

Usage:
  pdbtk renumber-residues [flags] [input_file]

Flags:
  -s, --start int          Starting residue number (can be negative) (default 1)
  -c, --chain string       Chain ID to renumber (default: all chains)
  -z, --exclude-zero       Skip residue number zero when using negative start values
  -f, --force-sequential   Force sequential numbering without gaps
  -h, --help               help for renumber-residues
  -o, --output string      Output file (default: stdout)
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