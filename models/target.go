package models

type Target struct {
	Name             string            `json:"name"`
	DeploymentUser   string            `json:"deployment_user"`
	DeploymentSshKey string            `json:"deployment_ssh_key"`
	DeployUsernames  []string          `json:"deploy_usernames"`
	Hosts            []*Host           `json:"hosts"`
	Roles            []*Role           `json:"roles"`
	AvailableStages  []DeploymentStage `json:"available_stages"`
	DefaultStages    []DeploymentStage `json:"default_stages"`
	BugsnagApiKey    string            `json:"bugsnag_api_key"`
	FlowdockEndpoint string            `json:"flowdock_endpoint"`
	NewRelicApiKey   string            `json:"new_relic_api_key"`
	NewRelicAppId    string            `json:"new_relic_app_id"`
	SlackUrl         string            `json:"slack_url"`
}

func (t *Target) IsDeployer(userName string) bool {
	return isInList(userName, t.DeployUsernames)
}

func (t *Target) IsDefaultStage(s DeploymentStage) bool {
	for _, def := range t.DefaultStages {
		if def == s {
			return true
		}
	}

	return false
}

func (t *Target) AreValidStages(stages []DeploymentStage) bool {
	stagesLength := len(stages)
	posInAvailable := func(s DeploymentStage) int {
		for i, v := range t.AvailableStages {
			if s == v {
				return i
			}
		}
		return -1
	}

	for i := 0; i < stagesLength; i++ {
		currentPos := posInAvailable(stages[i])
		if currentPos == -1 {
			return false
		}
		if i+1 < stagesLength {
			if nextPos := posInAvailable(stages[i+1]); currentPos > nextPos {
				return false
			}
		}

	}
	return true
}
