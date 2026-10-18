# AGENTS.md

Guidance for agents and contributors working in this repository. This file is
the constraints that are not obvious from the source. The other docs each own
one thing:

- README.md: what httphq is, and its configuration for self-hosters.
- docs/scripts.md: setup, every script, the stylesheet, the build and running
  the image.
- docs/api.md: the JSON API.
- DESIGN.md: the visual system. PRODUCT.md: the product and its principles.
- `agents/`: a recipe per area, listed at the end of this file.

## `pnpm verify` is the definition of done

It runs every step of `scripts.verify` in package.json. A change that has not
passed it is not finished. CI runs the same package.json scripts, split into
jobs in `pipeline.yml`, and also builds and scans the image and checks the
Markdown links offline.

The spelling check runs typos through `uvx`, so it needs uv, and `check:audit`
reaches the registry. The Playwright suite needs Chromium once
(docs/scripts.md) and loads Alpine and the syntax highlighter from public CDNs,
so a screen that never hydrates is a network failure before it is a code one:
run it again before debugging it.

## Boundaries

- Always: run `pnpm verify` before calling a change done. Fix every instance
  of a defect, not only the one that was reported. See a regression test fail
  without the fix before trusting it: take the fix out, watch the test go
  red, put it back. Unenforced: reviewer's call.
- Always: rebuild `public/app.css` with `pnpm run css` when a template or a
  script gains a class, and commit it with the change. `pnpm css:check`
  enforces it.
- Ask first: a new third-party origin on any page, or anything that moves one
  onto the home page (DESIGN.md, Typography); anything that makes captures
  durable or adds accounts (PRODUCT.md, Product Principles); editing a vendored
  check under `scripts/` (agents/testing.md).
- Never: skip the pre-commit hook with `--no-verify`; lower the coverage floor
  or raise a complexity cap to make a change pass; add a check to `test.yml`
  or `release.yml` instead of `pipeline.yml`.

## Comments say what the code does now

A comment explains why, and names the constraint the next person has to
respect: "excluded by default so new surfaces are safe". Write it so it still
makes sense in a year. Unenforced: reviewer's call.

- No history: not what the code used to do, not the bug that prompted it. That
  belongs in the commit message, where it is attached to the change.
- No ticket or issue references.
- A regression risk is worth stating as a rule rather than a story: "a header
  set on the way in is discarded by a handler that resets the response".
- No hyper-specific framing: describe the generic, reusable purpose rather than
  the one-time scenario, especially in shared helpers and test fixtures.

## Commits say what the change does

Unenforced: reviewer's call, apart from the hook.

- The subject is a sentence in the imperative that says what the change does:
  "Refuse an oversized body instead of storing a bodyless capture", not
  `fix: refuse oversized body`. No conventional-commit prefixes.
- Pull requests are squash-merged, with the pull request number as a `(#N)`
  suffix on the subject.
- The body carries facts: why, what was measured, what was checked and how,
  and what was deliberately left out. Not a summary of the diff, which the
  reader already has. The comment rules above govern commit bodies and pull
  request descriptions too.
- No assistant attribution of any kind: no co-author trailer, no session link,
  no generated-with footer.
- One concern per commit. The pre-commit hook formats and lints what is
  staged, Go files included.

## Writing

No em dashes in prose: docs, comments, UI copy and commit messages. Rewrite the
clause instead. A paired aside becomes commas or parentheses; a dash that
introduces an explanation becomes a colon or a new sentence. `check:prose`
fails on one in any tracked file; commit messages are the reviewer's call.

User-facing copy names the format or the action, never a consumer such as
agents or LLMs: `Copy request`, `Copy shown (N)`. The constraint is on product
copy only; commit messages and docs may mention agents freely. Unenforced:
reviewer's call.

A doc that names a path, an anchor, a `pnpm` script or a `pnpm-workspace.yaml`
key names one that exists. `pnpm check:doc-refs` enforces it
(agents/checks.md).

## Tooling policy

- Lint is clean and types are strict: `pnpm lint` runs with
  `--max-warnings 0`, and what each linter bans is explained in its own config
  (`.golangci.yml`, `eslint.config.mjs`, the tsconfigs).
- An exception goes in the tool's config, scoped to the file and the rule,
  with the reason beside it: `eslint.config.mjs` for lint, where inline
  `eslint-disable` is switched off and reported, `.golangci.yml` for Go, and
  `knip.ts` for dead code.
- The caps only go down. `gocyclo -over 4` for Go, and `complexity`,
  `max-params` and `max-depth` in ESLint, sit at the worst function that
  exists, and the 93% coverage floor in `go:coverage:check` sits where the
  suite stands. When a new function trips a cap, split the function; lower a
  cap when its worst offender is split, and raise the floor when coverage
  rises. `pnpm go:cyclo`, `pnpm lint` and `pnpm go:coverage:check` enforce the
  current values; moving them only one way is the reviewer's call.
- Dead code is checked on both sides, separately: `go:deadcode` and
  golangci-lint for Go, knip for the JavaScript. A quiet run of one says
  nothing about the other (agents/checks.md).
- An advisory that has no fix, or cannot be reached from this code, goes in
  `audit.ignore` in `pnpm-workspace.yaml` with its reason
  (agents/dependencies.md).

## Invariants and traps

- The toolchain pins move together; agents/dependencies.md lists them.
  Unenforced: reviewer's call.
- Every Go command is scoped to `./src`, never `./...`; docs/scripts.md says
  why. Unenforced: reviewer's call.
- `public/app.css` is generated and committed, so the binary and the container
  need no Node. `pnpm css:check` enforces its freshness.
- A new background task that writes to the store has to stop before the store
  closes on SIGTERM (agents/go-application.md). Unenforced: reviewer's call.
- `newApplication` builds the routing surface from arguments; anything that
  reads the environment belongs in `main`, not in a handler. Unenforced:
  reviewer's call.
- `PLATFORM` trusts its platform's client-IP header unconditionally
  ([README.md](README.md#platform)). `platform_test.go` holds each strategy.
- Headers and bodies are never logged, and paths are logged without their
  query string (agents/go-application.md). The tests in `src/logging` hold
  the redaction backstop; the rest is unenforced: reviewer's call.
- DESIGN.md's palette and `src/styles/theme.css` are held together by
  `src/design_test.go` (agents/testing.md).

## Reading CI

- `pipeline.yml` is the single definition of what verified means, and both
  `test.yml` (pull requests and a nightly run) and `release.yml` (pushes to
  master) call it. Adding a check means adding a script to package.json, to
  `verify`, and to `pipeline.yml`.
- The nightly run exists because the page scripts pull Alpine and highlight.js
  from public CDNs at runtime, so the suite can fail on a day nobody pushed.
  `audit.yml` asks the same of the dependencies and the base image once a
  week.
- Read the checks table, not a watcher's exit code:
  `gh pr checks <n> --watch --fail-fast` can exit 0 with a failed run in the
  table.
- `gh pr merge --auto` does not wait here. Auto-merge is disabled on this
  repository, so the flag is accepted and the merge happens immediately.
- A cancelled job hit its `timeout-minutes` cap, which is not a regression.
  Check with `gh api repos/{owner}/{repo}/actions/jobs/<id> --jq '.steps[]'`
  before treating a timeout as a code problem.

## Recipes

- [agents/go-application.md](agents/go-application.md): Open when you touch
  `src/*.go`: the code layout, configuration, shutdown, client IP and logging.
- [agents/database.md](agents/database.md): Open when you touch
  `src/database/`: the schema standards test and why captures are ephemeral.
- [agents/testing.md](agents/testing.md): Open when you touch `*_test.go`,
  `e2e/` or `tests/`: how the Go, Playwright and script suites are laid out.
- [agents/checks.md](agents/checks.md): Open when you touch `eslint.config.mjs`,
  `.golangci.yml`, `knip.ts` or `scripts/check-*.mjs`: coverage, lint, dead
  code and doc references.
- [agents/dependencies.md](agents/dependencies.md): Open when you touch
  `package.json`, `pnpm-workspace.yaml`, `go.mod` or `.tool-versions`:
  advisories, pins and release age.
