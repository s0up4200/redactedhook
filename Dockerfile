# build app
FROM golang:1.20-alpine3.16 AS app-builder

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME

# Install only necessary packages for the build
RUN apk add --no-cache git tzdata

ENV SERVICE=redactedhook

WORKDIR /src

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of the source code
COPY . ./

RUN go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o bin/redactedhook cmd/redactedhook/main.go

# build runner
FROM alpine:latest

LABEL org.opencontainers.image.source = "https://github.com/s0up4200/redactedhook"

ENV HOME="/redactedhook" \
    XDG_CONFIG_HOME="/redactedhook" \
    XDG_DATA_HOME="/redactedhook"

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl tzdata jq

WORKDIR /redactedhook

VOLUME /redactedhook

COPY --from=app-builder /src/bin/redactedhook /usr/local/bin/

EXPOSE 42135

ENTRYPOINT ["/usr/local/bin/redactedhook", "--config", "config.toml"]
