# Development
FROM golang:1.12.7-alpine AS development

RUN apk --no-cache update && \
    apk --no-cache upgrade && \
    apk add build-base git

# Using Go module (go 1.12 need this variable to be set to enable modules)
ENV GO111MODULE on

WORKDIR /go/src/github.com/mdblp/shoreline
COPY . .
RUN go get
RUN ./build.sh

CMD ["./dist/shoreline"]

# Release
FROM alpine:latest AS release

RUN apk --no-cache update && \
    apk --no-cache upgrade && \
    apk add --no-cache ca-certificates && \
    adduser -D tidepool

WORKDIR /home/tidepool

USER tidepool

COPY --from=development --chown=tidepool /go/src/github.com/mdblp/shoreline/dist/shoreline .

CMD ["./shoreline"]
