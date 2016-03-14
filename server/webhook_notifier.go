package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/applikatoni/applikatoni/models"
)

type WebhookApplication struct {
	Name        string `json:"application_name"`
	GitHubOwner string `json:"github_owner"`
	GitHubRepo  string `json:"github_repo"`
}

type WebhookDeployment struct {
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

type WebhookTarget struct {
	Name            string                   `json:"target_name"`
	DeploymentUser  string                   `json:"deployment_user"`
	DeployUsernames []string                 `json:"deploy_usernames"`
	Hosts           []*models.Host           `json:"hosts"`
	Roles           []*models.Role           `json:"roles"`
	AvailableStages []models.DeploymentStage `json:"available_stages"`
	DefaultStages   []models.DeploymentStage `json:"default_stages"`
}

type WebhookMsg struct {
	Timestamp time.Time              `json:"timestamp"`
	State     models.DeploymentState `json:"state"`

	Application WebhookApplication `json:"application"`
	Deployment  WebhookDeployment  `json:"deployment"`
	Target      WebhookTarget      `json:"target"`
}

func NotifyWebhooks(ev *DeploymentEvent) {
	if len(ev.Target.Webhooks) == 0 {
		return
	}

	msg := WebhookMsg{
		Timestamp: time.Now(),
		State:     ev.State,
		Application: WebhookApplication{
			Name:        ev.Application.Name,
			GitHubOwner: ev.Application.GitHubOwner,
			GitHubRepo:  ev.Application.GitHubRepo,
		},
		Deployment: WebhookDeployment{
			Id:             ev.Deployment.Id,
			CommitSha:      ev.Deployment.CommitSha,
			Branch:         ev.Deployment.Branch,
			State:          ev.Deployment.State,
			Comment:        ev.Deployment.Comment,
			CreatedAt:      ev.Deployment.CreatedAt,
			URL:            ev.DeploymentURL(),
			DeployerID:     ev.Deployment.UserId,
			DeployerName:   ev.Deployment.User.Name,
			DeployerAvatar: ev.Deployment.User.AvatarUrl,
		},
		Target: WebhookTarget{
			Name:            ev.Target.Name,
			DeploymentUser:  ev.Target.DeploymentUser,
			DeployUsernames: ev.Target.DeployUsernames,
			Hosts:           ev.Target.Hosts,
			Roles:           ev.Target.Roles,
			AvailableStages: ev.Target.AvailableStages,
			DefaultStages:   ev.Target.DefaultStages,
		},
	}

	for _, w := range ev.Target.Webhooks {
		go sendWebhookMsg(w, msg)
	}
}

func sendWebhookMsg(hook string, msg WebhookMsg) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error creating WebhookMsg %s\n", err)
		return
	}

	resp, err := http.Post(hook, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Error while notifying Webhook %s about deployment of %v on %v! err: %s\n",
			hook, msg.Application.Name, msg.Target.Name, err)
		return
	}

	log.Printf("Notified Webhook %s about deployment of %v on %v! Response: %v",
		hook, msg.Application.Name, msg.Target.Name, resp.Status)
}
