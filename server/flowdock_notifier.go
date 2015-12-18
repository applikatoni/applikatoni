package main

import (
	"log"
	"net/http"
	"net/url"
	"text/template"

	"github.com/applikatoni/applikatoni/deploy"

	"database/sql"
)

const flowdockTmplStr = `{{.GitHubRepo}} {{if eq .State "successful"}}Successfully Deployed{{else if eq .State "failed"}}Deploy Failed{{end}}:
**{{.Username}}** deployed **{{.Branch}}** on **{{.Target}}** :pizza:

> {{.Comment}}

[View latest commit on GitHub]({{.GitHubUrl}})
[Open deployment in Applikatoni]({{.DeploymentURL}})
`

var flowdockTemplate = template.Must(template.New("flowdockSummary").Parse(flowdockTmplStr))

func NotifyFlowdock(db *sql.DB, deploymentId int) {
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

	if target.FlowdockEndpoint == "" {
		return
	}

	user, err := getUser(db, deployment.UserId)
	if err != nil {
		log.Printf("Could not find User with id %v, %s\n", deployment.UserId, err)
		return
	}

	summary, err := generateSummary(flowdockTemplate, application, deployment, user)
	if err != nil {
		log.Printf("Could not generate deployment summary, %s\n", err)
		return
	}

	params := url.Values{
		"event":   {"message"},
		"content": {summary},
		"tags":    {"deploy,applikatoni"},
	}

	resp, err := http.PostForm(target.FlowdockEndpoint, params)

	if err != nil || resp.StatusCode != 201 {
		log.Printf("Error while notifying Flowdock about deployment of %v on %v, %v! err: %s, resp: %s\n", deployment.ApplicationName,
			deployment.TargetName,
			deployment.CommitSha,
			err,
			resp.Status)
	} else {
		log.Printf("Successfully notified Flowdock about deployment of %v on %v, %v!\n", deployment.ApplicationName, deployment.TargetName, deployment.CommitSha)
	}
}

func newFlowdockNotifier(db *sql.DB) deploy.Listener {
	fn := func(logs <-chan deploy.LogEntry) {
		for entry := range logs {
			if entry.EntryType == deploy.DEPLOYMENT_SUCCESS || entry.EntryType == deploy.DEPLOYMENT_FAIL {
				go NotifyFlowdock(db, entry.DeploymentId)
			}
		}
	}

	return fn
}
