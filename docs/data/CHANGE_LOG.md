# Data Change Log

Design decisions and significant data model changes. See `docs/data/data_format.md` for current schema.

| Date | PR | Change | Reason |
|---|---|---|---|
| 2026-02-28 | #18 | Flattened receipt format; moved admin control into handlers | Nesting added complexity with no benefit; flat schema easier to query and extend |
| 2026-02-28 | #13 | Added persistent token table | Refresh tokens need to survive restarts; placeholder for future Redis migration |
| 2026-03-12 | #32 | Clarified extra fields / meta field semantics with examples | Field naming was ambiguous; examples added to prevent misuse by API consumers |
| 2026-03-15 | #33 | Added PostgreSQL schema alongside SQLite | Production target is PostgreSQL; both schemas must remain behaviourally identical — no SQLite-only features |
| 2026-03-15 | #34 | Added pagination metadata to list response shape | Clients need total count + page info to render paginated UIs without extra requests |
