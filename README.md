# go-project

https://go-project.fly.dev/

## Scripts

Run project locally:

```bash
go run ./src/application.go
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
go run ./src/application.go & (cd e2e && npx cypress open)
```

Deploy to Fly:

```bash
fly deploy
```

View Fly application logs:

```bash
fly logs
```
