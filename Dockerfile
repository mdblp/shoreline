# Development
FROM golang:1.12.7-alpine AS development

RUN apk --no-cache update && \
    apk --no-cache upgrade && \
    apk add build-base git

ENV GO111MODULE on

WORKDIR /go/src/github.com/mdblp
RUN git clone --branch feature/pt582-0.4.1 https://github.com/mdblp/go-common.git

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
    adduser -D mdblp

WORKDIR /home/mdblp

USER mdblp

COPY --from=development --chown=mdblp /go/src/github.com/mdblp/shoreline/dist/shoreline .

CMD ["./shoreline"]
