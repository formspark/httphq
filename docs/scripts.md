# Scripts

Install dependencies:

```bash
go mod download
pnpm install
pnpm --filter httphq-e2e exec playwright install chromium
```

This is a pnpm workspace of two projects, the root tooling and the Playwright
suite in `e2e`, so one install at the root covers both. The browser download is
separate because pnpm installs the Playwright package, not the Chromium it
drives, and it is filtered to `e2e` because that is the project Playwright
belongs to: a bare `pnpm exec playwright` at the root does not resolve.

Upkeep dependencies:

```bash
go mod tidy
```

## Every check is a script

`pnpm verify` runs every step of `scripts.verify` in package.json, and CI calls
the same scripts, split into jobs, so a tree that passes locally passes in the
pipeline.

| Script                    | What it does                                                                         |
| ------------------------- | ------------------------------------------------------------------------------------ |
| `pnpm dev`                | `go run ./src`, bound to localhost                                                   |
| `pnpm build`              | The production binary in `bin/httphq`, with the image's flags                        |
| `pnpm test`               | The Go tests, with the race detector                                                 |
| `pnpm test:coverage`      | The same, writing `coverage.out` and printing the total                              |
| `pnpm test:e2e`           | The Playwright suite, against `bin/httphq`                                           |
| `pnpm test:scripts`       | The Vitest suites for the checks under `scripts/`                                    |
| `pnpm go:vuln`            | govulncheck: advisories whose vulnerable symbol this code actually reaches           |
| `pnpm go:format:check`    | Fails when `gofmt` would rewrite a file                                              |
| `pnpm go:lint`            | golangci-lint, which runs `go vet` among its linters                                 |
| `pnpm go:deadcode`        | Fails on a Go function that no entry point reaches                                   |
| `pnpm go:cyclo`           | gocyclo's ceiling for the Go code                                                    |
| `pnpm go:coverage:check`  | Fails when the total in `coverage.out` is under the 93% floor                        |
| `pnpm lint`               | ESLint, with warnings failing like errors                                            |
| `pnpm format:check`       | Fails when Prettier would rewrite a file                                             |
| `pnpm check:lint-staged`  | Fails when Prettier formats an extension the pre-commit hook does not cover          |
| `pnpm check:package-json` | Fails when `package.json` is not in `sort-package-json` order                        |
| `pnpm check:prose`        | Fails on an em dash in any tracked file (AGENTS.md, "Writing")                       |
| `pnpm check:typos`        | typos, through `uvx`, so it needs uv installed                                       |
| `pnpm check:doc-refs`     | Fails on a doc reference to a path, anchor, script or config key that does not exist |
| `pnpm check:dead-code`    | knip: unused files, exports and dependencies in the JavaScript                       |
| `pnpm check:audit`        | `pnpm audit` over production dependencies, failing at high severity                  |
| `pnpm typecheck`          | Type-checks three of the page scripts and the two test suites written in TypeScript  |
| `pnpm css:check`          | Rebuilds the stylesheet and fails if the committed one differs                       |

Every Go tool is pinned in its script, so a later release cannot change the
verdict on a tree that did not change. Every Go command is scoped to `./src`:
`./...` also walks `node_modules`, which ships a stray Go package once the npm
tooling is installed.

## The stylesheet

```bash
pnpm run css
```

`public/app.css` is generated from `src/styles/app.css` and is committed, so the
binary, the container, and `go run` never need Node. Rebuild it whenever a
template or a script gains a class the sheet does not already carry, and commit
the result; `css:check` fails otherwise. During design work,
`pnpm run css:watch` regenerates on save.

## Types and lint

The type check covers the Playwright suite, the Vitest suites under
`tests/scripts`, and three of the page scripts:
`index.js`, `render-body.js` and `har.js`, checked as JavaScript against the
globals declared in `types/page-scripts.d.ts`. The Playwright suite reads the
same declarations, so a helper and the tests that call it cannot disagree about its
signature. `endpoint.js` is left out, because its Alpine component reads
`this.$el`, which only Alpine's own component typing can describe.

The Playwright suite is linted with type information, so `e2e` needs its own
dependencies installed.

Both languages carry a complexity ceiling, set at the worst score the tree
currently holds: `-over 4` for Go, and `complexity` at 4 in `eslint.config.mjs`
for the page scripts, the Playwright suite and the Node scripts. The JavaScript
side carries two more set the same way, `max-params` at 3 and `max-depth` at 2.
To decide what to simplify next:

```bash
go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -top 10 ./src
```

## The Playwright suite

`pnpm test:e2e` starts `bin/httphq` itself, so build it first with
`pnpm build`. The suite drives a real browser against a real binary, so it needs
the port that binary listens on. `port` in `src/application.go` is a constant;
to run beside something already holding 8080, change it and point the suite at
the same place:

```bash
HTTPHQ_BASE_URL=http://localhost:8099 pnpm test:e2e
```

## Coverage

```bash
pnpm test:coverage
go tool cover -html=coverage.out
```

CI prints the total to the job summary, and `go:coverage:check` fails when it
drops under the 93% floor.

## Formatting

```bash
go fmt ./src/... && pnpm run format
```

The pre-commit hook formats what is staged, Go files included, so the checks
above are rarely what tells you. `prettier --write .` is not always a fixed
point: a single pass can leave a file that `--check` still rejects, and a second
pass converges it. Anything that scripts a format step should check afterwards
rather than assume one pass settled it.

## Build and run

```bash
pnpm build
./bin/httphq
```

`APP_VERSION` names the build at link time, and `/api/health` reports it; a
build without it reports `dev`.

```bash
docker build . -t httphq --build-arg APP_VERSION=local
docker run -dp 8080:8080 httphq
```
