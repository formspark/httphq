# ***** Builder *****

FROM golang:1.26-alpine AS builder

# The image tag tracks the 1.26 line but lags its newest patch, and go.mod names
# the patch that carries the standard library fixes. The official images set
# GOTOOLCHAIN=local, which refuses to close that gap, so this lets the build
# fetch exactly what go.mod asks for rather than pinning the tag to a patch that
# does not exist yet.
ENV GOTOOLCHAIN=auto

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

RUN addgroup -S -g 1001 httphq \
 && adduser  -S -u 1001 -G httphq httphq \
 && chown -R httphq:httphq /app

USER httphq

ENV APPLICATION_ENV=production

EXPOSE 8080

CMD ["./bin/httphq"]
