package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"
)

type WebHookMsg struct {
	Timestamp time.Time           `json:"timestamp"`
	Origin    string              `json:"origin"`
	EntryType deploy.LogEntryType `json:"entry_type"`
	Message   string              `json:"message"`

	//Application
	ApplicationName string `json:"application_name"`
	GitHubOwner     string `json:"github_owner"`
	GitHubRepo      string `json:"github_repo"`

	//Deployment
	DeploymentId int                    `json:"deployment_id"`
	CommitSha    string                 `json:"commit_sha"`
	Branch       string                 `json:"branch"`
	State        models.DeploymentState `json:"state"`
	Comment      string                 `json:"comment"`
	CreatedAt    time.Time              `json:"created_at"`

	//Target
	TargetName      string                   `json:"target_name"`
	DeploymentUser  string                   `json:"deployment_user"`
	DeployUsernames []string                 `json:"deploy_usernames"`
	Hosts           []*models.Host           `json:"hosts"`
	Roles           []*models.Role           `json:"roles"`
	AvailableStages []models.DeploymentStage `json:"available_stages"`
	DefaultStages   []models.DeploymentStage `json:"default_stages"`
}

func NotifyWebhooks(db *sql.DB, entry deploy.LogEntry) {
	deployment, err := getDeployment(db, entry.DeploymentId)
	if err != nil {
		log.Printf("Could not find deployment with id %v, %s\n", entry.DeploymentId, err)
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

	msg := WebHookMsg{
		Timestamp:       entry.Timestamp,
		Origin:          entry.Origin,
		EntryType:       entry.EntryType,
		Message:         entry.Message,
		ApplicationName: application.Name,
		GitHubOwner:     application.GitHubOwner,
		GitHubRepo:      application.GitHubRepo,
		DeploymentId:    deployment.Id,
		CommitSha:       deployment.CommitSha,
		Branch:          deployment.Branch,
		State:           deployment.State,
		Comment:         deployment.Comment,
		CreatedAt:       deployment.CreatedAt,
		TargetName:      target.Name,
		DeploymentUser:  target.DeploymentUser,
		DeployUsernames: target.DeployUsernames,
		Hosts:           target.Hosts,
		Roles:           target.Roles,
		AvailableStages: target.AvailableStages,
		DefaultStages:   target.DefaultStages,
	}

	for i := range target.WebHooks {
		go sendWebhookMsg(target.WebHooks[i], msg)
	}
}

func sendWebhookMsg(hook string, msg WebHookMsg) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error creating WebhookMsg %s\n", err)
		return
	}

	_, err = http.Post(hook, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Error while notifying Webhook %s about deployment of %v on %v! err: %s\n",
			hook,
			msg.ApplicationName,
			msg.TargetName,
			err)
	} else {
		log.Printf("Successfully notified Webhook %s about deployment of %v on %v!\n",
			hook,
			msg.ApplicationName,
			msg.TargetName)
	}
}

func newWebHookNotifier(db *sql.DB) deploy.Listener {
	fn := func(logs <-chan deploy.LogEntry) {
		for entry := range logs {
			go NotifyWebhooks(db, entry)
		}
	}
	return fn
}
