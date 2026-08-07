# Novel Spec (novel-spec)

> A **repository-is-a-novel** specification. One novel = one git repository. Any
> compliant agent can read the repo, continue the story, review it, or migrate
> it — without depending on any particular tool's private database.
>
> Status: **v0.1 — draft**. The contract is the directory layout, the JSON
> Schemas in `schema/`, and the three principles below. Change the contract
> first, then the tools.

## Why

OpenAPI makes "an API" a standard artifact any client can consume. MCP makes
"a tool" a standard capability any agent can call. **novel-spec makes "a novel"
a standard artifact any agent can create, continue, or review** — stored as an
ordinary git repository, diffable, rollbackable, portable across agent
harnesses.

The problem it solves: long-form fiction breaks when the conversation ends.
State lives in a chat app's private context window, the model forgets facts
across chapters, and switching tools means losing the work. novel-spec moves
all narrative state into version-controlled files. **Dialogue is responsible
for creation, not memory.** The repo is the memory.

## The three principles

### 1. `.novel/state/` is the single source of truth

Every narrative fact — who characters are, what they know, where they are, what
foreshadow is planted, what the timeline is — lives in `.novel/state/`. The
prose in `chapters/` and the static setting in `bible/` are *projections* of
this state, never the authority. An agent entering the repo reads state first,
does its work, then updates state. No narrative memory lives in conversation
history.

### 2. Chapter prose carries metadata in frontmatter, body stays pure Markdown

```markdown
---
id: v01-c003
volume: 1
chapter: 3
word_count: 3120
state_after: a1b2c3d
foreshadow_planted: [fs-012, fs-013]
foreshadow_resolved: [fs-008]
status: draft
---
(prose body in plain Markdown...)
```

The frontmatter is machine-readable state linkage (`state_after` is the git
commit hash of the state snapshot after this chapter was written); the body
stays human-readable prose. See `schema/chapter-frontmatter.json`.

### 3. State is versioned with `as_of_chapter`

This is the key to solving long-form forgetting. A character's state is not
"current" — it is "as of chapter N". When chapter 50 recalls something from
chapter 10, the agent queries the state **as it was at chapter 10**, not the
latest. A protagonist who didn't know a secret at chapter 10 must not leak it
when remembering chapter 10.

```json
{
  "id": "protagonist",
  "current": { "...state after the latest chapter..." },
  "history": [
    { "as_of_chapter": 10, "snapshot": { "...state at chapter 10..." }, "commit": "a1b2c3d" },
    { "as_of_chapter": 25, "snapshot": { "...state at chapter 25..." }, "commit": "d4e5f6g" }
  ]
}
```

Each chapter write appends a `history` entry. The `commit` field ties the
snapshot to a git commit for full reproducibility.

## Repository layout

```
my-novel/
├── NOVEL.toml                      # spec version + metadata (title/genre/language/modules)
├── .novel/
│   ├── spec.json                   # novel-spec version this repo conforms to
│   └── state/                      # the single source of truth (principle 1)
│       ├── world.json              # world snapshot
│       ├── characters/
│       │   ├── {id}.json           # character: current + history[] (principle 3)
│       │   └── _index.json         # character id → name mapping
│       ├── timeline.json           # events (absolute time + narrative time)
│       ├── foreshadow.json         # planted_at / resolved_at / status
│       ├── consistency.json        # latest consistency-check report
│       └── pov.json                # viewpoint / narrative focus current state
├── bible/                          # static setting (immutable facts)
│   ├── worldbuilding.md
│   ├── magic-system.md
│   └── factions.md
├── outline/                        # outline tree (volume → stage → chapter)
│   └── volume-01/
│       ├── stage-01.md
│       └── chapter-001.md          # chapter outline (NOT prose)
├── chapters/                       # prose
│   └── v01-c001.md                 # frontmatter metadata + pure Markdown body
├── style/                          # style assets
│   ├── voice.md                    # voice/tone sample
│   ├── anti-ai-rules.md            # anti-AI-cliché rules (blacklist phrases/patterns)
│   └── benchmarks/                 # reference-author deconstruction samples
│       └── {author}/
│           ├── scene-description.md
│           ├── dialogue.md
│           ├── combat.md
│           ├── psychology.md
│           ├── transition.md
│           └── voice-profile.json  # sentence-length dist / punctuation habits / banned words
└── reviews/                        # review reports (output of each review pass)
    └── v01-c003-{timestamp}.json
```

### Directory semantics

| Path | Mutability | Purpose |
|---|---|---|
| `NOVEL.toml` | rarely | Book-level metadata and module declarations |
| `.novel/state/` | per-chapter | **Authority.** Updated after every chapter write. |
| `bible/` | mostly static | Immutable world facts. Changed only by deliberate retcon. |
| `outline/` | per-arc | Chapter outlines, written before prose. |
| `chapters/` | per-chapter | Prose. Frontmatter links to state snapshot. |
| `style/` | rarely | Voice samples and anti-AI rules. |
| `reviews/` | append-only | Each review pass writes a timestamped report. |

## NOVEL.toml

```toml
spec_version = "0.1"

[book]
title = "My Novel"
genre = "scifi"          # free-form; informs which modules apply
language = "zh"          # ISO code of the prose language

# Optional type-specific modules. Core spec is genre-agnostic;
# modules add optional files under .novel/state/ or bible/.
[modules]
# webnovel = true        # → pacing/cool-points.json
# scifi = true            # → bible/tech-tree.md
# mystery = true          # → state/mysteries.json
```

## Core schemas (genre-agnostic)

All schemas live in `schema/` and are JSON Schema (draft 2020-12).

- **`novel.toml.json`** — validates `NOVEL.toml` structure.
- **`character.json`** — the `current` + `history[as_of_chapter]` shape. Fields:
  `id`, `name`, `current` (location, psychology, knowledge[], inventory[],
  relationships{}, status), `history[]` (as_of_chapter, snapshot, commit).
- **`foreshadow.json`** — list of foreshadow entries: `id`, `planted_at`
  (chapter id), `resolved_at` (chapter id or null), `status`
  (planted | resolved | abandoned), `description`.
- **`timeline.json`** — events: `id`, `chapter`, `absolute_time`, `narrative_time`,
  `description`, `participants[]`.
- **`chapter-frontmatter.json`** — validates chapter file frontmatter: `id`,
  `volume`, `chapter`, `word_count`, `state_after`, `foreshadow_planted[]`,
  `foreshadow_resolved[]`, `status` (draft | reviewed | final).

## Optional modules

Core spec is type-agnostic. Genre-specific concerns are optional modules
declared in `NOVEL.toml`. A module adds optional files; their absence is valid.

| Module | Adds | Purpose |
|---|---|---|
| `webnovel` | `pacing/cool-points.json` | 爽点 (satisfaction-point) rhythm tracking |
| `scifi` | `bible/tech-tree.md` | Technology dependency tree |
| `mystery` | `.novel/state/mysteries.json` | Mystery vs. revealed-truth progress |

A repo self-describes which modules it uses via `NOVEL.toml [modules]`. A
compliant agent reads this to know which optional state to maintain.

## Tool semantics (conventional, not enforced)

novel-spec defines the *artifact*, not the agent. But for interoperability,
three tool semantics are conventional — any compliant agent SHOULD provide
equivalents:

- **`inject_context(chapter_id)`** — before writing chapter N, gather the
  characters/locations/foreshadow relevant to N, take their state **as of
  chapter N-1**, and attach the matching style samples. Read-only on state.
- **`update_state(chapter_text)`** — after writing a chapter, extract factual
  changes and write them back to `.novel/state/`, appending a `history` entry.
  Write-only on state.
- **`check_consistency(chapter_id)`** — compare prose against the state
  snapshot it claims (`state_after`) and report contradictions to
  `.novel/state/consistency.json`. Read + write `consistency.json`.

These can be MCP tools, extension-protocol tools, or agent skills — novel-spec
does not prescribe the mechanism, only the file contracts they read/write.

## Conformance

A repo is **novel-spec v0.1 conformant** if:

1. It has a `NOVEL.toml` with `spec_version = "0.1"`.
2. It has a `.novel/state/` directory containing at least `world.json` and
   `characters/_index.json`.
3. Every file under `chapters/` has frontmatter validating against
   `schema/chapter-frontmatter.json`.
4. Every character file under `.novel/state/characters/` validates against
   `schema/character.json` (including the `history[as_of_chapter]` shape).
5. Declared modules' files, if present, are valid.

The `novel-spec check <path>` CLI enforces all of the above.

## Portability

novel-spec is harness-agnostic. The reference implementation is a Reasonix
sidecar extension (`reasonix-novel`), but any agent that can read/write files
and follow the three principles can be a compliant agent — a Claude Code skill
pack, a Cursor rule set, an AionUi custom assistant, or a standalone script.
The novel outlives the tool.

## Versioning

This is v0.1. The contract will evolve. Schemas are versioned by
`spec_version` in `NOVEL.toml`; breaking changes bump the minor version
(0.1 → 0.2) and a migration note appears in `CHANGELOG.md`. Within a minor,
only optional fields are added.
