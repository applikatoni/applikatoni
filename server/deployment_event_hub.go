package main

import (
	"database/sql"
	"fmt"

	"github.com/applikatoni/applikatoni/models"
)

type Subscriber func(*sql.DB, *DeploymentEvent)

type DeploymentEventHub struct {
	db          *sql.DB
	Subscribers map[models.DeploymentState][]Subscriber
}

func NewDeploymentEventHub(db *sql.DB) *DeploymentEventHub {
	hub := &DeploymentEventHub{}

	hub.db = db

	hub.Subscribers = make(map[models.DeploymentState][]Subscriber)
	hub.Subscribers[models.DEPLOYMENT_NEW] = []Subscriber{}
	hub.Subscribers[models.DEPLOYMENT_ACTIVE] = []Subscriber{}
	hub.Subscribers[models.DEPLOYMENT_SUCCESSFUL] = []Subscriber{}
	hub.Subscribers[models.DEPLOYMENT_FAILED] = []Subscriber{}

	return hub
}

func (hub *DeploymentEventHub) Subscribe(states []models.DeploymentState, s Subscriber) {
	for _, state := range states {
		hub.Subscribers[state] = append(hub.Subscribers[state], s)
	}
}

func (hub *DeploymentEventHub) Publish(state models.DeploymentState, d *models.Deployment) {
	subscribers := hub.Subscribers[state]
	if len(subscribers) == 0 {
		return
	}

	deployment, err := getDeployment(hub.db, d.Id)
	if err != nil {
		return
	}

	application, err := findApplication(d.ApplicationName)
	if err != nil {
		err = fmt.Errorf("Could not find application with name %q, %s\n", deployment.ApplicationName, err)
		return
	}

	target, err := findTarget(application, deployment.TargetName)
	if err != nil {
		return
	}

	user, err := getUser(hub.db, deployment.UserId)
	if err != nil {
		return
	}

	event := &DeploymentEvent{
		Deployment:  d,
		Application: application,
		Target:      target,
		User:        user,
	}

	for _, subscriber := range subscribers {
		go subscriber(hub.db, event)
	}
}
