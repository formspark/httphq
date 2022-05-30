# ***** Builder *****

FROM golang:1.18-alpine as builder

WORKDIR /usr/src/app

RUN apk add build-base

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./public ./public
COPY ./src ./src

RUN go build -o ./bin/httphq ./src

# ***** Application *****

FROM alpine

COPY --from=builder ./usr/src/app/bin ./bin
COPY --from=builder ./usr/src/app/public ./public
COPY --from=builder ./usr/src/app/src ./src

ENV APPLICATION_ENV=production

EXPOSE 8080

CMD ["./bin/httphq"]