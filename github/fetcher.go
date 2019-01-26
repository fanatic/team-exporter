package github

import (
	"context"
	"time"

	"github.com/shurcooL/githubv4"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func (m *GitHubExporter) Fetch(ctx context.Context) error {
	log.WithFields(log.Fields{"ref": "github.fetch", "at": "start"}).Info()
	startTime := time.Now()

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: m.Token},
	)
	httpClient := oauth2.NewClient(ctx, src)

	var client *githubv4.Client
	if m.baseURL == "" {
		client = githubv4.NewClient(httpClient)

	} else {
		client = githubv4.NewEnterpriseClient(m.baseURL, httpClient)
	}

	var q Query
	v := map[string]interface{}{
		"organizationName": githubv4.String(m.OrganizationName),
	}
	err := client.Query(ctx, &q, v)
	if err != nil {
		return err
	}

	m.resultCache = &q

	log.WithFields(log.Fields{"ref": "github.fetch", "at": "finish", "duration": time.Since(startTime)}).Info()
	return nil
}

type Query struct {
	RateLimit struct {
		Limit     int
		Cost      int
		Remaining int
		ResetAt   time.Time
	}
	Organization struct {
		MembersWithRole struct {
			Nodes []struct {
				Login          string
				CommitComments struct {
					TotalCount int
				}
				Issues struct {
					TotalCount int
				}
				IssueComments struct {
					TotalCount int
				}
				PullRequests struct {
					TotalCount int
				}

				ContributionsCollection struct {
					TotalCommitContributions            int
					TotalIssueContributions             int
					TotalPullRequestContributions       int
					TotalPullRequestReviewContributions int
				}
			}
		} `graphql:"membersWithRole(first: 100)"`
		Repositories struct {
			Nodes []struct {
				NameWithOwner string
				OpenIssues    struct {
					TotalCount int
				} `graphql:"openIssues: issues(states:OPEN)"`
				ClosedIssues struct {
					TotalCount int
				} `graphql:"issues(states:CLOSED)"`
				OpenPullRequests struct {
					TotalCount int
				} `graphql:"openPullRequests: pullRequests(states: OPEN)"`
				ClosedPullRequests struct {
					TotalCount int
				} `graphql:"pullRequests(states: [CLOSED, MERGED])"`
				DefaultBranchRef struct {
					Target struct {
						Commit struct {
							History struct {
								TotalCount int
							}
						} `graphql:"... on Commit"`
					}
				}
			}
		} `graphql:"repositories(first: 100)"`
	} `graphql:"organization(login: $organizationName)"`
}
