package main

import (
	"flag"
	"log"
	"metric_usage_exporter/pkg/scanner"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metric prefix
const namespace = "metric_usage"

// Build Info
var (
	buildVersion  = "dev"
	buildRevision = "unknown"
)

// Adding a new metric requires adding the metrics here into this struct as well as
// - adding a new 'prometheus.NewDesc' into the NewExporter constructor
// - publishing the description to the channel in 'Describe' method
// - publishing the metric to the channel in 'Collect' method
// - adding it to exporter's testdata/ golden files
// - adding it to the README metrics table
type Exporter struct {
	scanner                     *scanner.Scanner
	buildInfoDesc               *prometheus.Desc
	upDesc                      *prometheus.Desc
	unusedMetricCardinalityDesc *prometheus.Desc
	unusedLabelCardinalityDesc  *prometheus.Desc
	dashboardCountDesc          *prometheus.Desc
	alertRuleCountDesc          *prometheus.Desc
	prometheusRuleCountDesc     *prometheus.Desc
	totalTSDBMetricsDesc        *prometheus.Desc
	totalTSDBLabelsDesc         *prometheus.Desc
	totalUsedMetricsDesc        *prometheus.Desc
	totalUsedLabelsDesc         *prometheus.Desc
	totalUnusedMetricsDesc      *prometheus.Desc
	totalUnusedLabelsDesc       *prometheus.Desc
	metricsExportLimitDesc      *prometheus.Desc
	labelsExportLimitDesc       *prometheus.Desc
}

func NewExporter(tsdbMetricsLimit int, grafanaURL string, adminUser string, adminPassword string, apiKey string, datasource string, exportLimitMetrics int, exportLimitLabels int, grafanaTimeout time.Duration) *Exporter {
	return &Exporter{
		scanner: scanner.NewScanner(
			tsdbMetricsLimit,
			grafanaURL,
			adminUser,
			adminPassword,
			apiKey,
			datasource,
			exportLimitMetrics,
			exportLimitLabels,
			grafanaTimeout,
		),
		buildInfoDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "build_info"),
			"Build information",
			[]string{"version", "revision"},
			nil,
		),
		upDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Was the last background scan successful",
			nil,
			nil,
		),
		unusedMetricCardinalityDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "unused_metric_cardinality"),
			"Series count of a top-K unused metric",
			[]string{"job", "metric"},
			nil,
		),
		unusedLabelCardinalityDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "unused_label_cardinality"),
			"Series count of a top-K unused label",
			[]string{"job", "label"},
			nil,
		),
		dashboardCountDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "dashboard_count"),
			"Number of Grafana dashboards analyzed for metric/label references",
			nil,
			nil,
		),
		alertRuleCountDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "alert_rule_count"),
			"Number of Grafana alert rules analyzed for metric/label references",
			nil,
			nil,
		),
		prometheusRuleCountDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "prometheus_rule_count"),
			"Number of Prometheus rules analyzed for metric/label references",
			nil,
			nil,
		),
		totalTSDBMetricsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "tsdb_metrics"),
			"Total number of metrics retrieved from TSDB status API (limited by tsdb-metrics-limit)",
			nil,
			nil,
		),
		totalTSDBLabelsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "tsdb_labels"),
			"Total number of labels retrieved from TSDB status API (limited by tsdb-metrics-limit)",
			nil,
			nil,
		),
		totalUsedMetricsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "used_metrics"),
			"Number of unique metrics referenced in dashboards and rules",
			nil,
			nil,
		),
		totalUsedLabelsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "used_labels"),
			"Number of unique labels referenced in dashboards and rules",
			nil,
			nil,
		),
		totalUnusedMetricsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "unused_metrics"),
			"Number of unused metrics identified in the last successful background scan (capped by export limit)",
			nil,
			nil,
		),
		totalUnusedLabelsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "unused_labels"),
			"Number of unused labels identified in the last successful background scan (capped by export limit)",
			nil,
			nil,
		),
		metricsExportLimitDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "metrics_export_limit"),
			"Configured limit on number of unused metrics to export (metrics-limit flag)",
			nil,
			nil,
		),
		labelsExportLimitDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "labels_export_limit"),
			"Configured limit on number of unused labels to export (labels-limit flag)",
			nil,
			nil,
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.buildInfoDesc
	ch <- e.upDesc
	ch <- e.unusedMetricCardinalityDesc
	ch <- e.unusedLabelCardinalityDesc
	ch <- e.dashboardCountDesc
	ch <- e.alertRuleCountDesc
	ch <- e.prometheusRuleCountDesc
	ch <- e.totalTSDBMetricsDesc
	ch <- e.totalTSDBLabelsDesc
	ch <- e.totalUsedMetricsDesc
	ch <- e.totalUsedLabelsDesc
	ch <- e.totalUnusedMetricsDesc
	ch <- e.totalUnusedLabelsDesc
	ch <- e.metricsExportLimitDesc
	ch <- e.labelsExportLimitDesc

}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		e.buildInfoDesc,
		prometheus.GaugeValue,
		1,
		buildVersion,
		buildRevision,
	)

	// Reads the latest snapshot published by the background scan.
	snap, err := e.scanner.GetSnapshot()

	// Cold start: no successful scan yet. Emit an empty result
	if snap == nil {
		ch <- prometheus.MustNewConstMetric(e.upDesc, prometheus.GaugeValue, 0)
		return
	}

	// up reflects the the actual last ATTEMPT status;
	// The metrics emitted comes from the last SUCCESS. (i.e they can be stale)
	up := 1.0
	if err != nil {
		up = 0
	}
	ch <- prometheus.MustNewConstMetric(
		e.upDesc,
		prometheus.GaugeValue,
		up,
	)

	for metric, perJobCount := range snap.UnusedMetrics {
		for job, count := range perJobCount {
			ch <- prometheus.MustNewConstMetric(
				e.unusedMetricCardinalityDesc,
				prometheus.GaugeValue,
				float64(count),
				job, metric,
			)

		}
	}

	for label, perJobCount := range snap.UnusedLabels {
		for job, count := range perJobCount {
			ch <- prometheus.MustNewConstMetric(
				e.unusedLabelCardinalityDesc,
				prometheus.GaugeValue,
				float64(count),
				job, label,
			)

		}
	}

	ch <- prometheus.MustNewConstMetric(
		e.dashboardCountDesc,
		prometheus.GaugeValue,
		float64(snap.DashboardCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.alertRuleCountDesc,
		prometheus.GaugeValue,
		float64(snap.AlertRuleCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.prometheusRuleCountDesc,
		prometheus.GaugeValue,
		float64(snap.PrometheusRuleCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.totalTSDBMetricsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalTSDBMetrics),
	)
	ch <- prometheus.MustNewConstMetric(
		e.totalTSDBLabelsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalTSDBLabels),
	)
	ch <- prometheus.MustNewConstMetric(
		e.totalUsedMetricsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalUsedMetrics),
	)
	ch <- prometheus.MustNewConstMetric(
		e.totalUsedLabelsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalUsedLabels),
	)
	ch <- prometheus.MustNewConstMetric(
		e.totalUnusedMetricsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalUnusedMetrics),
	)
	ch <- prometheus.MustNewConstMetric(
		e.totalUnusedLabelsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalUnusedLabels),
	)
	ch <- prometheus.MustNewConstMetric(
		e.metricsExportLimitDesc,
		prometheus.GaugeValue,
		float64(snap.MetricsExportLimit),
	)
	ch <- prometheus.MustNewConstMetric(
		e.labelsExportLimitDesc,
		prometheus.GaugeValue,
		float64(snap.LabelsExportLimit),
	)
}

func main() {
	log.Println("starting metric_usage_exporter...")

	var (
		addr               = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
		grafanaURL         = flag.String("grafana", "http://localhost:3000", "Grafana URL")
		datasource         = flag.String("datasource", "", "Prometheus Datasource UID to collect metrics from")
		tsdbMetricsLimit   = flag.Int("tsdb-metrics-limit", 10000, "Number of top metrics to analyze from TSDB status")
		exportLimitMetrics = flag.Int("metrics-limit", 50, "Limit the number of metrics to export")
		exportLimitLabels  = flag.Int("labels-limit", 10, "Limit the number of labels to export")
		adminUser          = flag.String("user", "", "Username for authentication")
		adminPassword      = flag.String("password", "", "Password for authentication")
		apiKey             = flag.String("api-key", "", "Grafana API key for authentication")
		scanInterval       = flag.Duration("scan-interval", 5*time.Minute, "How often to rescan Grafana and Prometheus in the background")
		grafanaTimeout     = flag.Duration("grafana-timeout", 60*time.Second, "Timeout for each HTTP request to Grafana, including Prometheus queries sent through its datasource proxy")
	)
	flag.Parse()
	if *datasource == "" {
		log.Fatal("datasourceUID for a Prometheus type datasource MUST be provided")
	}
	usingBasicAuth := *adminUser != "" || *adminPassword != ""
	usingAPIKey := *apiKey != ""

	if usingBasicAuth && usingAPIKey {
		log.Fatal("Provide either --api-key or --user/--password, not both")
	}
	if !usingBasicAuth && !usingAPIKey {
		log.Fatal("Authentication required: provide --api-key or --user and --password")
	}
	if usingBasicAuth && (*adminUser == "" || *adminPassword == "") {
		log.Fatal("Both --user and --password are required when using basic auth")
	}
	if *exportLimitMetrics <= 0 {
		log.Fatal("metrics-limit must be a positive integer")
	}
	if *exportLimitLabels <= 0 {
		log.Fatal("labels-limit must be a positive integer")
	}
	if *scanInterval <= 0 {
		log.Fatal("scan-interval must be a positive duration")
	}
	if *grafanaTimeout <= 0 {
		log.Fatal("grafana-timeout must be a positive duration")
	}

	exporter := NewExporter(*tsdbMetricsLimit, *grafanaURL, *adminUser, *adminPassword, *apiKey, *datasource, *exportLimitMetrics, *exportLimitLabels, *grafanaTimeout)

	// Test Grafana connection before starting
	exporter.scanner.TestConnection(*datasource)

	reg := prometheus.NewRegistry()

	reg.MustRegister(exporter)

	// Scans run in the background; Collect only reads the latest snapshot.
	go exporter.scanner.Run(*scanInterval)

	// Expose /metrics HTTP endpoint using the created custom registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
