package trello

import (
	"context"
	"time"

	"github.com/adlio/trello"
	log "github.com/sirupsen/logrus"
)

func (e *TrelloExporter) Fetch(ctx context.Context) error {
	log.WithFields(log.Fields{"ref": "trello.fetch", "at": "start"}).Info()
	startTime := time.Now()

	client := trello.NewClient(e.appKey, e.token)
	me, err := client.GetMember("me", trello.Defaults())
	if err != nil {
		return err
	}

	boards, err := me.GetBoards(map[string]string{"filter": "open", "lists": "open", "list_fields": "id,name"})
	if err != nil {
		return err
	}

	q := []Query{}
	for _, board := range boards {
		//log.WithFields(log.Fields{"ref": "trello.fetch", "at": "start", "board": board.Name}).Info()
		memberListCardCount := map[string]map[string]int{}
		listNames := map[string]string{}
		cards, err := board.GetCards(map[string]string{"fields": "idList", "members": "true", "member_fields": "username", "filter": "visible"})
		if err != nil {
			return err
		}

		// Aggregate cards per list per member
		for _, list := range board.Lists {
			listNames[list.ID] = list.Name
		}
		for _, card := range cards {
			listName := listNames[card.IDList]
			for _, member := range card.Members {
				if memberListCardCount[member.Username] == nil {
					memberListCardCount[member.Username] = map[string]int{}
				}
				memberListCardCount[member.Username][listName]++
			}
			// Remember to count cards unassociated, but not twice so we can still aggregate
			if len(card.Members) == 0 {
				if memberListCardCount["none"] == nil {
					memberListCardCount["none"] = map[string]int{}
				}
				memberListCardCount["none"][listName]++
			}
		}

		// Transform counts for prometheus
		for username, lists := range memberListCardCount {
			for name, count := range lists {
				q = append(q, Query{
					Board: board.Name,
					List:  name,
					User:  username,
					Count: count,
				})
			}
		}
	}

	e.resultCache = q
	log.WithFields(log.Fields{"ref": "trello.fetch", "at": "finish", "duration": time.Since(startTime)}).Info()
	return nil
}

type Query struct {
	Board string
	List  string
	User  string
	Count int
}
