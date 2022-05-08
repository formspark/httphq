FROM golang:1.18-alpine
WORKDIR /usr/src/app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY ./src ./src
RUN go build -o ./bin/go-project ./src
EXPOSE 8080
CMD [ "./bin/go-project" ]