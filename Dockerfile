

# build app
FROM golang:1.20-alpine3.16 AS app-builder

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME

RUN apk add --no-cache git make build-base tzdata

ENV SERVICE=redactedhook

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

#ENV GOOS=linux
#ENV CGO_ENABLED=0

RUN go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o bin/redactedhook ./main.go

# build runner
FROM alpine:latest

LABEL org.opencontainers.image.source = "https://github.com/s0up4200/redactedhook"

ENV HOME="/config" \
XDG_CONFIG_HOME="/config" \
XDG_DATA_HOME="/config"

RUN apk --no-cache add ca-certificates curl tzdata jq

WORKDIR /app

VOLUME /config

COPY --from=app-builder /src/bin/redactedhook /usr/local/bin/

EXPOSE 42135

ENTRYPOINT ["/usr/local/bin/redactedhook"]
#CMD ["--config", "/config"]