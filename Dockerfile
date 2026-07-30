# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=development
ARG REVISION=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/invest-stock ./cmd/web

FROM alpine:3.22
ARG VERSION=development
ARG REVISION=unknown
ARG CREATED=unknown
LABEL org.opencontainers.image.title="invest-stock" \
      org.opencontainers.image.source="https://github.com/tangredtea/invest-stock" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.created="${CREATED}"

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S invest \
    && adduser -S -G invest invest \
    && mkdir -p /app/data \
    && chown -R invest:invest /app
WORKDIR /app
COPY --from=build /out/invest-stock ./invest-stock
ENV INVEST_ADDR=:8081
USER invest
VOLUME ["/app/data"]
EXPOSE 8081
ENTRYPOINT ["/app/invest-stock"]
