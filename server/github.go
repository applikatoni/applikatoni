package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/applikatoni/applikatoni/models"
	"golang.org/x/oauth2"
)

const gitHubAPI = "https://api.github.com"

type GitHubCommit struct {
	Author  *models.User `json:"author"`
	Sha     string       `json:"sha"`
	HtmlURL string       `json:"html_url"`
	Commit  struct {
		Message   string `json:"message"`
		Committer struct {
			ComittedAt time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

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
	TravisImageURL  string `json:"travis_image_url"`
	TravisImageLink string `json:"travis_image_link"`
}

type GitHubBranch struct {
	Name            string       `json:"name"`
	CurrentCommit   GitHubCommit `json:"commit"`
	TravisImageURL  string       `json:"travis_image_url"`
	TravisImageLink string       `json:"travis_image_link"`
}

type GitHubDiff struct {
	GitHubCompareURL string         `json:"html_url"`
	Status           string         `json:"status"`
	AheadBy          int            `json:"ahead_by"`
	BehindBy         int            `json:"behind_by"`
	Commits          []GitHubCommit `json:"commits"`
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
		link, err := buildTravisLink(a.TravisImageURL)
		if err != nil {
			return nil, err
		}

		for i, _ := range pulls {
			imageURL, err := addBranchTravisURL(a.TravisImageURL, pulls[i].Head.Branch)
			if err != nil {
				return nil, err
			}
			pulls[i].TravisImageURL = imageURL
			pulls[i].TravisImageLink = link
		}
	}

	return pulls, nil
}

func (gc *GitHubClient) GetBranches(a *models.Application) ([]GitHubBranch, error) {
	branches := []GitHubBranch{}

	var travisLink string
	if a.TravisImageURL != "" {
		var err error
		travisLink, err = buildTravisLink(a.TravisImageURL)
		if err != nil {
			return nil, err
		}
	}

	for _, branchName := range a.GitHubBranches {
		branch := GitHubBranch{}
		url := fmt.Sprintf("%s/repos/%s/%s/branches/%s", gitHubAPI, a.GitHubOwner, a.GitHubRepo, branchName)

		err := gc.GetDecode(url, &branch)
		if err != nil {
			return nil, err
		}

		if a.TravisImageURL != "" {
			imageURL, err := addBranchTravisURL(a.TravisImageURL, branchName)
			if err != nil {
				return nil, err
			}
			branch.TravisImageURL = imageURL
			branch.TravisImageLink = travisLink
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

func (gc *GitHubClient) Compare(a *models.Application, oldSha, newSha string) (*GitHubDiff, error) {
	diff := &GitHubDiff{}
	url := fmt.Sprintf("%s/repos/%s/%s/compare/%s...%s",
		gitHubAPI, a.GitHubOwner, a.GitHubRepo, oldSha, newSha)

	err := gc.GetDecode(url, diff)
	if err != nil {
		return nil, err
	}

	return diff, nil
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

func addBranchTravisURL(travisURL string, branch string) (string, error) {
	u, err := url.Parse(travisURL)
	if err != nil {
		msg := fmt.Sprintf("failed to parse travis_image_url: %q", travisURL)
		return "", errors.New(msg)
	}

	q := u.Query()
	q.Set("branch", branch)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func buildTravisLink(travisImageURL string) (string, error) {
	u, err := url.Parse(travisImageURL)
	if err != nil {
		msg := fmt.Sprintf("failed to parse travis_image_url: %q", travisImageURL)
		return "", errors.New(msg)
	}

	u.RawQuery = ""
	u.Path = strings.Replace(u.Path, ".svg", "", 1)

	return u.String(), nil
}
