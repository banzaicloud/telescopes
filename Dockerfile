# build stage
FROM golang:1.9.3-alpine3.7 as backend
RUN apk update && apk add ca-certificates curl git make tzdata

RUN mkdir -p /go/src/github.com/banzaicloud/telescopes
ADD Gopkg.* Makefile /go/src/github.com/banzaicloud/telescopes/
WORKDIR /go/src/github.com/banzaicloud/telescopes
RUN make vendor
ADD . /go/src/github.com/banzaicloud/telescopes

RUN go build -o /bin/telescopes ./cmd/telescopes



FROM alpine:3.7
COPY --from=backend /usr/share/zoneinfo/ /usr/share/zoneinfo/
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=backend /bin/telescopes /bin



ENTRYPOINT ["/bin/telescopes"]

