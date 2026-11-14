# LESSONS - auto-maintained by scripts/lessons.py

> Machine-owned. Do NOT hand-edit. Changes are overwritten on the next `lessons.py` write.
> Canonical state lives in `.specs/lessons.json`. Edit lessons only via the script.
> promote_threshold=2 distinct features · window_days=45 · quarantine_threshold=2

## Confirmed (load these at Specify/Design)

Corroborated across multiple features. Safe to apply as guidance.

_none_

## Candidates (under observation - do NOT load as guidance yet)

Seen once or not yet corroborated. Tracked, not trusted.

### L-001 - A test that wraps the call under test in a closure returning a hardcoded nil asserts nothing: capture the real return value so the exit-code contract is discriminated.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `cmd/mem` · harmful: 0
- features: fix-release-onnx-provider
- evidence: mutant F (cmd/mem/main.go:1711) (cmd/mem)
- last seen: 2026-11-13T23:08:16Z

### L-002 - Assert the pinned version and the final install path of a downloaded artifact, not only that the bytes arrived.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `internal/embedder` · harmful: 0
- features: fix-release-onnx-provider
- evidence: AC5 (internal/embedder/model_downloader.go:23) (internal/embedder)
- last seen: 2026-11-13T23:08:17Z

### L-003 - When the product advertises a label in the command output, assert that label at the command layer, not only at the internal library layer.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `cmd/mem` · harmful: 0
- features: fix-release-onnx-provider
- evidence: AC4 (.github/workflows/release.yml:123) (cmd/mem)
- last seen: 2026-11-13T23:08:17Z

### L-004 - When two acceptance criteria of the same feature count the same thing, name the unit once and reuse it in both.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: fix-release-onnx-provider
- evidence: AC9 (cmd/mem/main.go:1505) (spec)
- last seen: 2026-11-13T23:08:17Z

### L-005 - When a criterion only requires a message to declare a state, pin the wording in the spec; otherwise the test freezes an invented string.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: fix-release-onnx-provider
- evidence: AC10 (cmd/mem/main.go:1515) (spec)
- last seen: 2026-11-13T23:08:18Z

### L-006 - A vector search that reads TurboQuant rows is not killed by zeroing the raw vector column: assert the path the search actually reads, or the sensor reports a false green.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `cmd/mem` · harmful: 0
- features: fix-release-onnx-provider
- evidence: cmd/mem/main.go:2222 (cmd/mem)
- last seen: 2026-11-14T11:14:09Z

### L-007 - Toda chave array nova no payload de uma tool MCP precisa entrar na normalizacao da fronteira do structuredContent e no teste que proibe null: o contrato do wire e por chave, nao por tipo.
- signal: `gate_fail` · recurrence: 1 feature(s) · scope: `internal/mcp` · harmful: 0
- features: fix-doctor-unindexed-links
- evidence: internal/mcp/structured_content.go:65 (internal/mcp)
- last seen: 2026-11-14T18:27:21Z

### L-008 - Numero de Health Score citado em Out of Scope envelhece quando outra spec mexe na mesma formula: remeça antes de reutilizar o valor medido por outra spec.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: fix-doctor-unindexed-links
- evidence: .specs/features/fix-doctor-unindexed-links/spec.md:68 (spec)
- last seen: 2026-11-14T18:27:21Z

## Quarantined (failed when applied - ignore)

A confirmed lesson that recurred alongside failure. Kept for the maintainer to review.

_none_
