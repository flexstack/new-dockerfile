FROM docker.io/library/golang:1.21 AS builder

ARG GOPROXY=direct
WORKDIR /app
COPY . .
RUN GOPROXY=${GOPROXY} CGO_ENABLED=0 go build -o new-dockerfile cmd/new-dockerfile/main.go

FROM docker.io/library/alpine:3.19.1

WORKDIR /app
COPY --from=builder /app/new-dockerfile /usr/local/bin

CMD [ "new-dockerfile" ]
