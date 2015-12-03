package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/applikatoni/applikatoni/models"
)

func TestNotifyBugsnag(t *testing.T) {
	target := &models.Target{Name: "staging", BugsnagApiKey: "APIKEY"}

	application := &models.Application{
		GitHubOwner: "shipping-co",
		GitHubRepo:  "main-web-app",
	}

	deployment := &models.Deployment{
		TargetName: target.Name,
		Branch:     "master",
		CommitSha:  "f00b4r",
	}

	tests := []struct {
		formKey  string
		expected string
	}{
		{"apiKey", target.BugsnagApiKey},
		{"releaseStage", target.Name},
		{"repository", application.RepositoryURL()},
		{"branch", deployment.Branch},
		{"revision", deployment.CommitSha},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := 200
		for _, tt := range tests {
			actual := r.FormValue(tt.formKey)
			if actual != tt.expected {
				t.Errorf("sent wrong value for %s. want=%s, got=%s", tt.formKey, tt.expected, actual)
				response = 422
			}
		}
		w.WriteHeader(response)
	}))
	defer ts.Close()

	SendBugsnagRequest(ts.URL, deployment, target, application)
}
