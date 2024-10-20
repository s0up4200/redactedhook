# build app
FROM --platform=$BUILDPLATFORM golang:1.23-alpine3.20  AS app-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME
ARG TARGETOS TARGETARCH

RUN --network=none --mount=target=. \
    BUILDTIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    REVISION=$(git rev-parse --short HEAD) \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.buildDate=${BUILDTIME}" \
    -o /out/bin/redactedhook cmd/redactedhook/main.go

# build runner
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.source = "https://github.com/s0up4200/redactedhook"
LABEL org.opencontainers.image.licenses = "MIT"
LABEL org.opencontainers.image.base.name = "distroless/static-debian12:nonroot"

ENV HOME="/redactedhook" \
    XDG_CONFIG_HOME="/redactedhook" \
    XDG_DATA_HOME="/redactedhook"

WORKDIR /redactedhook
VOLUME /redactedhook

EXPOSE 42135

COPY --from=app-builder /out/bin/redactedhook /usr/local/bin/

USER nobody
ENTRYPOINT ["/usr/local/bin/redactedhook", "--config", "config.toml"]