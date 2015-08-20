package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/flinc/applikatoni/models"
	"golang.org/x/oauth2"
)

const gitHubAPI = "https://api.github.com"

type GitHubPullRequest struct {
	Id        int64        `json:"id"`
	Url       string       `json:"html_url"`
	Title     string       `json:"title"`
	User      *models.User `json:"user"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Head      struct {
		Branch    string `json:"ref"`
		CommitSha string `json:"sha"`
	} `json:"head"`
	TravisImageURL string `json:"travis_image_url"`
}

type GitHubBranch struct {
	Name          string `json:"name"`
	CurrentCommit struct {
		Author *models.User `json:"author"`
		Sha    string       `json:"sha"`
		Commit struct {
			Message   string `json:"message"`
			Committer struct {
				ComittedAt time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	} `json:"commit"`
	TravisImageURL string `json:"travis_image_url"`
}

type GitHubClient struct{ *http.Client }

func NewGitHubClient(u *models.User) *GitHubClient {
	token := &oauth2.Token{AccessToken: u.AccessToken}
	client := oauthCfg.Client(oauth2.NoContext, token)

	return &GitHubClient{client}
}

func (gc *GitHubClient) GetPullRequests(a *models.Application) ([]GitHubPullRequest, error) {
	pulls := []GitHubPullRequest{}

	url := fmt.Sprintf("%s/repos/%s/%s/pulls?state=open", gitHubAPI, a.GitHubOwner, a.GitHubRepo)
	err := gc.GetDecode(url, &pulls)
	if err != nil {
		return nil, err
	}

	if a.TravisImageURL != "" {
		for i, _ := range pulls {
			pulls[i].TravisImageURL = fmt.Sprintf("%s&branch=%s", a.TravisImageURL, pulls[i].Head.Branch)
		}
	}

	return pulls, nil
}

func (gc *GitHubClient) GetBranches(a *models.Application) ([]GitHubBranch, error) {
	branches := []GitHubBranch{}

	for _, branchName := range a.GitHubBranches {
		branch := GitHubBranch{}
		url := fmt.Sprintf("%s/repos/%s/%s/branches/%s", gitHubAPI, a.GitHubOwner, a.GitHubRepo, branchName)

		err := gc.GetDecode(url, &branch)
		if err != nil {
			return nil, err
		}

		if a.TravisImageURL != "" {
			branch.TravisImageURL = fmt.Sprintf("%s&branch=%s", a.TravisImageURL, branchName)
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

func (gc *GitHubClient) UpdateUser(u *models.User) error {
	url := fmt.Sprintf("%s/user", gitHubAPI)

	err := gc.GetDecode(url, u)
	return err
}

func (gc *GitHubClient) GetDecode(url string, v interface{}) error {
	res, err := gc.Get(url)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		msg := fmt.Sprintf("GitHub responded with %d instead of 200", res.StatusCode)
		return errors.New(msg)
	}

	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(v)
	if err != nil {
		return err
	}

	return nil
}
