# Releasing metric_usage_exporter

This is the checklist for cutting a release. A release has three outputs, and all of
them are built from the same tagged commit:

- three binaries attached to a GitHub Release
- a multi-arch Docker image on GHCR
- a Grafana dashboard revision on grafana.com, if the dashboard changed

Set the version once at the start. Every step below uses it:

```bash
VERSION=v0.2.0
```

A pushed tag and its release are effectively permanent once anyone has downloaded
them, so get the checks right before tagging.

## 1. Checks

CI runs these on every PR and every push to `main`. Before tagging, confirm the
latest `main` run is green, and run them locally as well:

```bash
gofmt -l .                     # must print nothing
go vet ./...
go test -count=1 -race ./...
```

`-race` matters: the background scan writes the snapshot while `Collect` reads it.

## 2. Release from `main`

Release from merged code, not from a feature branch:

```bash
git checkout main && git pull
git status            # must be clean
```

## 3. Tag

```bash
git tag -a $VERSION -m "Release $VERSION"
git push origin $VERSION
```

## 4. Cross-compile the binaries

Tag first, so the revision embedded in the binaries is the tagged commit:

```bash
REV=$(git rev-parse --short HEAD)
LDFLAGS="-X main.buildVersion=$VERSION -X main.buildRevision=$REV"

# Linux amd64 (primary server target)
GOOS=linux  GOARCH=amd64 go build -ldflags "$LDFLAGS" -o metric_usage_exporter-linux-amd64 .

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o metric_usage_exporter-darwin-arm64 .

# macOS amd64 (Intel)
GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o metric_usage_exporter-darwin-amd64 .
```

## 5. Verify the build metadata

The README tells users to read the version from `metric_usage_build_info`, so check it
before publishing:

```bash
./metric_usage_exporter-darwin-arm64 --help | head -3
# start it against the local stack (see CONTRIBUTING.md), then:
curl -s localhost:10062/metrics | grep metric_usage_build_info
# expect version="$VERSION" revision="$REV", not version="dev"
```

## 6. Create the GitHub Release

Create the release from the tag, attach the three binaries and write the release
notes. Keep the asset names from step 4: the README's Installation section links to
them.

## 7. Build and push the Docker image

Build from the same tag, so the image and the binaries report the same
`metric_usage_build_info`. This reuses `VERSION` and `REV` from step 4. The `Dockerfile`
cross-compiles in its build stage, so `linux/amd64` and `linux/arm64` both build in
one command without QEMU emulation.

```bash
# One-time: create a builder that can do multi-platform builds
docker buildx create --name metric-usage --use

# Log in to GHCR with a classic PAT that has the write:packages scope
echo "$GHCR_TOKEN" | docker login ghcr.io -u parthivrmenon --password-stdin

docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=$VERSION \
  --build-arg REVISION=$REV \
  -t ghcr.io/parthivrmenon/metric_usage_exporter:$VERSION \
  -t ghcr.io/parthivrmenon/metric_usage_exporter:latest \
  --push .
```

To test the image before pushing, build it for your own platform only and load it
into the local daemon. `--load` can't take a multi-platform build.

```bash
docker build --build-arg VERSION=$VERSION --build-arg REVISION=$REV \
  -t metric_usage_exporter:$VERSION .
docker run --rm -p 10062:10062 metric_usage_exporter:$VERSION \
  -grafana http://host.docker.internal:3000 -user admin -password admin \
  -datasource PBFA97CFB590B2093
curl -s localhost:10062/metrics | grep metric_usage_build_info
```

You have to set `-grafana` explicitly, because the default `http://localhost:3000`
points at the container itself. `host.docker.internal` reaches the host on Docker
Desktop.

**First push only:** GHCR creates the package as private. Make it public under
GitHub → Packages → metric_usage_exporter → Package settings → Change visibility, and
link it to the repository so it shows up on the repo page.

## 8. Dashboard

Skip this step if the dashboard didn't change in this release.

- `docker/grafana/provisioning/dashboards/metric_usage.json` is the canonical dashboard.
  The grafana.com upload is a copy of it with the `uid` removed, so each import
  gets its own `uid`. The maintainer keeps that copy, and its listing README, in
  `grafana_com_assets/`, which isn't tracked in git.
- Upload the updated JSON as a new revision on grafana.com. Refresh the listing
  README if the panels or requirements changed.
- Re-capture `docs/images/dashboard.png` if the layout changed.

## 9. Post-release

Confirm that the README's download and Docker instructions actually work:

```bash
curl -sIL https://github.com/parthivrmenon/metric_usage_exporter/releases/latest/download/metric_usage_exporter-linux-amd64 | tail -2

# Pull while logged out, which proves the package is public
docker logout ghcr.io
docker pull ghcr.io/parthivrmenon/metric_usage_exporter:$VERSION
docker buildx imagetools inspect ghcr.io/parthivrmenon/metric_usage_exporter:$VERSION   # expect amd64 + arm64
```
