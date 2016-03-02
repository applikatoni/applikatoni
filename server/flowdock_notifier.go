package main

import (
	"log"
	"net/http"
	"net/url"
	"text/template"

	"github.com/applikatoni/applikatoni/models"
)

const flowdockTmplStr = `{{.GitHubRepo}} {{if .Success}}Successfully Deployed{{else}}Deploy Failed{{end}}:
**{{.Username}}** deployed **{{.Branch}}** on **{{.Target}}** :pizza:

> {{.Comment}}

[View latest commit on GitHub]({{.GitHubUrl}})
[Open deployment in Applikatoni]({{.DeploymentURL}})
`

var flowdockTemplate = template.Must(template.New("flowdockSummary").Parse(flowdockTmplStr))

func NotifyFlowdock(ev *DeploymentEvent) {
	if ev.Target.FlowdockEndpoint == "" {
		return
	}

	summary, err := generateSummary(flowdockTemplate, ev)
	if err != nil {
		log.Printf("Could not generate deployment summary, %s\n", err)
		return
	}

	SendFlowdockRequest(ev.Target.FlowdockEndpoint, ev.Deployment, summary)
}

func SendFlowdockRequest(endpoint string, d *models.Deployment, summary string) {
	params := url.Values{
		"event":   {"message"},
		"content": {summary},
		"tags":    {"deploy,applikatoni"},
	}

	resp, err := http.PostForm(endpoint, params)
	if err != nil {
		log.Printf("Notifying Flowdock failed (%s on %s, %s): err=%s\n",
			d.ApplicationName, d.TargetName, d.CommitSha, err)
		return
	}
	if resp.StatusCode != 201 {
		log.Printf("Notifying Flowdock failed (%s on %s, %s): status=%d\n",
			d.ApplicationName, d.TargetName, d.CommitSha, resp.StatusCode)
		return
	}

	log.Printf("Successfully notified Flowdock about deployment of %s on %s, %s!\n",
		d.ApplicationName, d.TargetName, d.CommitSha)
}
