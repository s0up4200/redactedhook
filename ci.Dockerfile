# build app
FROM --platform=$BUILDPLATFORM golang:1.20-alpine3.16 AS app-builder

# Set work directory
WORKDIR /src

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of the source code
COPY . ./

# Build arguments
ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME
ARG TARGETOS TARGETARCH

# Build the application
RUN --mount=target=. \
    BUILDTIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    REVISION=$(git rev-parse --short HEAD) \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.buildDate=${BUILDTIME}" \
    -o /out/bin/redactedhook cmd/redactedhook/main.go

# build runner
FROM gcr.io/distroless/base-debian12

# Set metadata and environment variables
LABEL org.opencontainers.image.source = "https://github.com/s0up4200/redactedhook"
ENV HOME="/redactedhook" \
    XDG_CONFIG_HOME="/redactedhook" \
    XDG_DATA_HOME="/redactedhook"

# Set work directory and expose necessary ports
WORKDIR /redactedhook
VOLUME /redactedhook
EXPOSE 42135

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/redactedhook", "--config", "config.toml"]

# Copy binary from app-builder
COPY --from=app-builder /out/bin/redactedhook /usr/local/bin/