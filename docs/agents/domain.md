# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the
codebase.

> **This repo does not follow the default layout.** The glossary is at `docs/Steering/CONTEXT.md`,
> not at the repo root. Read that file, not a root `CONTEXT.md` — there isn't one.

## Before exploring, read these

- **`docs/Steering/CONTEXT.md`** — the glossary. Ubiquitous language for the claims domain: business
  line segmentation, claim lifecycle, settlement values, reinsurance advices (PLA / Pre-DLA /
  DLA), roles, and the four distinct status concepts. Terms carry an origin marker:
  `[BISNIS]` confirmed by the business owner · `[KODE]` inferred from source · `[TERBUKA]` still
  unresolved.

- **`docs/Steering/00-DECISION-LOG.md`** — **72 recorded decisions** across three sessions
  (`D-01` … `D-72`). Each entry holds the question asked, the options offered, the answer given,
  the resulting decision, and which Steering documents it affects. See "Decision Log vs ADRs"
  below before you write an ADR.

- **`docs/ADR/`** — **29 Architecture Decision Records** (24 `Accepted`, 5 `Proposed`). Read the
  ADRs touching the area you are about to work in, starting from `docs/ADR/README.md`.
  **Five are `Proposed`** — they carry context and options but **no `Keputusan` section**, and
  must not be used as a basis for implementation.

- **`docs/Steering/21-RIWAYAT-REVISI.md`** — what changed between Steering v1.0 and v2.0 and why.
  Read this before trusting any number you remember from an earlier read.

There is no `CONTEXT-MAP.md`: this is a single-context repo.

All of these files **do exist**. `docs/ADR/` is no longer created lazily — it is populated and
under review.

## File structure

```
/
├── docs/
│   ├── AGENTS.md
│   ├── Steering/
│   │   ├── CONTEXT.md              ← the glossary (NOT at repo root)
│   │   ├── 00-DECISION-LOG.md      ← 72 decisions (D-01 … D-72)
│   │   ├── 01-FRONTEND-ANALYSIS.md … 21-RIWAYAT-REVISI.md
│   │   └── STEERING.md             ← BUILT artifact — do not edit directly
│   ├── ADR/
│   │   ├── README.md               ← index of the 29 ADRs
│   │   ├── 0001-*.md … 0029-*.md
│   │   └── ADR.md                  ← BUILT artifact — do not edit directly
│   ├── BRD/BRD.md
│   ├── ticketing/                  ← work tickets (D-31)
│   ├── tools/                      ← document generators
│   └── agents/                     ← this file and its siblings
└── …                               ← Pega rule XML export, by rule type (READ-ONLY)
```

## Decision Log vs ADRs

Two records of decisions live in this repo. Keeping them apart matters — two places holding the
same decision will eventually disagree, which is exactly what a glossary and a decision log exist
to prevent.

| | `docs/Steering/00-DECISION-LOG.md` | `docs/ADR/` |
|---|---|---|
| Covers | Migration discovery decisions | Architecture decisions made while implementing |
| Period | Before implementation started | During and after implementation |
| Status | **Final.** Do not rewrite | Living |
| Format | Question · options offered · answer · resulting decision · affected documents | Standard ADR |

**Rules:**

1. **Read the Decision Log before writing an ADR.** If the decision is already recorded there,
   do not restate it as an ADR — reference its ID (`D-14`, `D-22`) instead.
2. **Never edit a Decision Log entry to reflect a later change of mind.** Its value is that it
   records what was decided and why *at that time*. Write a new ADR that supersedes it, and say
   which decision ID it supersedes.
3. **Surface contradictions explicitly.** If your work conflicts with a logged decision, say so
   rather than quietly working around it:

   > _Contradicts D-20 (single portable SQL set), but worth reopening because…_

4. **Open decisions are marked in the log.** Entries carrying `OPEN` still need input from
   outside the development team. Check `Steering/README.md` for the current list before assuming
   something is settled.

## Use the glossary's vocabulary

When your output names a domain concept — an issue title, a refactor proposal, a hypothesis, a
test name, a table name, a Go type, an API path — use the term as defined in
`Steering/CONTEXT.md`. Don't drift to synonyms the glossary explicitly avoids.

The glossary carries a section listing terms **deliberately abandoned** from the Pega system,
because they were misleading or collided with programming meaning:

| Abandoned | Use instead |
|---|---|
| `Object` / `ObjectList` | Objek Pertanggungan / Insured Item |
| `Adjustment` / `AdjustmentList` | Settlement Line |
| `CaseID` | An explicit name for the specific context |
| `pzInsKey` / `ASM-FW-GCNMFW-WORK …` prefix | Nomor Klaim as the business identity |

Using an abandoned term reintroduces the confusion the migration exists to remove.

If the concept you need isn't in the glossary yet, that's a signal: either you're inventing
language the project doesn't use (reconsider) or there's a real gap (note it for
`/domain-modeling`).

## Language convention

Domain terms are in **Indonesian**, because that is the language the business and the team
actually use. Technical terms follow **English** Go and TypeScript convention. A type named
`Klaim` with a method `Validate()` is correct and intentional — see
`Steering/08-TECHNICAL-STRATEGY.md` §4.1.
