package github

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type GitHubExporter struct {
	Metrics          map[string]*prometheus.Desc
	Token            string
	OrganizationName string
	baseURL          string
	resultCache      *Query
}

func New(baseURL, token, organizationName string) (*GitHubExporter, error) {
	metrics := map[string]*prometheus.Desc{}
	metrics["UserCommitComments"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "user_commit_comments"),
		"Total number of user commit comments",
		[]string{"user"}, nil,
	)
	metrics["UserIssues"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "user_issues"),
		"Total number of user issues",
		[]string{"user"}, nil,
	)
	metrics["UserIssueComments"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "user_issue_comments"),
		"Total number of user issue comments",
		[]string{"user"}, nil,
	)
	metrics["UserPullRequests"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "user_pull_requests"),
		"Total number of user pull requests",
		[]string{"user"}, nil,
	)
	metrics["UserCommitContributions"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "user_commit_contributions"),
		"Total number of user commit contributions",
		[]string{"user"}, nil,
	)
	metrics["UserIssueContributions"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "user_issue_contributions"),
		"Total number of user issue contributions",
		[]string{"user"}, nil,
	)
	metrics["UserPullRequestContributions"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "user_pull_request_contributions"),
		"Total number of user pull request contributions",
		[]string{"user"}, nil,
	)
	metrics["UserPullRequestReviewContributions"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "user_pull_request_review_contributions"),
		"Total number of user pull request review contributions",
		[]string{"user"}, nil,
	)
	metrics["RepoOpenIssues"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "repo_open_issues"),
		"Total number of repo open issues",
		[]string{"repo"}, nil,
	)
	metrics["RepoClosedIssues"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "repo_closed_issues"),
		"Total number of repo closed issues",
		[]string{"repo"}, nil,
	)
	metrics["RepoOpenPullRequests"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "repo_open_pull_requests"),
		"Total number of repo open pull requests",
		[]string{"repo"}, nil,
	)
	metrics["RepoClosedPullRequests"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "repo_closed_pull_requests"),
		"Total number of repo closed pull requests",
		[]string{"repo"}, nil,
	)
	metrics["RepoCommits"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "repo_commits"),
		"Total number of repo commits",
		[]string{"repo"}, nil,
	)
	metrics["Limit"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "rate_limit"),
		"Number of API queries allowed in a 60 minute window",
		[]string{}, nil,
	)
	metrics["Remaining"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "rate_remaining"),
		"Number of API queries remaining in the current window",
		[]string{}, nil,
	)
	metrics["Cost"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "rate_cost"),
		"Cost of GitHub API query",
		[]string{}, nil,
	)
	metrics["Reset"] = prometheus.NewDesc(
		prometheus.BuildFQName("team", "github", "rate_reset"),
		"The time at which the current rate limit window resets in UTC epoch seconds",
		[]string{}, nil,
	)

	exporter := &GitHubExporter{
		Metrics:          metrics,
		Token:            token,
		baseURL:          baseURL,
		OrganizationName: organizationName,
	}

	// Fetch once so any bugs are triggered on startup
	if err := exporter.Fetch(context.Background()); err != nil {
		return nil, err
	}

	return exporter, nil
}

// Describe - loops through the API metrics and passes them to prometheus.Describe
func (e *GitHubExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.Metrics {
		ch <- m
	}
}

// Collect is called when a scrape is peformed on the /metrics page
func (e *GitHubExporter) Collect(ch chan<- prometheus.Metric) {
	q := e.resultCache
	if q == nil {
		return
	}

	// Rate Limits
	ch <- prometheus.MustNewConstMetric(e.Metrics["Limit"], prometheus.GaugeValue, float64(q.RateLimit.Limit))
	ch <- prometheus.MustNewConstMetric(e.Metrics["Remaining"], prometheus.GaugeValue, float64(q.RateLimit.Remaining))
	ch <- prometheus.MustNewConstMetric(e.Metrics["Cost"], prometheus.GaugeValue, float64(q.RateLimit.Cost))
	ch <- prometheus.MustNewConstMetric(e.Metrics["Reset"], prometheus.GaugeValue, float64(q.RateLimit.ResetAt.Unix()))

	// User Stats
	for _, member := range q.Organization.MembersWithRole.Nodes {
		ch <- prometheus.MustNewConstMetric(e.Metrics["UserCommitComments"], prometheus.GaugeValue, float64(member.CommitComments.TotalCount), member.Login)
		ch <- prometheus.MustNewConstMetric(e.Metrics["UserIssues"], prometheus.GaugeValue, float64(member.Issues.TotalCount), member.Login)
		ch <- prometheus.MustNewConstMetric(e.Metrics["UserIssueComments"], prometheus.GaugeValue, float64(member.IssueComments.TotalCount), member.Login)
		ch <- prometheus.MustNewConstMetric(e.Metrics["UserPullRequests"], prometheus.GaugeValue, float64(member.PullRequests.TotalCount), member.Login)

		ch <- prometheus.MustNewConstMetric(e.Metrics["UserCommitContributions"], prometheus.GaugeValue, float64(member.ContributionsCollection.TotalCommitContributions), member.Login)
		ch <- prometheus.MustNewConstMetric(e.Metrics["UserIssueContributions"], prometheus.GaugeValue, float64(member.ContributionsCollection.TotalIssueContributions), member.Login)
		ch <- prometheus.MustNewConstMetric(e.Metrics["UserPullRequestContributions"], prometheus.GaugeValue, float64(member.ContributionsCollection.TotalPullRequestContributions), member.Login)
		ch <- prometheus.MustNewConstMetric(e.Metrics["UserPullRequestReviewContributions"], prometheus.GaugeValue, float64(member.ContributionsCollection.TotalPullRequestReviewContributions), member.Login)
	}

	// Repository Stats
	for _, repository := range q.Organization.Repositories.Nodes {
		ch <- prometheus.MustNewConstMetric(e.Metrics["RepoOpenIssues"], prometheus.GaugeValue, float64(repository.OpenIssues.TotalCount), repository.NameWithOwner)
		ch <- prometheus.MustNewConstMetric(e.Metrics["RepoClosedIssues"], prometheus.GaugeValue, float64(repository.ClosedIssues.TotalCount), repository.NameWithOwner)
		ch <- prometheus.MustNewConstMetric(e.Metrics["RepoOpenPullRequests"], prometheus.GaugeValue, float64(repository.OpenPullRequests.TotalCount), repository.NameWithOwner)
		ch <- prometheus.MustNewConstMetric(e.Metrics["RepoClosedPullRequests"], prometheus.GaugeValue, float64(repository.ClosedPullRequests.TotalCount), repository.NameWithOwner)
		ch <- prometheus.MustNewConstMetric(e.Metrics["RepoCommits"], prometheus.GaugeValue, float64(repository.DefaultBranchRef.Target.Commit.History.TotalCount), repository.NameWithOwner)
	}
}
