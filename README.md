# pdbtk

A simple PDB (and PDBx/mmCIF) structure file manipulation toolkit (in the spirit of `seqtk`, `seqkit`, `csvtk`).

`pdbtk` (currently) strives to be practical over 100% standards compliant.

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