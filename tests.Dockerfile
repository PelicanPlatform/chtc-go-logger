FROM golang:1.23
# Run as non-root

WORKDIR /app
COPY --chown=1000:1000 go.mod go.mod
COPY --chown=1000:1000 go.sum go.sum
RUN go mod download

RUN mkdir /.cache && chown 1000 /.cache && chown 1000 /app

COPY --chown=1000:1000 cmd/ cmd/
COPY --chown=1000:1000 config/ config/
COPY --chown=1000:1000 logger/ logger/

USER 1000
CMD go test -v ./...
