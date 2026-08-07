# novel-spec

**A portable, repository-first specification for long-form fiction.**

One novel is one Git repository. Chapters remain readable Markdown, narrative
state remains explicit JSON, and Git provides history, diffs, rollback, and
portability between writing agents.

> Status: **v0.1 draft**. This is an early contract intended for experimentation
> and interoperability work.

## Why

Long-form fiction fails when its durable memory is trapped in a chat transcript
or an application's private database. Characters learn facts too early, objects
move without explanation, chronology drifts, foreshadowing disappears, and a
new session no longer knows what the previous one established.

novel-spec makes the repository—not the conversation—the source of truth:

- `.novel/state/` stores versioned character, world, timeline, POV, and
  foreshadowing state.
- `chapters/` stores prose as Markdown with machine-readable YAML frontmatter.
- `bible/`, `outline/`, and `style/` separate immutable facts, plans, and craft
  constraints.
- `history[].as_of_chapter` prevents later knowledge from leaking into earlier
  chapters.
- `state_after` links chapter prose to the state snapshot produced after it.

The full draft contract is in [SPEC.md](SPEC.md).

## Repository shape

```text
my-novel/
├── NOVEL.toml
├── .novel/
│   └── state/
│       ├── world.json
│       ├── characters/{id}.json
│       ├── timeline.json
│       ├── foreshadow.json
│       └── pov.json
├── bible/
├── outline/
├── chapters/v01-c001.md
├── style/
│   ├── voice.md
│   ├── anti-ai-rules.md
│   └── benchmarks/{author}/{scene-type}.md
└── reviews/
```

## Three core principles

1. **`.novel/state/` is the authority.** Chat history is not narrative memory.
2. **Prose stays Markdown.** Chapter metadata lives in YAML frontmatter.
3. **State is historical.** Character snapshots are selected by
   `as_of_chapter`, not only by a mutable `current` value.

## Validate a repository

Requires Go 1.25 or later.

```bash
go run ./cmd/novel-spec check examples/sample-novel
```

Expected output:

```text
novel-spec 0.1: examples/sample-novel OK
```

Build the validator:

```bash
go build -o novel-spec ./cmd/novel-spec
```

Then validate any novel repository:

```bash
./novel-spec check /path/to/my-novel
```

## Schemas

The v0.1 JSON Schema contracts are under [schema/](schema/):

- `novel.toml.json`
- `character.json`
- `foreshadow.json`
- `timeline.json`
- `chapter-frontmatter.json`

The CLI also enforces semantic constraints that JSON Schema alone does not
express, such as strictly increasing character history chapters.

## Reference implementation

[reasonix-novel](https://github.com/kael-odin/reasonix-novel) is the first
implementation. It packages a Reasonix runtime sidecar, an MCP server, a lead
writer skill, and three read-only editorial reviewers.

novel-spec itself is harness-independent. A Cherry Studio agent, StaffDeck
employee, Claude Code skill, Cursor rule set, or standalone program can
implement the same artifact and tool semantics.

## Example

[examples/sample-novel](examples/sample-novel) demonstrates:

- two chapters with valid frontmatter;
- historical state for two characters;
- planted and resolved foreshadowing;
- absolute and narrative timelines;
- project voice, anti-AI prose rules, and scene-specific benchmarks.

## Development

```bash
go test ./...
```

```bash
go vet ./...
```

## License

[MIT](LICENSE)
