# build stage
FROM golang:1.9.3-alpine3.7

ADD . /go/src/github.com/banzaicloud/cluster-recommender
WORKDIR /go/src/github.com/banzaicloud/cluster-recommender
RUN go build -o /bin/cluster-recommender .

FROM alpine:latest
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=0 /bin/cluster-recommender /bin
ENTRYPOINT ["/bin/cluster-recommender"]
