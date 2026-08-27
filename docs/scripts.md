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

Build the stylesheet:

```bash
pnpm run css
```

`public/app.css` is generated from `src/styles/app.css` and is committed, so the
binary, the container, and `go run` never need Node. Rebuild it whenever a
template or a script gains a class the sheet does not already carry, and commit
the result; CI fails if the committed file differs from a fresh build. During
design work, `pnpm run css:watch` regenerates on save.

Run project:

```bash
go run ./src
```

Lint the page scripts and the Playwright suite:

```bash
pnpm run lint
pnpm run lint:fix
```

The Playwright suite is linted with type information, so `e2e` needs its own
dependencies installed. The Go application is checked by `go vet` instead.

Run unit tests:

```bash
go vet ./src/...
go test ./src/...
```

Every Go package lives under `src`. `./...` also walks `node_modules`, which
ships a stray Go package once the npm tooling is installed.

Check cyclomatic complexity:

```bash
go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -over 5 ./src
go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -top 10 ./src
```

Both sides carry a ceiling, and both are set at the worst score the tree
currently holds: `-over 5` for Go, and `complexity` at 6 in `eslint.config.mjs`
for the page scripts and the Playwright suite. The JavaScript side carries two
more ratchets set the same way, `max-params` at 3 and `max-depth` at 2. `-top`
takes no position and is the one to run when deciding what to simplify next.
No ceiling grandfathers anything, so lower them as hotspots go.

Run E2E tests (Playwright auto-starts the binary; build it first):

```bash
CGO_ENABLED=0 go build -o ./bin/httphq ./src
pnpm --filter httphq-e2e exec playwright test
```

The suite drives a real browser against a real binary, so it needs the port that
binary listens on. `port` in `src/application.go` is a constant; to run beside
something already holding 8080, change it and point the suite at the same place:

```bash
HTTPHQ_BASE_URL=http://localhost:8099 pnpm --filter httphq-e2e exec playwright test
```

View test coverage:

```bash
go test ./src/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

CI runs the same profile and prints the total to the job summary. It is reported
rather than gated: a threshold set before the number is known is a guess.

Format project:

```bash
go fmt ./src && pnpm run format
```

Both sides are enforced. CI runs `gofmt -l ./src` and `prettier --check .` and
fails on any file either one would rewrite, and a pre-commit hook formats what
is staged so the gate is rarely what tells you. Prettier is a dependency of this
package now rather than an ad hoc download, so everyone runs the same version.

Build and run binary:

```bash
CGO_ENABLED=0 go build -o ./bin/httphq ./src
./bin/httphq
```

Build and run container:

```bash
docker build . -t httphq
docker run -dp 8080:8080 httphq
docker container ls -s
```

One thing worth knowing about the formatter: `prettier --write .` is not always a
fixed point. A single pass has left files that `--check` still rejected, with a
second pass converging them. Anything that scripts a format step should check
afterwards rather than assume one pass settled it.
