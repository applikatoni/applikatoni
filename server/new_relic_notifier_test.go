package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/applikatoni/applikatoni/models"
)

func TestSendNewRelicRequest(t *testing.T) {
	config = &Configuration{Host: "example.com", SSLEnabled: true}

	user := &models.User{Name: "mrnugget"}
	target := &models.Target{
		Name:           "staging",
		NewRelicAppId:  "12345",
		NewRelicApiKey: "keykeykey",
	}

	application := &models.Application{
		GitHubOwner: "shipping-co",
		GitHubRepo:  "main-web-app",
	}

	deployment := &models.Deployment{
		State:           models.DEPLOYMENT_SUCCESSFUL,
		Id:              999,
		ApplicationName: "web",
		TargetName:      target.Name,
		Branch:          "master",
		CommitSha:       "f00b4r",
		Comment:         "hi",
	}

	event := &DeploymentEvent{
		State:       models.DEPLOYMENT_SUCCESSFUL,
		Deployment:  deployment,
		Application: application,
		Target:      target,
		User:        user,
	}

	expectedSummary, err := generateSummary(newRelicTemplate, event)
	if err != nil {
		t.Errorf("generating test summary failed: %s\n", err)
		return
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"deployment[app_id]", target.NewRelicAppId},
		{"deployment[description]", deployment.Comment},
		{"deployment[revision]", deployment.CommitSha},
		{"deployment[user]", user.Name},
		{"deployment[changelog]", expectedSummary},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sentApiKey := r.Header.Get("x-api-key")
		if sentApiKey != target.NewRelicApiKey {
			t.Errorf("api key wrong. expected=%s, got=%s\n", target.NewRelicApiKey, sentApiKey)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading body failed: %s\n", err)
			return
		}

		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Errorf("parsing the body failed: %s\n", err)
			return
		}

		for _, tt := range tests {
			actual := values.Get(tt.key)
			if actual != tt.expected {
				t.Errorf("key=%s, value wrong. expected=%s, got=%s", tt.key, tt.expected, actual)
				w.WriteHeader(422)
				return
			}
		}

		w.WriteHeader(201)
	}))

	defer ts.Close()

	SendNewRelicRequest(ts.URL, event)
}
