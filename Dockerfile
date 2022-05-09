FROM golang:1.18-alpine

WORKDIR /usr/src/app

RUN apk add build-base

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./public ./public
COPY ./src ./src

ENV APPLICATION_ENV=production
ENV GIT_SHA=${GIT_SHA}

RUN go build -o ./bin/go-project ./src

EXPOSE 8080

CMD [ "./bin/go-project" ]