package stackoverflow

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type StackOverflowExporter struct {
	Metrics     map[string]*prometheus.Desc
	apiKey      string
	tag         string
	baseURL     string
	resultCache *Query
}

func New(baseURL, apiKey, tag string) (*StackOverflowExporter, error) {
	metrics := map[string]*prometheus.Desc{}
	metrics["QuestionsTotal"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "stackoverflow", "questions_total"),
		"Total number of questions",
		[]string{"tag", "owner"}, nil,
	)
	metrics["AskerScore"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "stackoverflow", "asker_score"),
		"Total user score for questions",
		[]string{"tag", "user"}, nil,
	)
	metrics["AskerPostCount"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "stackoverflow", "asker_post_count"),
		"Total number of questions by user",
		[]string{"tag", "user"}, nil,
	)
	metrics["AnswererScore"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "stackoverflow", "answerer_score"),
		"Total user score for answers",
		[]string{"tag", "user"}, nil,
	)
	metrics["AnswererPostCount"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "stackoverflow", "answerer_post_count"),
		"Total number of answers by user",
		[]string{"tag", "user"}, nil,
	)

	exporter := &StackOverflowExporter{
		Metrics: metrics,
		baseURL: baseURL,
		apiKey:  apiKey,
		tag:     tag,
	}

	// Fetch once so any bugs are triggered on startup
	if err := exporter.Fetch(context.Background()); err != nil {
		return nil, err
	}

	return exporter, nil
}

// Describe - loops through the API metrics and passes them to prometheus.Describe
func (e *StackOverflowExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.Metrics {
		ch <- m
	}
}

// Collect is called when a scrape is peformed on the /metrics page
func (e *StackOverflowExporter) Collect(ch chan<- prometheus.Metric) {
	q := e.resultCache
	if q == nil {
		return
	}

	for _, q := range q.Questions {
		ch <- prometheus.MustNewConstMetric(e.Metrics["QuestionsTotal"], prometheus.GaugeValue, float64(q.Count), q.Tag, q.User)
	}
	for _, u := range q.Askers {
		ch <- prometheus.MustNewConstMetric(e.Metrics["AskerScore"], prometheus.GaugeValue, float64(u.Score), u.Tag, u.User)
		ch <- prometheus.MustNewConstMetric(e.Metrics["AskerPostCount"], prometheus.GaugeValue, float64(u.Count), u.Tag, u.User)
	}
	for _, u := range q.Answerers {
		ch <- prometheus.MustNewConstMetric(e.Metrics["AnswererScore"], prometheus.GaugeValue, float64(u.Score), u.Tag, u.User)
		ch <- prometheus.MustNewConstMetric(e.Metrics["AnswererPostCount"], prometheus.GaugeValue, float64(u.Count), u.Tag, u.User)
	}
}
