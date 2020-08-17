# Development
FROM golang:1.12-alpine AS development
# Using Go module (go 1.12 need this variable to be set to enable modules)
# The variable should default to "on", in Go 1.14 release
ENV GO111MODULE on
WORKDIR /go/src/github.com/tidepool-org/shoreline
RUN adduser -D tidepool && \
    chown -R tidepool /go/src/github.com/tidepool-org/shoreline
RUN apk add --no-cache git
USER tidepool
COPY --chown=tidepool go.* ./
RUN go get
COPY --chown=tidepool . .
RUN ./build.sh
CMD ["./dist/shoreline"]

# Production
FROM alpine:latest AS production
WORKDIR /app
RUN apk --no-cache update && \
    apk --no-cache upgrade && \
    apk add --no-cache ca-certificates && \
    adduser -D tidepool
USER tidepool
COPY --from=development --chown=root /go/src/github.com/tidepool-org/shoreline/dist/shoreline .
ENTRYPOINT ["/app/shoreline"]
