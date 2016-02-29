package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/applikatoni/applikatoni/models"
)

func TestNotifySlack(t *testing.T) {
	target := &models.Target{Name: "staging"}

	application := &models.Application{
		GitHubOwner: "shipping-co",
		GitHubRepo:  "main-web-app",
	}

	deployment := &models.Deployment{
		State:           models.DEPLOYMENT_SUCCESSFUL,
		ApplicationName: "web",
		TargetName:      target.Name,
		Branch:          "master",
		CommitSha:       "f00b4r",
	}
	user := &models.User{
		Name: "Foo Bar",
	}

	event := &DeploymentEvent{
		Deployment:  deployment,
		Application: application,
		Target:      target,
		User:        user,
	}

	expectedMessage := slackMsg{
		Text: "test summary",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := 200
		body, _ := ioutil.ReadAll(r.Body)
		expectedJson, _ := json.Marshal(expectedMessage)
		if string(body) != string(expectedJson) {
			t.Errorf("sent wrong payload expected=%v got=%v", expectedMessage, string(body))
			response = 422
		}
		w.WriteHeader(response)
	}))
	defer ts.Close()

	target.SlackUrl = ts.URL

	SendSlackRequest(event, "test summary")
}
