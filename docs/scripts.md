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

Run E2E tests (Playwright auto-starts the binary; build it first):

```bash
CGO_ENABLED=0 go build -o ./bin/httphq ./src
cd e2e && npx playwright test
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
