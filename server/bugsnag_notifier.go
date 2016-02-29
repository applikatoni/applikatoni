package main

import (
	"database/sql"
	"log"
	"net/http"
	"net/url"
)

const (
	bugsnagNotifyEndpoint = "https://notify.bugsnag.com/deploy"
)

func NotifyBugsnag(db *sql.DB, ev *DeploymentEvent) {
	if ev.Target.BugsnagApiKey != "" {
		SendBugsnagRequest(bugsnagNotifyEndpoint, ev)
	}
}

func SendBugsnagRequest(endpoint string, ev *DeploymentEvent) {
	params := url.Values{
		"apiKey":       {ev.Target.BugsnagApiKey},
		"releaseStage": {ev.Deployment.TargetName},
		"repository":   {ev.Application.RepositoryURL()},
		"branch":       {ev.Deployment.Branch},
		"revision":     {ev.Deployment.CommitSha},
	}

	resp, err := http.PostForm(endpoint, params)
	if err != nil {
		log.Printf("Notifying Bugsnag failed (%s on %s, %s): err=%s\n",
			ev.Application.Name, ev.Target.Name, ev.Deployment.CommitSha, err)
		return
	}
	if resp.StatusCode != 200 {
		log.Printf("Notifying Bugsnag failed (%s on %s, %s): status=%d\n",
			ev.Application.Name, ev.Target.Name, ev.Deployment.CommitSha,
			resp.StatusCode)
		return
	}

	log.Printf("Successfully notified Bugsnag about deployment of %s on %s, %s!\n",
		ev.Application.Name, ev.Target.Name, ev.Deployment.CommitSha)
}
