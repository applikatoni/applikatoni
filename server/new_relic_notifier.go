package main

import (
	"bytes"
	"log"
	"net/http"
	"net/url"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"

	"database/sql"
)

const (
	newRelicNotifyEndpoint = "https://api.newrelic.com/deployments.xml"
)

func NotifyNewRelic(db *sql.DB, endpoint string, deploymentId int) {
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

	user, err := getUser(db, deployment.UserId)
	if err != nil {
		log.Printf("Could not find User with id %v, %s\n", deployment.UserId, err)
		return
	}

	if target.NewRelicApiKey == "" || target.NewRelicAppId == "" {
		return
	}

	SendNewRelicRequest(newRelicNotifyEndpoint, deployment, target, application, user)
}

func SendNewRelicRequest(endpoint string, d *models.Deployment, t *models.Target, a *models.Application, u *models.User) {
	summary, err := generateSummary(a, d, u)
	if err != nil {
		log.Printf("Could not generate deployment summary, %s\n", err)
	}

	data := url.Values{}
	data.Set("deployment[app_id]", t.NewRelicAppId)
	data.Set("deployment[description]", d.Comment)
	data.Set("deployment[revision]", d.CommitSha)
	data.Set("deployment[user]", u.Name)
	data.Set("deployment[changelog]", summary)

	client := &http.Client{}

	// post URL-encoded payload, must satisfy io interface
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	req.Header.Set("x-api-key", t.NewRelicApiKey)

	resp, err := client.Do(req)

	if err != nil || resp.StatusCode != 201 {
		log.Printf("Error while notifying New Relic about deployment of %v on %v, %v! err: %s, resp: %s\n",
			d.ApplicationName,
			d.TargetName,
			d.CommitSha,
			err,
			resp.Status)
		return
	}

	log.Printf("Successfully notified New Relic about deployment of %v on %v, %v!\n", d.ApplicationName, d.TargetName, d.CommitSha)
}

func newNewRelicNotifier(db *sql.DB) deploy.Listener {
	fn := func(logs <-chan deploy.LogEntry) {
		for entry := range logs {
			if entry.EntryType == deploy.DEPLOYMENT_SUCCESS {
				go NotifyNewRelic(db, bugsnagNotifyEndpoint, entry.DeploymentId)
			}
		}
	}

	return fn
}
