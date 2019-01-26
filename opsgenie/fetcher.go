package opsgenie

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

type Query struct {
	WhosOnCall    string
	UnAckedAlerts int
	AckedAlerts   int
	ClosedAlerts  int
}

func (e *OpsGenieExporter) Fetch(ctx context.Context) error {
	log.WithFields(log.Fields{"ref": "opsgenie.fetch", "at": "start"}).Info()
	startTime := time.Now()

	oncall, err := e.GetOncalls(ctx)
	if err != nil {
		return err
	}
	unacked, err := e.AlertCount(ctx, "status:open AND acknowledged:false")
	if err != nil {
		return err
	}
	acked, err := e.AlertCount(ctx, "status:open AND acknowledged:true")
	if err != nil {
		return err
	}
	closed, err := e.AlertCount(ctx, "status:closed")
	if err != nil {
		return err
	}

	e.resultCache = &Query{
		WhosOnCall:    oncall,
		UnAckedAlerts: unacked,
		AckedAlerts:   acked,
		ClosedAlerts:  closed,
	}

	log.WithFields(log.Fields{"ref": "opsgenie.collect", "at": "finish", "duration": time.Since(startTime)}).Info()
	return nil
}

func (e *OpsGenieExporter) GetOncalls(ctx context.Context) (string, error) {
	result := struct {
		Recipients []string `json:"onCallRecipients"`
	}{}
	err := e.getRequest(ctx, fmt.Sprintf("/schedules/%s/on-calls?scheduleIdentifierType=name&flat=true", e.schedule), &result)
	if err != nil {
		return "", err
	}
	for _, r := range result.Recipients {
		return r, nil
	}
	return "no-one", err
}

func (e *OpsGenieExporter) AlertCount(ctx context.Context, query string) (int, error) {
	result := struct {
		Count int `json:"count"`
	}{}
	v := url.Values{}
	v.Set("query", query)

	err := e.getRequest(ctx, "/alerts/count?"+v.Encode(), &result)
	if err != nil {
		return 0, err
	}
	return result.Count, nil
}

func (e *OpsGenieExporter) getRequest(ctx context.Context, path string, b interface{}) error {
	//log.WithFields(log.Fields{"ref": "opsgenie.get-request", "at": "start", "url": "https://api.opsgenie.com/v2" + path}).Info()

	req, err := http.NewRequest("GET", "https://api.opsgenie.com/v2"+path, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "GenieKey "+e.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result := struct {
		Data      json.RawMessage `json:"data"`
		Message   string          `json:"message"`
		Took      float64         `json:"took"`
		RequestID string          `json:"requestId"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{"ref": "opsgenie.get-request", "at": "finish", "status": resp.StatusCode, "message": result.Message, "took": result.Took, "request-id": result.RequestID}).Info()

	return json.Unmarshal(result.Data, b)
}
