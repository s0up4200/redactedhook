# build app
FROM golang:1.22-alpine3.19 AS app-builder

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME

ENV SERVICE=redactedhook

WORKDIR /src

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of the source code
COPY . ./

RUN go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o bin/redactedhook cmd/redactedhook/main.go

# build runner
FROM gcr.io/distroless/static-debian12:nonroot

ENV HOME="/redactedhook" \
    XDG_CONFIG_HOME="/redactedhook" \
    XDG_DATA_HOME="/redactedhook"


WORKDIR /redactedhook

VOLUME /redactedhook

COPY --from=app-builder /src/bin/redactedhook /usr/local/bin/

EXPOSE 42135

ENTRYPOINT ["/usr/local/bin/redactedhook", "--config", "config.toml"]
