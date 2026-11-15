# Build stage runs on the host's platform and cross-compiles for the target,
# so multi-arch builds with buildx don't need emulation.
FROM --platform=$BUILDPLATFORM golang:1.25 AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG REVISION=unknown

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath \
    -ldflags "-s -w -X main.buildVersion=$VERSION -X main.buildRevision=$REVISION" \
    -o /out/metric_usage_exporter .

# distroless/static ships CA certificates (needed for HTTPS Grafana) and runs as
# a non-root user, with no shell or package manager.
FROM gcr.io/distroless/static-debian12:nonroot

# Links the GHCR package to the repository.
LABEL org.opencontainers.image.source=https://github.com/parthivrmenon/metric_usage_exporter

COPY --from=build /out/metric_usage_exporter /metric_usage_exporter

EXPOSE 10062
USER nonroot:nonroot
ENTRYPOINT ["/metric_usage_exporter"]
