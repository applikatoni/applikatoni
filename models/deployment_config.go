package models

import "time"

const assetsTimestampLayout string = "200601021504.05"

func NewDeploymentConfig(d *Deployment, t *Target, stages []DeploymentStage) *DeploymentConfig {
	return &DeploymentConfig{
		User:       t.DeploymentUser,
		SshKey:     []byte(t.DeploymentSshKey),
		Stages:     stages,
		Hosts:      t.Hosts,
		Roles:      t.Roles,
		StartTime:  time.Now(),
		Deployment: d,
	}
}

type DeploymentConfig struct {
	User       string
	SshKey     []byte
	Stages     []DeploymentStage
	Hosts      []*Host
	Roles      []*Role
	StartTime  time.Time
	Deployment *Deployment
}

func (dc *DeploymentConfig) ScriptOptions() map[string]string {
	return map[string]string{
		"CommitSha":       dc.Deployment.CommitSha,
		"AssetsTimestamp": dc.StartTime.UTC().Format(assetsTimestampLayout),
	}
}
