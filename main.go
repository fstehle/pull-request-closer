package main // import "github.com/Jimdo/pull-request-closer"
import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
)

type PullRequestCloser struct {
	client *github.Client
}

func NewPullRequestCloser(githubAccessToken string) *PullRequestCloser {
	p := &PullRequestCloser{}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubAccessToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	p.client = github.NewClient(tc)

	return p
}

func (p *PullRequestCloser) findPullRequests(owner string, repo string, days int) ([]github.PullRequest, error) {
	pullRequests, _, err := p.client.PullRequests.List(owner, repo, nil)
	if err != nil {
		return nil, err
	}

	var oldPullRequests []github.PullRequest
	for _, pullRequest := range pullRequests {
		if (*pullRequest.UpdatedAt).Before(time.Now().AddDate(0, 0, -days)) {
			oldPullRequests = append(oldPullRequests, pullRequest)
		}
	}

	return oldPullRequests, nil
}

func (p *PullRequestCloser) closePullRequest(owner string, repo string, pull github.PullRequest, comment string, label string) error {

	issueComment := &github.IssueComment{Body: github.String(comment)}
	_, _, err := p.client.Issues.CreateComment(owner, repo, *pull.Number, issueComment)
	if err != nil {
		return err
	}

	if label != "" {
		_, _, err = p.client.Issues.AddLabelsToIssue(owner, repo, *pull.Number, []string{label})
		if err != nil {
			return err
		}
	}

	pull.State = github.String("closed")
	_, _, err = p.client.PullRequests.Edit(owner, repo, *pull.Number, &pull)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var (
		accessToken = kingpin.Flag("access-token", "GitHub access token").Required().PlaceHolder("TOKEN").String()
		owner       = kingpin.Flag("owner", "GitHub repository owner whose pull requests will be operated upon").Required().PlaceHolder("OWNER").String()
		repository  = kingpin.Flag("repository", "GitHub repository name whose pull requests will be operated upon").Required().PlaceHolder("REPO").String()
		label       = kingpin.Flag("label", "Name of a label used on the pull request to indicate that it has automatically been closed").PlaceHolder("LABEL").String()
		comment     = kingpin.Flag("comment", "Content of comment, which will be created when pull request is closed").PlaceHolder("TEXT").Default("Pull request was automatically closed").String()
		days        = kingpin.Flag("days", "Integer number of days. If at least this number of days elapses after a pull request has been created without any new comment or commits being posted it will be closed and an explanatory comment will be posted").Required().PlaceHolder("LABEL").Int()
	)

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate)
	kingpin.CommandLine.Help = "Tool to auto-close old GitHub pull requests that were forgotten by their committer"
	kingpin.Parse()

	p := NewPullRequestCloser(*accessToken)

	pullRequests, err := p.findPullRequests(*owner, *repository, *days)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("Failed to find pull requests")
	}
	for _, pullRequest := range pullRequests {
		log.WithFields(log.Fields{"pull_request": *pullRequest.HTMLURL}).Info("Closing pull request")
		err = p.closePullRequest(*owner, *repository, pullRequest, *comment, *label)
		if err != nil {
			log.WithFields(log.Fields{"error": err, "pull_request": *pullRequest.HTMLURL}).Warn("Failed to close pull request")
		}
	}
}
