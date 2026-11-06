# Font license records

`OFL-1.1.txt` is the canonical SIL Open Font License 1.1 text retained with
this acquisition kit. `GNU-Unifont-NOTICE.txt` is a tracked, SHA-256-locked
attribution record for the exact GNU Unifont 17.0.05 files in the catalog. It
states their copyright attribution, dual-license metadata, upstream license
URL, and this catalog's selected OFL distribution route.

`fetch_fonts.py` also downloads the immutable upstream notices listed in
`../font-lock.json` into this directory, including the complete GNU Unifont
`LICENSE.txt`. They are deliberately ignored by Git because they are
reproducible staging output. The lock binds each downloaded notice and the
tracked GNU attribution record to a SHA-256 digest before staging.

`package_fonts.py` includes every staged legal record in the generated
`kvm-ui-fonts.tar.gz` rootfs archive. Extract that one artifact at root as
described in `../README.md`; it places the records under
`/usr/share/fonts/LICENSES/` alongside the catalog fonts.

The approved catalog is distributed as unmodified upstream font files under
the OFL 1.1 route. This record is implementation evidence, not legal advice.
