# Metric Usage Exporter - Helm Chart

Deploys [metric_usage_exporter](https://github.com/parthivrmenon/metric_usage_exporter), a Prometheus exporter that reports metrics and labels in a Prometheus datasource that no Grafana dashboard, Grafana alert rule or Prometheus rule uses.

## Prerequisites

- Kubernetes 1.23+
- Helm 4 (Helm 3 also works: the chart uses the `apiVersion: v2` format both read)

## Install

The chart can't default the Grafana URL, the datasource UID or the credentials, since they are specific to your Grafana. Put them in a values file:

```yaml
# my-values.yaml
grafana:
  url: http://grafana.monitoring.svc:3000
  datasource: <prometheus-datasource-uid>
  auth:
    existingSecret: grafana-credentials
```

Then install the chart from GHCR:

```bash
helm install metric-usage oci://ghcr.io/parthivrmenon/charts/metric-usage-exporter \
  --version 0.1.0 \
  --namespace monitoring --create-namespace \
  -f my-values.yaml
```

If any of the three is missing, `helm install` fails with a message naming it. See [`values.yaml`](values.yaml) for how to create the Secret and for every other option.

To install from a clone of the repository instead, use `./charts/metric-usage-exporter` in place of the `oci://` reference and drop `--version`.

### With Kustomize

Kustomize can render the chart through its `helmCharts` field. Build with `kustomize build --enable-helm`:

```yaml
# kustomization.yaml
namespace: monitoring
helmCharts:
  - name: metric-usage-exporter
    repo: oci://ghcr.io/parthivrmenon/charts
    version: 0.1.0
    releaseName: metric-usage
    valuesFile: my-values.yaml
```

Kustomize runs your installed `helm` binary. The Kustomize built into `kubectl` (v5.7 at the time of writing) fails with Helm 4 because it calls `helm version -c`, a flag Helm 4 removed. Use Helm 3, or a standalone `kustomize` release that works with Helm 4.

Alternatively, render the chart once with `helm template` and list the output under `resources:`.

## Upgrade

```bash
helm upgrade metric-usage oci://ghcr.io/parthivrmenon/charts/metric-usage-exporter \
  --version <new-version> \
  --namespace monitoring \
  -f my-values.yaml
```

Pass the same values file again: `helm upgrade` does not reuse values from the previous release unless you add `--reuse-values`. Each chart release sets the default exporter version (`appVersion`); to pin a different one, set `image.tag`.

## Uninstall

```bash
helm uninstall metric-usage --namespace monitoring
```

A Secret you created yourself for `grafana.auth.existingSecret` is not removed.

## Configuration

Every option is documented, with examples, in [`values.yaml`](values.yaml). To see them:

```bash
helm show values oci://ghcr.io/parthivrmenon/charts/metric-usage-exporter --version 0.1.0
```

### Scraping

The scrape job must use `honor_labels: true`, or the per-job breakdowns collapse into one. Set `serviceMonitor.enabled=true` if you run the Prometheus Operator; otherwise see the `serviceMonitor` section of `values.yaml` for the scrape job to add.

### Replicas

The Deployment always runs one replica: more would repeat every scan against Grafana and export duplicate series.
