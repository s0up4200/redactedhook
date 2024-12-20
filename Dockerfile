# build app
FROM golang:1.23-alpine3.20 AS app-builder

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME

ENV SERVICE=redactedhook

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN --network=none \
    go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o bin/redactedhook cmd/redactedhook/main.go

# build runner
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="redactedhook"
LABEL org.opencontainers.image.description="RedactedHook is a webhook companion service for autobrr."
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${REVISION}"
LABEL org.opencontainers.image.created="${BUILDTIME}"

ENV HOME="/redactedhook" \
    XDG_CONFIG_HOME="/redactedhook" \
    XDG_DATA_HOME="/redactedhook"

WORKDIR /redactedhook
VOLUME /redactedhook

EXPOSE 42135

COPY --from=app-builder /src/bin/redactedhook /usr/local/bin/

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/redactedhook", "--config", "config.toml"]
