FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/connversed ./cmd/client-wish

FROM alpine:3.21
RUN adduser -D -u 10001 connversed \
    && mkdir -p /data/.ssh \
    && chown -R connversed /data
USER connversed
COPY --from=build /out/connversed /usr/local/bin/connversed

# Listen on every interface inside the container. The host key lives on a
# volume so it survives redeploys; without it clients get a "host key changed"
# warning after every deploy.
ENV CONNVERSED_HOST=0.0.0.0 \
    CONNVERSED_PORT=23234 \
    CONNVERSED_HOST_KEY_PATH=/data/.ssh/id_ed25519
VOLUME /data/.ssh
EXPOSE 23234

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
    CMD nc -z 127.0.0.1 23234 || exit 1

ENTRYPOINT ["connversed"]
