package main

import (
	"log"
	"net/http"
	"net/url"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"

	"database/sql"
)

const (
	bugsnagNotifyEndpoint = "https://notify.bugsnag.com/deploy"
)

func NotifyBugsnag(db *sql.DB, endpoint string, deploymentId int) {
	deployment, err := getDeployment(db, deploymentId)
	if err != nil {
		log.Printf("Could not find deployment with id %v, %s\n", deploymentId, err)
		return
	}

	application, err := findApplication(deployment.ApplicationName)
	if err != nil {
		log.Printf("Could not find application with name %v, %s\n", deployment.ApplicationName, err)
		return
	}

	target, err := findTarget(application, deployment.TargetName)
	if err != nil {
		log.Printf("Could not find target with name %v, %s\n", deployment.TargetName, err)
		return
	}

	if target.BugsnagApiKey == "" {
		return
	}

	SendBugsnagRequest(bugsnagNotifyEndpoint, deployment, target, application)
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
		log.Printf("Notifying Bugsnag failed (%s on %s, %s): status=%s\n",
			d.ApplicationName, d.TargetName, d.CommitSha, resp.StatusCode)
		return
	}

	log.Printf("Successfully notified Bugsnag about deployment of %s on %s, %s!\n",
		d.ApplicationName, d.TargetName, d.CommitSha)
}

func newBugsnagNotifier(db *sql.DB) deploy.Listener {
	fn := func(logs <-chan deploy.LogEntry) {
		for entry := range logs {
			if entry.EntryType == deploy.DEPLOYMENT_SUCCESS {
				go NotifyBugsnag(db, bugsnagNotifyEndpoint, entry.DeploymentId)
			}
		}
	}

	return fn
}
