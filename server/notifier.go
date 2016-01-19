package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"
)

type DeploymentEvent struct {
	Entry       deploy.LogEntry
	Deployment  *models.Deployment
	Application *models.Application
	Target      *models.Target
	User        *models.User
}

func NewDeploymentEvent(e deploy.LogEntry) (*DeploymentEvent, error) {
	deployment, err := getDeployment(db, e.DeploymentId)
	if err != nil {
		err = fmt.Errorf("Could not find deployment with id %d, %s\n", e.DeploymentId, err)
		return nil, err
	}

	application, err := findApplication(deployment.ApplicationName)
	if err != nil {
		err = fmt.Errorf("Could not find application with name %q, %s\n", deployment.ApplicationName, err)
		return nil, err
	}

	target, err := findTarget(application, deployment.TargetName)
	if err != nil {
		err = fmt.Errorf("Could not find target with name %q, %s\n", deployment.TargetName, err)
		return nil, err
	}

	user, err := getUser(db, deployment.UserId)
	if err != nil {
		err = fmt.Errorf("Could not find User with id %id, %s\n", deployment.UserId, err)
		return nil, err
	}

	event := &DeploymentEvent{
		Entry:       e,
		Deployment:  deployment,
		Application: application,
		Target:      target,
		User:        user,
	}

	return event, nil
}

type Notifier func(*DeploymentEvent)

func NewDeploymentListener(db *sql.DB, fn Notifier, evs []deploy.LogEntryType) deploy.Listener {
	return func(logs <-chan deploy.LogEntry) {
		for entry := range logs {
			for _, entryType := range evs {
				if entry.EntryType != entryType {
					continue
				}

				go func() {
					event, err := NewDeploymentEvent(entry)
					if err != nil {
						log.Printf("Error creating DeploymentEvent. error=%s\n", err)
						return
					}

					fn(event)
				}()
			}
		}
	}
}
