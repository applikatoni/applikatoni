package main

import (
	"database/sql"
	"testing"

	"github.com/applikatoni/applikatoni/models"
)

func TestSubscribe(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	testSubscriber := func(db *sql.DB, ev *DeploymentEvent) {}

	hub := NewDeploymentEventHub(db)
	hub.Subscribe(testSubscriber, []models.DeploymentState{models.DEPLOYMENT_NEW})

	if len(hub.Subscribers[models.DEPLOYMENT_NEW]) != 1 {
		t.Errorf("subscriber not added.")
	}
}

func TestPublish(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	user := buildUser(12345, "mrnugget")
	err := createUser(db, user)
	checkErr(t, err)

	deployment := buildDeployment(user.Id)
	err = createDeployment(db, deployment)
	checkErr(t, err)

	target := &models.Target{Name: deployment.TargetName}
	application := &models.Application{
		Name:    deployment.ApplicationName,
		Targets: []*models.Target{target},
	}
	config = &Configuration{Applications: []*models.Application{application}}

	testDone := make(chan struct{})
	testSubscriber := func(db *sql.DB, ev *DeploymentEvent) {
		if ev.User.Id != user.Id {
			t.Errorf("deployment event has wrong user")
		}

		if ev.Deployment.Id != deployment.Id {
			t.Errorf("subscriber called with wrong deployment event")
		}

		if ev.Application != application {
			t.Errorf("deployment event has wrong application")
		}

		if ev.Target.Name != target.Name {
			t.Errorf("deployment event has wrong target.")
		}

		testDone <- struct{}{}
	}

	hub := NewDeploymentEventHub(db)
	hub.Subscribe(testSubscriber, []models.DeploymentState{models.DEPLOYMENT_NEW})

	hub.Publish(models.DEPLOYMENT_NEW, deployment)

	<-testDone
}
