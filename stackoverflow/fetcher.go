package stackoverflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

type Query struct {
	Questions []Owner
	Askers    []TagUser
	Answerers []TagUser
}

type Owner struct {
	Tag   string
	User  string
	Count int
}

type TagUser struct {
	Tag   string
	User  string
	Score int
	Count int
}

func (e *StackOverflowExporter) Fetch(ctx context.Context) error {
	log.WithFields(log.Fields{"ref": "stack-overflow.fetch", "at": "start"}).Info()
	startTime := time.Now()

	q := &Query{}

	questionsByOwner, _, err := e.GetQuestions(ctx)
	if err != nil {
		return err
	}
	for owner, questions := range questionsByOwner {
		q.Questions = append(q.Questions, Owner{Tag: e.tag, User: owner, Count: len(questions)})
	}

	topAskers, err := e.GetTopAskers(ctx)
	if err != nil {
		return err
	}
	for _, asker := range topAskers {
		q.Askers = append(q.Askers, TagUser{Tag: e.tag, User: asker.User.Display_name, Score: asker.Score, Count: asker.Post_count})
	}
	topAnswerers, err := e.GetTopAnswerers(ctx)
	if err != nil {
		return err
	}
	for _, answerer := range topAnswerers {
		q.Answerers = append(q.Answerers, TagUser{Tag: e.tag, User: answerer.User.Display_name, Score: answerer.Score, Count: answerer.Post_count})
	}

	e.resultCache = q

	log.WithFields(log.Fields{"ref": "stack-overflow.fetch", "at": "finish", "duration": time.Since(startTime)}).Info()
	return nil

}

func (e *StackOverflowExporter) GetQuestions(ctx context.Context) (map[string][]int, map[int]int64, error) {
	result := []Question{}
	v := url.Values{}
	v.Set("page", "1")
	v.Set("order", "desc")
	v.Set("sort", "activity")
	v.Set("filter", "default")
	v.Set("tagged", e.tag)

	err := e.getRequest(ctx, "/2.2/questions", v, &result)
	if err != nil {
		return nil, nil, err
	}

	questionsByOwner := map[string][]int{}
	questionCreation := map[int]int64{}

	for _, q := range result {
		questionsByOwner[q.Owner.Display_name] = append(questionsByOwner[q.Owner.Display_name], q.Question_id)
		questionCreation[q.Question_id] = q.Creation_date
	}

	return questionsByOwner, questionCreation, nil
}

func (e *StackOverflowExporter) GetTopAskers(ctx context.Context) ([]TagScore, error) {
	result := []TagScore{}
	v := url.Values{}
	err := e.getRequest(ctx, "/2.2/tags/"+e.tag+"/top-askers/all_time", v, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e *StackOverflowExporter) GetTopAnswerers(ctx context.Context) ([]TagScore, error) {
	result := []TagScore{}
	v := url.Values{}
	err := e.getRequest(ctx, "/2.2/tags/"+e.tag+"/top-answerers/all_time", v, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e *StackOverflowExporter) getRequest(ctx context.Context, path string, v url.Values, b interface{}) error {
	//log.WithFields(log.Fields{"ref": "stackoverflow.get-request", "at": "start", "url": e.baseURL + path + "?" + v.Encode()}).Info()
	startTime := time.Now()

	v.Set("key", e.apiKey)
	req, err := http.NewRequest("GET", e.baseURL+path+"?"+v.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result := struct {
		Items          json.RawMessage `json:"items"`
		ErrorId        int             `json:"error_id"`
		ErrorName      string          `json:"error_name"`
		ErrorMessage   string          `json:"error_message"`
		Backoff        int             `json:"backoff"`
		HasMore        bool            `json:"has_more"`
		Page           int             `json:"page"`
		Page_size      int             `json:"page_size"`
		QuotaMax       int             `json:"quota_max"`
		QuotaRemaining int             `json:"quota_remaining"`
		Total          int             `json:"total"`
		Type           string          `json:"type"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{"ref": "stackoverflow.get-request", "at": "finish", "status": resp.StatusCode, "quota-max": result.QuotaMax, "quota-remaining": result.QuotaRemaining, "has-more": result.HasMore, "duration": time.Since(startTime)}).Info()

	return json.Unmarshal(result.Items, b)
}

type Question struct {
	Question_id          int
	Last_edit_date       int64
	Creation_date        int64
	Last_activity_date   int64
	Locked_date          int64
	Community_owned_date int64
	Score                int
	Answer_count         int
	Accepted_answer_id   int
	Bounty_closes_date   int64
	Bounty_amount        int
	Closed_date          int64
	Protected_date       int64
	Body                 string
	Title                string
	Tags                 []string
	Closed_reason        string
	Up_vote_count        int
	Down_vote_count      int
	Favorite_count       int
	View_count           int
	Owner                ShallowUser
	Comments             []Comment
	Answers              []Answer
	Link                 string
	Is_answered          bool
}

type ShallowUser struct {
	User_id       int
	Display_name  string
	Reputation    int
	User_type     string //one of unregistered, registered, moderator, or does_not_exist
	Profile_image string
	Link          string
}

type Comment struct {
	Comment_id    int
	Post_id       int
	Creation_date int64
	Post_type     string //one of question, or answer
	Score         int
	Edited        bool
	Body          string
	Owner         ShallowUser
	Reply_to_user ShallowUser
	Link          string
}

type Answer struct {
	Question_id          int
	Answer_id            int
	Locked_date          int64
	Creation_date        int64
	Last_edit_date       int64
	Last_activity_date   int64
	Score                int
	Community_owned_date int64
	Is_accepted          bool
	Body                 string
	Owner                ShallowUser
	Title                string
	Up_vote_count        int
	Down_vote_count      int
	Comments             []Comment
	Link                 string
}

type TagScore struct {
	User       ShallowUser
	Score      int
	Post_count int
}
