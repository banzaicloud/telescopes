# build stage
FROM golang:1.9.3-alpine3.7

ADD . /go/src/github.com/banzaicloud/spot-recommender
WORKDIR /go/src/github.com/banzaicloud/spot-recommender
RUN go build -o /bin/spot-recommender .

FROM alpine:latest
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=0 /bin/spot-recommender /bin
ENTRYPOINT ["/bin/spot-recommender"]
