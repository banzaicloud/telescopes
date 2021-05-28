# build stage
FROM golang:1.12.3-alpine AS builder
ENV GOFLAGS="-mod=readonly"

RUN apk add --update --no-cache ca-certificates make git curl mercurial bzr

RUN mkdir -p /build
WORKDIR /build

COPY go.* /build/
RUN go mod download

COPY . /build
RUN BINARY_NAME=telescopes make build-release

FROM alpine:3.9.3

RUN apk add --update --no-cache ca-certificates tzdata bash curl

COPY --from=builder /build/build/release/telescopes /bin

ENTRYPOINT ["/bin/telescopes"]
CMD ["/bin/telescopes"]

