# API Contract Changelog

This file tracks wire-contract changes synced from the gateway. It is
machine-parseable; the gateway's `scripts/check-changelog.py` CI job expects
one entry here per wire-shape-changing ADR amendment.

Entry format:

```
## [gateway-vA.B.C] — YYYY-MM-DD
- kind: <added|changed|removed|deprecated|fixed|security>
  path: <HTTP path or gRPC RPC>
  adr:  <ADR reference>
  notes: <short human-readable note>
```

## [unreleased]
- kind: added
  path: (initial scaffold)
  adr: sdk-0001-repo-and-versioning
  notes: SDK v0 skeleton. No wire-contract entries yet; populated on first sync.
