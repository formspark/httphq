# Scripts

Install dependencies:

```bash
go mod download
cd e2e && npm install
```

Upkeep dependencies:

```bash
go mod tidy
```

Run project:

```bash
go run ./src/application.go
```

Run tests:

```bash
watchman-make -p 'src/**/*.go' --make=go -t test ./...
```

Run E2E tests:

```bash
cd e2e && npx cypress open
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
go build -o ./bin/httphq ./src
./bin/httphq
```

Build and run container:

```bash
docker build . -t httphq
docker run -dp 8080:8080 httphq
docker container ls -s
```

Deploy to Fly:

```bash
fly deploy
```

View Fly logs:

```bash
fly logs
```
