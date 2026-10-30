# CodeRabbit review fixes

Scope: address PR #1 review on 81b6884, without merging or publishing a version.

- [x] Reproduce and fix YAML merge precedence, scalar alias keys, duplicate normalized keys, cycle detection, and excessive alias expansion. Conversion budgets: 100,000 node visits, 16 MiB expanded scalar text including keys, 256 nested nodes; check cancellation before decoding and during traversal.
- [x] Preserve --help/-h as option values; align CLI JSON validation with duplicate-key rejection; stop acronym lookahead at separators.
- [x] Register shared JSON tools across subgroups and call the selected validator while retaining the previous data result.
- [x] Use explicit preview IDs, preserve all built-in preview eligibility, flush only pending edits on Escape, retain status/footer on short terminals.
- [x] Align RTK/PRD with the authorized five-item ID-only recent history. Document timestamp inference limits rather than adding a new unit API.
- [x] Add requested meaningful coverage for nested duplicates, JWT malformed/duplicate payloads and preserved fields, null MIME, header value order and token non-determinism. Tighten release tag validation for leading-zero components and malformed prereleases.
- [x] Run local verification and independent review. Full tests, race, format, vet, build, actionlint, GoReleaser snapshot, four archive/checksum checks and native packaged CLI/TUI smoke checks passed. Independent review identified the large-scalar amplification hole; the added scalar-byte/depth regression tests now pass.

Review thread 4423949772 contains attribution text but no actionable defect; its agent prompt explicitly says no code issue is described. No change was made to runPathQuery for that thread.

GitHub CI and automated re-review run after pushing the fix commit. No PR merge, version tag or release is included in this task.
