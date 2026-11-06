# Font catalog release-compliance kit

This directory is the reproducible acquisition kit and release-compliance
gate for the device font catalog. Fetched font bytes are not tracked in
Git and are never embedded in the `kvm_ui` executable.

## Rules

- `kvm-ui-fonts.tar.gz` is the only supported device release payload for
  this catalog and its legal records.
- Never edit downloaded font binaries or bump a version in place; an
  intentional update means a reviewed lock change, recalculated hashes,
  a license-record review, and a newly generated package.
- Extract the single archive at the filesystem root
  (`sudo tar -xzf kvm-ui-fonts.tar.gz -C /`) so the fonts and their legal
  records install under `/usr/share/fonts/`.
- Fetch and package with the `fetch_fonts.py` and `package_fonts.py`
  scripts in this directory.

## Legal records

All fonts are distributed under their recorded SIL Open Font License
terms; Reserved Font Names apply to modified versions only. The tracked
compliance records live in [LICENSES/](LICENSES/README.md).

See [README.md](../README.md) for the project overview.
