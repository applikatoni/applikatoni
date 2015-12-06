package main

import (
	"encoding/json"
	"github.com/applikatoni/applikatoni/models"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
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

func TestGenerateSlackSummary(t *testing.T) {
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
		Comment:         "hi",
	}

	user := &models.User{
		Name: "Foo Bar",
	}

	config = &Configuration{
		Host:       "example.com",
		SSLEnabled: true,
	}

	expectedSuccessMsg := `main-web-app Successfully Deployed:
Foo Bar deployed master on staging :pizza:

> hi
<https://github.com/shipping-co/main-web-app/commit/f00b4r|View latest commit on GitHub>
<https://example.com/main-web-app/deployments/0|Open deployment in Applikatoni>`

	expectedFailMsg := `main-web-app Deploy Failed:
Foo Bar deployed master on staging :pizza:

> hi
<https://github.com/shipping-co/main-web-app/commit/f00b4r|View latest commit on GitHub>
<https://example.com/main-web-app/deployments/0|Open deployment in Applikatoni>`

	actualSuccessMsg, _ := generateSlackSummary(deployment, application, user, true)
	actualFailMsg, _ := generateSlackSummary(deployment, application, user, false)

	if expectedSuccessMsg != actualSuccessMsg {
		t.Errorf("sent wrong message expected=%v got=%v", expectedSuccessMsg, actualSuccessMsg)
	}

	if expectedFailMsg != actualFailMsg {
		t.Errorf("sent wrong message expected=%v got=%v", expectedFailMsg, actualFailMsg)
	}
}
