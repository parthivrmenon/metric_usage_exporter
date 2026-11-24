# Releasing the Helm chart

This is the checklist for publishing a chart version. The chart is released
separately from the exporter and is
pushed as an OCI artifact to GHCR, next to the image:

```
oci://ghcr.io/parthivrmenon/charts/metric-usage-exporter
```

## 1. Chart Versioning

`Chart.yaml` carries two versions that move independently:

- **`version`**: the chart's own version. Bump it on **every** change to anything in
  this folder, following SemVer
- **`appVersion`**: the exporter image tag the chart deploys by default. Change it
  only to a version whose image is already on GHCR, and bump `version` along with it.

When the exporter is released, release the image first, then bump `appVersion` here.


Set the chart version once. Every step below uses it:

```bash
CHART_VERSION=0.1.0
CHART=charts/metric-usage-exporter
```

## 2. Release from `main`

Release from merged code, not from a feature branch:

```bash
git checkout main && git pull
git status            # must be clean
grep -E '^(version|appVersion):' $CHART/Chart.yaml
# expect version: $CHART_VERSION, and appVersion pointing at a published image
```

## 3. Checks

```bash
V=(--set grafana.url=http://g --set grafana.datasource=abc --set grafana.auth.apiKey=x)

helm lint --strict $CHART "${V[@]}"
helm template t $CHART "${V[@]}" --set serviceMonitor.enabled=true > /dev/null
helm template t $CHART 2>&1 | grep 'grafana.auth.apiKey'   # must fail with this message
```

Then install it on a real cluster against a real Grafana (for example kind with
kube-prometheus-stack) and confirm the pod is Running, `metric_usage_up` reaches 1 and
Prometheus scrapes the target with the original `job` labels intact.

## 4. Package

```bash
helm package $CHART -d dist/
tar tzf dist/metric-usage-exporter-$CHART_VERSION.tgz
```

Check the file list: it should hold `Chart.yaml`, `values.yaml`, `README.md`,
`templates/` and `.helmignore`, and nothing else. This checklist and editor files are
excluded by `.helmignore`.

## 5. Push to GHCR

Log in with a classic PAT that has the `write:packages` scope (the same token as for
the image):

```bash
echo "$GHCR_TOKEN" | helm registry login ghcr.io -u parthivrmenon --password-stdin
helm push dist/metric-usage-exporter-$CHART_VERSION.tgz oci://ghcr.io/parthivrmenon/charts
```

**First push only:** GHCR creates the package as private. Under GitHub → Packages →
charts/metric-usage-exporter → Package settings:

- Change visibility to public, or nobody can pull it.
- Connect it to the `metric_usage_exporter` repository so it shows up on the repo page.
  Unlike the image, the chart carries no `org.opencontainers.image.source` label to do
  this automatically.

## 6. Tag

Tag the commit the chart was packaged from. The `metric-usage-exporter-` prefix keeps
chart tags apart from the exporter's `v*` tags:

```bash
git tag -a metric-usage-exporter-$CHART_VERSION -m "Helm chart $CHART_VERSION"
git push origin metric-usage-exporter-$CHART_VERSION
```

## 7. Post-release

Pull while logged out, which proves the package is public:

```bash
helm registry logout ghcr.io
helm show chart oci://ghcr.io/parthivrmenon/charts/metric-usage-exporter --version $CHART_VERSION
# expect the version and appVersion from step 1
rm -rf dist/
```
