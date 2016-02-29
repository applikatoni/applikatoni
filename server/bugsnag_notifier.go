package main

import (
	"database/sql"
	"log"
	"net/http"
	"net/url"

	"github.com/applikatoni/applikatoni/models"
)

const (
	bugsnagNotifyEndpoint = "https://notify.bugsnag.com/deploy"
)

func NotifyBugsnag(db *sql.DB, ev *DeploymentEvent) {
	if ev.Target.BugsnagApiKey != "" {
		SendBugsnagRequest(bugsnagNotifyEndpoint, ev.Deployment, ev.Target, ev.Application)
	}
}

func SendBugsnagRequest(endpoint string, d *models.Deployment, t *models.Target, a *models.Application) {
	params := url.Values{
		"apiKey":       {t.BugsnagApiKey},
		"releaseStage": {d.TargetName},
		"repository":   {a.RepositoryURL()},
		"branch":       {d.Branch},
		"revision":     {d.CommitSha},
	}

	resp, err := http.PostForm(endpoint, params)
	if err != nil {
		log.Printf("Notifying Bugsnag failed (%s on %s, %s): err=%s\n",
			d.ApplicationName, d.TargetName, d.CommitSha, err)
		return
	}
	if resp.StatusCode != 200 {
		log.Printf("Notifying Bugsnag failed (%s on %s, %s): status=%d\n",
			d.ApplicationName, d.TargetName, d.CommitSha, resp.StatusCode)
		return
	}

	log.Printf("Successfully notified Bugsnag about deployment of %s on %s, %s!\n",
		d.ApplicationName, d.TargetName, d.CommitSha)
}
