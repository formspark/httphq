# ***** Builder *****

FROM golang:1.25-alpine AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY ./src ./src

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ./bin/httphq ./src

# ***** Application *****

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /usr/src/app/bin/httphq ./bin/httphq
COPY ./public ./public
COPY ./src/views ./src/views

ENV APPLICATION_ENV=production

EXPOSE 8080

CMD ["./bin/httphq"]
