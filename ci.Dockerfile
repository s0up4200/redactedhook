# syntax=docker/dockerfile:1

# Build stage
FROM --platform=$BUILDPLATFORM golang:1.23-alpine3.20 AS app-builder

# Install git for revision info and ca-certificates for potential downloads
RUN apk add --no-cache git ca-certificates tzdata

# Create non-root user for build
RUN adduser -D -g '' appuser

WORKDIR /src

# Copy dependency files first for better cache utilization
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Copy rest of the files
COPY . .

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME
ARG TARGETOS TARGETARCH

# Build with security flags and proper versioning
# Network is disabled during build
RUN --network=none --mount=target=. \
    BUILDTIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    REVISION=$(git rev-parse --short HEAD) \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.buildDate=${BUILDTIME} -extldflags '-static'" \
    -o /out/bin/redactedhook cmd/redactedhook/main.go

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.source="https://github.com/s0up4200/redactedhook"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.base.name="distroless/static-debian12:nonroot"
LABEL org.opencontainers.image.description="RedactedHook CI image"

# Set environment variables
ENV HOME="/redactedhook" \
    XDG_CONFIG_HOME="/redactedhook" \
    XDG_DATA_HOME="/redactedhook"

WORKDIR /redactedhook
VOLUME /redactedhook

# Copy the binary from builder
COPY --from=app-builder /out/bin/redactedhook /usr/local/bin/

# Expose the application port
EXPOSE 42135

# Use nonroot user
USER nonroot:nonroot

# Set entry point
ENTRYPOINT ["/usr/local/bin/redactedhook", "--config", "config.toml"]
