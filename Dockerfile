# Development
FROM golang:1.12.17-alpine AS development
WORKDIR /go/src/github.com/mdblp/shoreline
RUN apk --no-cache update && \
    apk --no-cache upgrade && \
    apk add --no-cache bash git ca-certificates && \
    adduser -D shoreline && \
    chown -v shoreline /go/src/github.com/mdblp/shoreline
# Using Go module (go 1.12 need this variable to be set to enable modules)
# The variable should default to "on", in Go 1.14 release
ENV GO111MODULE on
COPY . .
RUN /bin/bash -eu build.sh
CMD ["./dist/shoreline"]

# Production
FROM alpine:latest AS production
RUN apk --no-cache update && \
    apk --no-cache upgrade && \
    apk add --no-cache ca-certificates && \
    adduser -D shoreline
WORKDIR /app
COPY --from=development --chown=root:root /go/src/github.com/mdblp/shoreline/dist/shoreline .
USER shoreline
ENTRYPOINT ["/app/shoreline"]
