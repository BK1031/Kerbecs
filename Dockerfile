# syntax=docker/dockerfile:1.7
FROM --platform=$BUILDPLATFORM golang:1.25-alpine3.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /kerbecs

FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.source="https://github.com/BK1031/Kerbecs" \
      org.opencontainers.image.description="Kerbecs API Gateway" \
      org.opencontainers.image.licenses="MIT"

COPY --from=builder /kerbecs /kerbecs

ENV TZ=UTC
EXPOSE 10310 10300

ENTRYPOINT ["/kerbecs"]
