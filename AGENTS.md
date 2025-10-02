# Project Instructions

## Code style

- Do not write extraneous comments in the code; comment where code deviates from typical patterns or is particularly complex / cryptic. Don't comment on lines that would be self-explanatory to a junior developer.
- Secrets: never hardcode; always use environment variables and `.env` files.

## Documentation

- Update the documentation in the `doc/docs` folder whenever a new command or --flag is added or modified in behavior. Each subcommand should be documented in `doc/docs/usage.md`.

## CHANGELOG.md

After any bugfix, change in behaviour or an notable new feature has been successfully implemented, update the CHANGELOG.md file. Keep the notes short and concise. Follow the style of https://keepachangelog.com/en/1.1.0/

## PR checklist

- Run `make fmt` and `make test`
- Update documentation in `doc/docs` with any changes to the usage of a command or flag.
- Update `CHANGELOG.md` for notable features and fixes

## Reference documentation

 - For the (legacy) PDB format, refer to:
   - https://www.cgl.ucsf.edu/chimera/docs/UsersGuide/tutorials/pdbintro.html
   - https://www.wwpdb.org/documentation/file-format-content/format33/v3.3.html
   - https://www.wwpdb.org/documentation/file-format-content/format33/sect9.html
   - https://www.wwpdb.org/documentation/file-format-content/format33/sect4.html
   - Documented quirks in the PDB files found in the wild: https://proteopedia.org/wiki/index.php/Unusual_sequence_numbering
 - For the PDBx/mmCIF format, refer to:
   - https://mmcif.wwpdb.org/docs/user-guide/guide.html
   - https://pdb101.rcsb.org/learn/guide-to-understanding-pdb-data/beginner%E2%80%99s-guide-to-pdbx-mmcif

## Reference source code for dependencies

- The folder `.repos` might contain reference code that you can use to understand the dependencies used.
- DO NOT modify the code in `.repos`, it's not part of the project, just there for your reference.