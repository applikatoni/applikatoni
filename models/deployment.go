package models

import "time"

type DeploymentState string

const (
	DEPLOYMENT_NEW        DeploymentState = "new"
	DEPLOYMENT_ACTIVE     DeploymentState = "active"
	DEPLOYMENT_SUCCESSFUL DeploymentState = "successful"
	DEPLOYMENT_FAILED     DeploymentState = "failed"
)

type Deployment struct {
	Id              int
	CommitSha       string
	Branch          string
	State           DeploymentState
	Comment         string
	CreatedAt       time.Time
	UserId          int
	User            *User
	ApplicationName string
	TargetName      string
}
