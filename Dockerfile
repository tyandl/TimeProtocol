# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY TimeProtocol.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/timeprotocol .

# distroless "nonroot" runs as uid/gid 65532 with no shell and no package
# manager, keeping the runtime image's attack surface minimal.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Port 37 is a privileged port (<1024) and would require root or
# CAP_NET_BIND_SERVICE to bind directly. Listening on an unprivileged port
# here lets the container run fully unprivileged; map it to host port 37
# via Docker's port publishing instead (see docker-compose.yml).
ENV TIMEPROTOCOL_PORT=8037
EXPOSE 8037/tcp
EXPOSE 8037/udp

COPY --from=builder /out/timeprotocol /timeprotocol

USER nonroot:nonroot
ENTRYPOINT ["/timeprotocol"]
