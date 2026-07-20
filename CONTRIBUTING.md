# Contributing to OpenLogs

Thanks for your interest in improving OpenLogs! This project is built with
**spec-driven development** using [OpenSpec](https://github.com/Fission-AI/OpenSpec).
Every change — features, behavioural fixes, refactors that alter behaviour — is
planned as a set of spec artifacts *before* any code is written, and ships with
tests. Please read this document before opening a pull request.

## The rules in short

1. **Every contribution is an OpenSpec change** with four artifacts: a
   **proposal**, a **design**, **specs**, and **tasks**.
2. **Every contribution includes unit tests** covering the new or changed behaviour.
3. Code follows the existing conventions and passes `go build`, `go vet`, and
   `go test ./...`.

PRs that add or change behaviour without the spec artifacts or without tests will
be asked to add them before review.

## Why spec-driven development?

Planning in specs keeps the "what" and "why" separate from the "how", makes
review easier, and leaves a durable record of intent. The specs are the source of
truth for how OpenLogs behaves; the code implements them.

## The workflow

Changes live under `openspec/changes/<change-name>/`. Each change contains:

```
openspec/changes/<change-name>/
├── proposal.md      # WHY: the problem, what changes, which capabilities
├── design.md        # HOW: key technical decisions, trade-offs, alternatives
├── specs/           # WHAT: one spec.md per capability, with requirements + scenarios
│   └── <capability>/spec.md
└── tasks.md         # Implementation checklist
```

The OpenSpec CLI (and the `/opsx:*` skills, if you use them) scaffold and validate
these for you:

```bash
# 1. Explore the idea (optional, thinking only — no code)
/opsx:explore

# 2. Create a change and generate all artifacts
openspec new change <change-name>
#    or: /opsx:propose <description>

# 3. Implement against the tasks
#    /opsx:apply <change-name>

# 4. After it merges and ships, archive the change
#    /opsx:archive <change-name>
```

You can check progress at any time:

```bash
openspec status --change <change-name>
openspec validate --change <change-name>
```

### 1. Proposal (`proposal.md`)

States **why** the change is needed and **what** changes at a high level. It must
list the **capabilities** being added or modified — each new capability maps to a
`specs/<capability>/spec.md` file. Keep it focused on the problem, not the
implementation.

### 2. Design (`design.md`)

Explains **how** you intend to build it: the key technical decisions with their
rationale, alternatives considered, and known risks/trade-offs. This is where
architectural discussion belongs so it can be reviewed before code exists.

### 3. Specs (`specs/<capability>/spec.md`)

Define **what** the system must do as testable requirements. Each requirement uses
normative language (SHALL/MUST) and has at least one scenario in WHEN/THEN form:

```markdown
## ADDED Requirements

### Requirement: <name>
The system SHALL ...

#### Scenario: <name>
- **WHEN** <condition>
- **THEN** <expected outcome>
```

Scenarios are effectively your test cases — write them so they can be verified.

### 4. Tasks (`tasks.md`)

A dependency-ordered checklist of implementation steps as checkboxes
(`- [ ] 1.1 ...`). Mark them `- [x]` as you complete them.

## Testing requirements

**All contributions must include unit tests.** Every requirement/scenario you add
to a spec should be backed by a test that verifies it. Tests live alongside the
code they cover (e.g. `internal/db/db_test.go`, `internal/handler/handler_test.go`)
and use Go's standard `testing` package.

Before opening a PR, make sure the following all pass:

```bash
go build ./...
go vet ./...
go test ./...
```

Guidelines:

- Add tests in the same PR as the change — not a follow-up.
- Cover the happy path *and* the error/edge cases described in your scenarios.
- Prefer fast, hermetic tests (use `t.TempDir()` for a throwaway SQLite database;
  use `httptest` for HTTP handlers, as the existing tests do).
- If a behaviour is hard to unit test, note in the PR how you verified it.

## Code conventions

- **Go**, formatted with `gofmt` (run `gofmt -w` or `go fmt ./...`).
- Match the style of the surrounding code: comment density, naming, and idioms.
- No new heavyweight dependencies without discussion in the proposal/design —
  OpenLogs deliberately stays lightweight (single binary, pure-Go SQLite, HTMX,
  hand-written CSS, no frontend build step).
- Keep changes minimal and scoped to the tasks in your change.

## Pull request checklist

Before requesting review, confirm your PR includes:

- [ ] An OpenSpec change under `openspec/changes/<change-name>/` with
      `proposal.md`, `design.md`, `specs/`, and `tasks.md`
- [ ] `openspec validate --change <change-name>` passes
- [ ] Unit tests covering the new/changed behaviour
- [ ] `go build ./...`, `go vet ./...`, and `go test ./...` all pass
- [ ] Documentation updated (e.g. `README.md`) if behaviour or configuration changed

## Questions

If you're unsure how to scope a change or structure its specs, open a draft PR or
an issue describing the idea (or start with `/opsx:explore`). We'd rather discuss
the shape of a change early than after the code is written.
