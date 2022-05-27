# go-project

https://go-project.fly.dev/

## Scripts

Run project locally:

```bash
go run ./src/application.go
```

Run tests locally:

```bash
watchman-make -p 'src/**/*.go' --make=go -t test ./...
```

View test coverage:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Upkeep project dependencies:

```bash
go mod tidy
```

Format project:

```bash
go fmt ./src && npx prettier --write .
```

Build and run binary locally:

```bash
go build -o ./bin/go-project ./src
./bin/go-project
```

Build and run container locally:

```bash
docker build . -t go-project
docker run -dp 8080:8080 go-project
docker container ls -s
```

Run E2E tests:

```bash
cd e2e && npx cypress open
```

Deploy to Fly:

```bash
fly deploy
```

View Fly application logs:

```bash
fly logs
```
