package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"
)

type Application struct {
	Name        string `json:"application_name"`
	GitHubOwner string `json:"github_owner"`
	GitHubRepo  string `json:"github_repo"`
}

type Deployment struct {
	Id             int                    `json:"deployment_id"`
	CommitSha      string                 `json:"commit_sha"`
	Branch         string                 `json:"branch"`
	State          models.DeploymentState `json:"state"`
	Comment        string                 `json:"comment"`
	CreatedAt      time.Time              `json:"created_at"`
	URL            string                 `json:"deployment_url"`
	DeployerID     int                    `json:"deployer_id"`
	DeployerName   string                 `json:"deployer_name"`
	DeployerAvatar string                 `json:"deployer_avatar"`
}

type Target struct {
	Name            string                   `json:"target_name"`
	DeploymentUser  string                   `json:"deployment_user"`
	DeployUsernames []string                 `json:"deploy_usernames"`
	Hosts           []*models.Host           `json:"hosts"`
	Roles           []*models.Role           `json:"roles"`
	AvailableStages []models.DeploymentStage `json:"available_stages"`
	DefaultStages   []models.DeploymentStage `json:"default_stages"`
}

type WebHookMsg struct {
	Timestamp time.Time           `json:"timestamp"`
	Origin    string              `json:"origin"`
	EntryType deploy.LogEntryType `json:"entry_type"`
	Message   string              `json:"message"`

	Application Application `json:"application"`
	Deployment  Deployment  `json:"deployment"`
	Target      Target      `json:"target"`
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

	if len(target.Webhooks) == 0 {
		return
	}

	scheme := "http"
	if config.SSLEnabled {
		scheme = "https"
	}

	deploymentUrl := fmt.Sprintf("%s://%s/%v/deployments/%v", scheme, config.Host,
		application.GitHubRepo, deployment.Id)

	deployment.User, err = getUser(db, deployment.UserId)
	if err != nil {
		log.Printf("Could not find user with id %v, %s\n", deployment.UserId, err)
		return
	}

	msg := WebHookMsg{
		Timestamp: entry.Timestamp,
		Origin:    entry.Origin,
		EntryType: entry.EntryType,
		Message:   entry.Message,
		Application: Application{
			Name:        application.Name,
			GitHubOwner: application.GitHubOwner,
			GitHubRepo:  application.GitHubRepo,
		},
		Deployment: Deployment{
			Id:             deployment.Id,
			CommitSha:      deployment.CommitSha,
			Branch:         deployment.Branch,
			State:          deployment.State,
			Comment:        deployment.Comment,
			CreatedAt:      deployment.CreatedAt,
			URL:            deploymentUrl,
			DeployerID:     deployment.UserId,
			DeployerName:   deployment.User.Name,
			DeployerAvatar: deployment.User.AvatarUrl,
		},
		Target: Target{
			Name:            target.Name,
			DeploymentUser:  target.DeploymentUser,
			DeployUsernames: target.DeployUsernames,
			Hosts:           target.Hosts,
			Roles:           target.Roles,
			AvailableStages: target.AvailableStages,
			DefaultStages:   target.DefaultStages,
		},
	}

	for _, w := range target.Webhooks {
		if w.IsEntryWanted(string(entry.EntryType)) {
			go sendWebhookMsg(w.URL, msg)
		}
	}
}

func sendWebhookMsg(hook string, msg WebHookMsg) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error creating WebhookMsg %s\n", err)
		return
	}

	resp, err := http.Post(hook, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Error while notifying Webhook %s about deployment of %v on %v! err: %s\n",
			hook,
			msg.Application.Name,
			msg.Target.Name,
			err)
	} else {
		log.Printf("Notified Webhook %s about deployment of %v on %v! Response: %v",
			hook,
			msg.Application.Name,
			msg.Target.Name,
			resp.Status)
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
