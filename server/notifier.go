package main

import (
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
