package main

import (
	"database/sql"
	"testing"

	"github.com/applikatoni/applikatoni/models"
)

func TestSubscribe(t *testing.T) {
	testSubscriber := func(ev *DeploymentEvent) {}

	hub := NewDeploymentEventHub(&sql.DB{})
	hub.Subscribe([]models.DeploymentState{models.DEPLOYMENT_NEW}, testSubscriber)

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
	testSubscriber := func(ev *DeploymentEvent) {
		if ev.State != models.DEPLOYMENT_NEW {
			t.Errorf("deployment event has wrong state")
		}

		if ev.User.Id != user.Id {
			t.Errorf("deployment event has wrong user")
		}

		if ev.Deployment.Id != deployment.Id {
			t.Errorf("subscriber called with wrong deployment event")
		}

		if ev.Deployment.User == nil {
			t.Errorf("deployment user in event not set")
		}
		if ev.Deployment.User.Id != user.Id {
			t.Errorf("deployment user is wrong")
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
	hub.Subscribe([]models.DeploymentState{models.DEPLOYMENT_NEW}, testSubscriber)

	hub.Publish(models.DEPLOYMENT_NEW, deployment)

	<-testDone
}

func TestDeploymentEventDeploymentURL(t *testing.T) {
	config = &Configuration{
		Host:       "example.com",
		SSLEnabled: true,
	}

	deployment := &models.Deployment{
		Id:              999999,
		UserId:          888888,
		CommitSha:       "f133742",
		Branch:          "master",
		Comment:         "Deploying a hotfix",
		ApplicationName: "my-web-app",
		TargetName:      "production",
	}
	user := &models.User{
		Name: "mrnugget",
		Id:   deployment.UserId,
	}
	target := &models.Target{Name: "production"}
	application := &models.Application{
		Name:        "my-web-app",
		GitHubRepo:  "my-web-app",
		GitHubOwner: "shipping-co",
	}

	event := &DeploymentEvent{
		State:       models.DEPLOYMENT_SUCCESSFUL,
		Deployment:  deployment,
		Application: application,
		Target:      target,
		User:        user,
	}

	deploymentURL := event.DeploymentURL()
	if deploymentURL != "https://example.com/my-web-app/deployments/999999" {
		t.Errorf("DeploymentURL() returned wrong url. got=%q", deploymentURL)
	}
}
