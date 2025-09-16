# pdbtk

PDB (and PDBx/mmCIF) structure file manipulation toolkit (in the spirit of `seqtk`, `seqkit`, `csvtk`).

## Examples

```bash
# Extract chains from a PDB file
pdbtk extract --chains A,B,C --output 1a02_chainABC.pdb 1a02.pdb

# Extract chains to stdout
pdbtk extract --chains A,B,C 1a02.pdb >1a02_chainABC.pdb
```
