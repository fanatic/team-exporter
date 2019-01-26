package opsgenie

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type OpsGenieExporter struct {
	Metrics     map[string]*prometheus.Desc
	apiKey      string
	schedule    string
	resultCache *Query
}

func New(apiKey, schedule string) (*OpsGenieExporter, error) {
	metrics := map[string]*prometheus.Desc{}
	metrics["WhosOnCall"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "opsgenie", "oncall"),
		"Who is oncall",
		[]string{"schedule", "user"}, nil,
	)
	metrics["UnAckedAlerts"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "opsgenie", "unacked_alerts"),
		"Total number of unacked alerts",
		[]string{"schedule"}, nil,
	)
	metrics["AckedAlerts"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "opsgenie", "acked_alerts"),
		"Total number of acked alerts",
		[]string{"schedule"}, nil,
	)
	metrics["ClosedAlerts"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "opsgenie", "closed_alerts"),
		"Total number of closed alerts",
		[]string{"schedule"}, nil,
	)

	exporter := &OpsGenieExporter{
		Metrics:  metrics,
		apiKey:   apiKey,
		schedule: schedule,
	}

	// Fetch once so any bugs are triggered on startup
	if err := exporter.Fetch(context.Background()); err != nil {
		return nil, err
	}

	return exporter, nil
}

// Describe - loops through the API metrics and passes them to prometheus.Describe
func (e *OpsGenieExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.Metrics {
		ch <- m
	}
}

// Collect is called when a scrape is peformed on the /metrics page
func (e *OpsGenieExporter) Collect(ch chan<- prometheus.Metric) {
	q := e.resultCache
	if q == nil {
		return
	}

	ch <- prometheus.MustNewConstMetric(e.Metrics["WhosOnCall"], prometheus.GaugeValue, float64(1), e.schedule, q.WhosOnCall)
	ch <- prometheus.MustNewConstMetric(e.Metrics["UnAckedAlerts"], prometheus.GaugeValue, float64(q.UnAckedAlerts), e.schedule)
	ch <- prometheus.MustNewConstMetric(e.Metrics["AckedAlerts"], prometheus.GaugeValue, float64(q.AckedAlerts), e.schedule)
	ch <- prometheus.MustNewConstMetric(e.Metrics["ClosedAlerts"], prometheus.GaugeValue, float64(q.ClosedAlerts), e.schedule)
}
