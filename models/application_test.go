package models

import "testing"

func TestRepositoryURL(t *testing.T) {
	a := &Application{GitHubOwner: "owner", GitHubRepo: "repo"}
	expected := "git@github.com:owner/repo.git"

	got := a.RepositoryURL()
	if got != expected {
		t.Errorf("wrong repository URL. want=%s, got=%s", expected, got)
	}
}
