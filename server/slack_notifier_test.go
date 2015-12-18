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
		ApplicationName: "web",
		TargetName:      target.Name,
		Branch:          "master",
		CommitSha:       "f00b4r",
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

	SendSlackRequest(deployment, target, application, "test summary")
}
