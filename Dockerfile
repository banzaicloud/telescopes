# build stage
FROM golang:1.9.3-alpine3.7

ADD . /go/src/github.com/banzaicloud/telescopes
WORKDIR /go/src/github.com/banzaicloud/telescopes
RUN go build -o /bin/telescopes ./cmd/telescopes

FROM alpine:latest
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=0 /bin/telescopes /bin
ENTRYPOINT ["/bin/telescopes"]
