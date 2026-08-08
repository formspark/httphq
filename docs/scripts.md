# Scripts

Install dependencies:

```bash
go mod download
cd e2e && npm install && npx playwright install chromium
```

Upkeep dependencies:

```bash
go mod tidy
```

Run project:

```bash
go run ./src
```

Run unit tests:

```bash
go test ./...
```

Run E2E tests (Playwright auto-starts the binary; build it first):

```bash
CGO_ENABLED=0 go build -o ./bin/httphq ./src
cd e2e && npx playwright test
```

View test coverage:

```bash
go test ./... -coverprofile=coverage.out
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
