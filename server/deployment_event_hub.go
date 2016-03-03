package main

import (
	"database/sql"
	"log"

	"github.com/applikatoni/applikatoni/models"
)

type DeploymentEvent struct {
	State       models.DeploymentState
	Deployment  *models.Deployment
	Application *models.Application
	Target      *models.Target
	User        *models.User
}

type Subscriber func(*DeploymentEvent)

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

	event, err := hub.buildDeploymentEvent(state, d)
	if err != nil {
		log.Printf("Building deployment for deployment %d failed: %s\n", d.Id,
			err)
		return
	}

	for _, subscriber := range subscribers {
		go subscriber(event)
	}
}

func (hub *DeploymentEventHub) buildDeploymentEvent(s models.DeploymentState, d *models.Deployment) (*DeploymentEvent, error) {
	user, err := getUser(hub.db, d.UserId)
	if err != nil {
		return nil, err
	}
	d.User = user

	application, err := findApplication(d.ApplicationName)
	if err != nil {
		return nil, err
	}

	target, err := findTarget(application, d.TargetName)
	if err != nil {
		return nil, err
	}

	event := &DeploymentEvent{
		State:       s,
		Deployment:  d,
		Application: application,
		Target:      target,
		User:        user,
	}

	return event, nil
}
