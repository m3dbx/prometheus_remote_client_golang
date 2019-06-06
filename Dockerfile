# Stage 1: build
FROM golang:1.12-alpine AS builder
LABEL maintainer="The M3DB Authors <m3db@googlegroups.com>"

# Install Glide
RUN apk add --update git

# Add source code
RUN mkdir -p /src/prometheus_remote_client_golang
ADD . /src/prometheus_remote_client_golang

# Build cli tool binary
RUN cd /src/prometheus_remote_client_golang && \
    go build github.com/m3db/prometheus_remote_client_golang/cmd/promremotecli

# Stage 2: lightweight "release"
FROM alpine:latest
LABEL maintainer="The M3DB Authors <m3db@googlegroups.com>"

COPY --from=builder /src/prometheus_remote_client_golang/promremotecli /bin/promremotecli

ENTRYPOINT [ "/bin/promremotecli" ]
