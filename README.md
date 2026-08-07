# pdbtk

A simple PDB and PDBx/mmCIF structure file manipulation toolkit (in the spirit of `seqtk`, `seqkit`, `csvtk`).

`pdbtk` (currently) strives to be practical over 100% standards compliant.

Docs: https://pansapiens.github.io/pdbtk/

## Examples

```bash
# Extract chains from a PDB file
pdbtk extract --chains A,B,C --output 1a02_chainABC.pdb 1a02.pdb

# Extract chains to stdout
pdbtk extract --chains A,B,C 1a02.pdb >1a02_chainABC.pdb

# Extract sequences from a PDB file (one record per chain)
pdbtk extract-seq 1a02.pdb >1a02.fasta

# Extract sequences for specific chains
pdbtk extract-seq --chains A,B,C 1a02.pdb >1a02_chainABC.fasta

# The same operations work on mmCIF input
pdbtk extract --chains A,B 1a02.cif >1a02_chainAB.cif

# ...and pdbtk doubles as a format converter
pdbtk extract --out-format pdb 1a02.cif >1a02.pdb
pdbtk extract --output 1a02.cif 1a02.pdb
```

## Building

```bash
go build -o bin/pdbtk .

# or
# make build
```

### Building the documentation

```bash
cd doc
uv venv
source .venv/bin/activate
uv pip install mkdocs mkdocs-material mkdocs-macros-plugin

# To view locally
# mkdocs serve

mkdocs build
```