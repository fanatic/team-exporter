package main

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fanatic/team-exporter/github"
	"github.com/fanatic/team-exporter/opsgenie"
	"github.com/fanatic/team-exporter/stackoverflow"
	"github.com/fanatic/team-exporter/trello"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.WithFields(log.Fields{"ref": "main", "at": "start"}).Info()

	fetchers := []Fetcher{}

	ghExporter, err := github.New(os.Getenv("GITHUB_BASE_URL"), os.Getenv("GITHUB_TOKEN"), os.Getenv("GITHUB_ORGANIZATION"))
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(ghExporter)
	fetchers = append(fetchers, ghExporter)

	trelloExporter, err := trello.New(os.Getenv("TRELLO_APP_KEY"), os.Getenv("TRELLO_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(trelloExporter)
	fetchers = append(fetchers, trelloExporter)

	opsgenieExporter, err := opsgenie.New(os.Getenv("OPSGENIE_APIKEY"), os.Getenv("OPSGENIE_SCHEDULE"))
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(opsgenieExporter)
	fetchers = append(fetchers, opsgenieExporter)

	stackOverflowExporter, err := stackoverflow.New(os.Getenv("STACKOVERFLOW_BASE_URL"), os.Getenv("STACKOVERFLOW_KEY"), os.Getenv("STACKOVERFLOW_TAG"))
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(stackOverflowExporter)
	fetchers = append(fetchers, stackOverflowExporter)

	go PeriodicFetcher(fetchers)

	http.Handle("/metrics", prometheus.Handler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type Fetcher interface {
	Fetch(ctx context.Context) error
}

func PeriodicFetcher(fetchers []Fetcher) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	done := make(chan bool)

	for {
		select {
		case <-done:
			log.WithFields(log.Fields{"ref": "fetcher", "at": "stop"}).Info()
			return
		case <-ticker.C:
			log.WithFields(log.Fields{"ref": "fetcher", "at": "tick"}).Info()
			startTime := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel() // releases resources if slowOperation completes before timeout elapses
			concurrentFetcher(ctx, fetchers)

			log.WithFields(log.Fields{"ref": "fetcher", "at": "tock", "duration": time.Since(startTime)}).Info()
		}
	}
}

func concurrentFetcher(ctx context.Context, fetchers []Fetcher) {
	var wg sync.WaitGroup

	for _, fetcher := range fetchers {
		wg.Add(1)
		go func(ctx context.Context, f Fetcher) {
			if err := f.Fetch(ctx); err != nil {
				log.WithFields(log.Fields{"ref": "fetcher", "at": "error", "err": err}).Error("Error when fetching")
			}
			wg.Done()
		}(ctx, fetcher)
	}

	wg.Wait()
}
