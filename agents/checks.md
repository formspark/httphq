# Checks

What the lint, coverage, dead-code and doc checks hold, and why each is shaped
the way it is. The one-line summary of every script is in
[docs/scripts.md](../docs/scripts.md).

## The coverage floor

`go:coverage:check` reads the total out of `coverage.out` and fails under 93%,
which is where the suite stands. AGENTS.md has the ratchet rule it follows. The
Go and the npm sides are measured
differently and the npm side has no suite to speak of, so there is one floor
here.

## Lint is clean, types are strict, and the caps only go down

What each linter bans, and why, is in its config: `.golangci.yml` for Go,
`eslint.config.mjs` for the JavaScript and TypeScript, and the tsconfigs for the
compiler flags. Two things the configs cannot say on their own:

- A value that arrives untyped in the Playwright suite, a JSON body or the
  clipboard, is parsed with a zod schema in `e2e/tests/support/harness.ts`.
- `scripts/check-doc-refs.mjs` is exempt from the three ESLint caps, because
  it is vendored and kept identical to its canonical copy.
  `eslint.config.mjs` carries that reason beside the entry.
- `check:package-json` fails when `package.json` is not in the order
  `sort-package-json` puts it in. lint-staged sorts it on commit; this is what
  catches a commit made with `--no-verify`, the same pairing `format` and
  `format:check` use.
- `check:lint-staged` fails when the tree holds a file extension Prettier can
  format that no lint-staged pattern matches. Prettier decides what counts, so
  the two cannot drift, and a gap would otherwise be silent: CI formats the
  file, the hook never touches it.

## Dead code is checked on both sides, separately

This repository is a Go application with a small JavaScript periphery, and the
two need different tools. Neither covers the other, so a quiet run of one says
nothing about the other.

`go:deadcode` runs `deadcode` from `golang.org/x/tools`, pinned the way
govulncheck, golangci-lint and gocyclo are. It builds the whole call graph from
`main` and reports every function nothing reaches. golangci-lint's `unused` is
already in the standard set `go:lint` runs, but it works a package at a time: it
catches an unexported helper nobody calls and cannot see an exported function
that no entry point reaches. This closes that. It prints findings and exits zero
on its own, so the script is the same shape `go:format:check` uses, failing on
non-empty output.

`check:dead-code` is [knip](https://knip.dev), over the JavaScript only: the
page scripts under `public/`, the Node scripts under `scripts/` with their
suites under `tests/`, and the Playwright suite in the `e2e` workspace. Both hold
at zero.

`knip.ts` says in its own header that it does not cover `src/`, because the
tempting mistake is to read a passing knip run as coverage of the application.
The `entry` patterns there are mostly `<script>` tags in the Go templates under
`src/views`, which no module graph can see.

One dependency is exempt, `playwright`, which a script resolves through a
`createRequire` rooted at another workspace so it is not installed twice.
`knip.ts` carries that reason beside the entry.

Neither runs from the pre-commit hook. lint-staged hands a task the staged paths,
and a whole-graph analysis over a subset reports almost everything as unused.

## Doc references

`check:doc-refs` reads the Markdown docs and fails on a relative link, an
anchor, a backticked repository path, a `pnpm` script or a `pnpm-workspace.yaml`
key that does not exist. A reference meant to dangle goes in
`doc-refs-allowlist.json` under `scripts/` with its reason, a file that exists
only once it holds an entry; prefer fixing the doc. An
entry that stops matching fails the check too, so the list only shrinks. CI
also runs lychee offline over the same Markdown, for links and their anchors.

A bare file name is reported only once git history shows it deleted, so a
shallow CI checkout can report fewer of those than a full local clone.
