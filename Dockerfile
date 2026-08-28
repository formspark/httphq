# ***** Builder *****

FROM golang:1.26-alpine AS builder

# The tag tracks a minor line and can sit behind the patch go.mod names, and the
# official images set GOTOOLCHAIN=local, which refuses to close that gap. This
# lets the build fetch what go.mod asks for, so the version lives in one place
# rather than in a tag that has to be edited alongside it.
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
