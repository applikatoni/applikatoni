package models

import "fmt"

type Application struct {
	Name                 string    `json:"name"`
	Targets              []*Target `json:"targets"`
	ReadUsernames        []string  `json:"read_usernames"`
	GitHubOwner          string    `json:"github_owner"`
	GitHubRepo           string    `json:"github_repo"`
	GitHubBranches       []string  `json:"github_branches"`
	TravisImageURL       string    `json:"travis_image_url"`
	DailyDigestReceivers []string  `json:"daily_digest_receivers"`
	DailyDigestTarget    string    `json:"daily_digest_target"`
}

func (a *Application) IsReader(userName string) bool {
	return isInList(userName, a.ReadUsernames)
}

func (a *Application) RepositoryURL() string {
	return fmt.Sprintf("git@github.com:%s/%s.git", a.GitHubOwner, a.GitHubRepo)
}

// TODO: we can do this in O(1) if we use a map instead of slice for usernames
func isInList(username string, list []string) bool {
	for _, item := range list {
		if username == item {
			return true
		}
	}
	return false
}
