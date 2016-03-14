package main

import (
	"log"
	"sync"

	"github.com/applikatoni/applikatoni/models"
)

var githubDeployments map[int]*GitHubDeployment
var githubDeploymentsMutex *sync.Mutex

type GitHubNotifier struct {
	deployments map[int]*GitHubDeployment
	mutex       *sync.Mutex
}

func NewGitHubNotifier() *GitHubNotifier {
	return &GitHubNotifier{
		deployments: make(map[int]*GitHubDeployment),
		mutex:       &sync.Mutex{},
	}
}

func (notifier *GitHubNotifier) Notify(ev *DeploymentEvent) {
	notifier.mutex.Lock()
	defer notifier.mutex.Unlock()

	ghClient := NewGitHubClient(ev.User)

	if ev.State == models.DEPLOYMENT_NEW {
		githubDeployment, err := ghClient.CreateDeployment(ev.Application, ev.Deployment)
		if err != nil {
			log.Printf("Creating GitHub deployment failed: %s\n", err)
			return
		}
		notifier.deployments[ev.Deployment.Id] = githubDeployment
	} else {
		githubDeployment, ok := notifier.deployments[ev.Deployment.Id]
		if !ok {
			log.Printf("No GitHubDeployment for %d found\n", ev.Deployment.Id)
			return
		}
		err := ghClient.CreateDeploymentStatus(githubDeployment, ev.State)
		if err != nil {
			log.Printf("Creating GitHub deployment status failed: %s\n", err)
			return
		}
	}
}