# build stage
FROM golang:1.16-alpine3.13 AS builder
ENV GOFLAGS="-mod=readonly"

RUN apk add --update --no-cache ca-certificates make git curl mercurial

RUN mkdir -p /build
WORKDIR /build

COPY go.* /build/
RUN go mod download

COPY . /build
RUN BINARY_NAME=telescopes make build-release

FROM alpine:3.14.0

RUN apk add --update --no-cache ca-certificates tzdata bash curl

COPY --from=builder /build/build/release/telescopes /bin

ENTRYPOINT ["/bin/telescopes"]
CMD ["/bin/telescopes"]

