# Scripts

Install dependencies:

```bash
go mod download
npm install
cd e2e && npm install && npx playwright install chromium
```

Upkeep dependencies:

```bash
go mod tidy
```

Build the stylesheet:

```bash
npm run css
```

`public/app.css` is generated from `src/styles/app.css` and is committed, so the
binary, the container, and `go run` never need Node. Rebuild it whenever a
template or a script gains a class the sheet does not already carry, and commit
the result; CI fails if the committed file differs from a fresh build. During
design work, `npm run css:watch` regenerates on save.

Run project:

```bash
go run ./src
```

Lint the page scripts and the Playwright suite:

```bash
npm run lint
npm run lint:fix
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
currently holds: `-over 5` for Go, and the `complexity` rule in
`eslint.config.mjs` at 6 for the page scripts and the Playwright suite. `-top`
takes no position and is the one to run when deciding what to simplify next.
Neither ceiling grandfathers anything, so lower them as hotspots go.

Run E2E tests (Playwright auto-starts the binary; build it first):

```bash
CGO_ENABLED=0 go build -o ./bin/httphq ./src
cd e2e && npx playwright test
```

The suite drives a real browser against a real binary, so it needs the port that
binary listens on. `port` in `src/application.go` is a constant; to run beside
something already holding 8080, change it and point the suite at the same place:

```bash
HTTPHQ_BASE_URL=http://localhost:8099 npx playwright test
```

View test coverage:

```bash
go test ./src/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Format project:

```bash
go fmt ./src && npx prettier --write .
```

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
