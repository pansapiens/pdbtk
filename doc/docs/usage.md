# Usage and Examples

## Quick Guide

- **Download PDB files**: [get](#get-usage)
- **Coordinate extraction**: [extract](#extract-usage)
- **Sequence extraction**: [extract-seq](#extract-seq-usage)
- **Other**: [completion](#completion-usage)

## pdbtk Usage

```text
pdbtk -- a cross-platform, efficient and practical PDB structure file manipulation toolkit

Version: 0.1
Author: Perry
Source code: https://github.com/perry/pdbtk

pdbtk is a command-line toolkit for manipulating PDB structure files.
It provides various operations for extracting, filtering, and transforming protein structure data.

Usage:
  pdbtk [command]

Available Commands:
  get         Download a PDB file from the RCSB PDB database
  extract     Extract chains from a PDB file
  extract-seq Extract sequences from chains in a PDB file
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command

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
  -h, --help            help for extract
  -o, --output string   Output file (default: stdout)
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
  -h, --help            help for extract-seq
  -o, --output string   Output file (default: stdout)
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