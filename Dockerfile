FROM golang:1.16-alpine3.14 as build

ENV CGO_ENABLED=1
ENV GO111MODULE=on

RUN apk add --no-cache git gcc g++

# Set the Current Working Directory inside the container
WORKDIR /src

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o bin/etcdkeeper main.go

FROM alpine:3.14

WORKDIR /app

COPY --from=build /src/bin/etcdkeeper .

RUN chmod +x etcdkeeper

ENTRYPOINT ["./etcdkeeper"] 
