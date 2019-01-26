package trello

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type TrelloExporter struct {
	Metrics     map[string]*prometheus.Desc
	appKey      string
	token       string
	resultCache []Query
}

func New(appKey, token string) (*TrelloExporter, error) {
	metrics := map[string]*prometheus.Desc{}
	metrics["CardCount"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "trello", "cards"),
		"Total number of cards",
		[]string{"board", "list", "user"}, nil,
	)

	exporter := &TrelloExporter{
		Metrics: metrics,
		appKey:  appKey,
		token:   token,
	}

	// Fetch once so any bugs are triggered on startup
	if err := exporter.Fetch(context.Background()); err != nil {
		return nil, err
	}

	return exporter, nil
}

// Describe - loops through the API metrics and passes them to prometheus.Describe
func (e *TrelloExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.Metrics {
		ch <- m
	}
}

// Collect is called when a scrape is peformed on the /metrics page
func (e *TrelloExporter) Collect(ch chan<- prometheus.Metric) {
	queries := e.resultCache
	if queries == nil {
		return
	}

	for _, q := range queries {
		ch <- prometheus.MustNewConstMetric(e.Metrics["CardCount"], prometheus.GaugeValue, float64(q.Count), q.Board, q.List, q.User)
	}
}
