package deploy

import (
	"fmt"

	"github.com/applikatoni/applikatoni/models"
	"golang.org/x/crypto/ssh"
)

type Manager struct {
	config *models.DeploymentConfig

	workers   []*Worker
	sshConfig *ssh.ClientConfig

	logger *DeploymentLogger

	killChan chan struct{}
}

func NewManager(c *models.DeploymentConfig, r *LogRouter, kc chan struct{}) (*Manager, error) {
	ssh, err := newSSHClientConfig(c.User, c.SshKey)
	if err != nil {
		return nil, err
	}

	logger := NewDeploymentLogger(c.Deployment, r)

	m := &Manager{
		config:    c,
		sshConfig: ssh,
		logger:    logger,
		killChan:  kc,
	}

	err = m.assembleWorkers()
	if err != nil {
		return nil, err
	}

	return m, nil
}

// AnnounceStart starts the broadcasting of LogEntries with the DeploymentLogger
// and logs the start of the deployment
func (m *Manager) AnnounceStart() {
	m.logger.BroadcastLogs()
	m.logger.LogDeploymentStart()
}

func (m *Manager) Start() error {
	defer m.logger.Flush()

	err := m.connectWorkers()
	if err != nil {
		m.disconnectWorkers()
		m.logger.LogDeploymentFail(err)
		return err
	}
	defer m.disconnectWorkers()

	for _, stage := range m.config.Stages {
		err := m.executeStage(stage)
		if err != nil {
			m.logger.LogDeploymentFail(err)
			return err
		}
	}

	m.logger.LogDeploymentSuccess()
	return nil
}

func (m *Manager) assembleWorkers() error {
	configOptions := m.config.ScriptOptions()

	for _, h := range m.config.Hosts {
		w, err := m.newWorker(h, configOptions)
		if err != nil {
			return err
		}

		m.workers = append(m.workers, w)
	}

	return nil
}

func (m *Manager) connectWorkers() error {
	for _, w := range m.workers {
		err := w.Connect()
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) disconnectWorkers() {
	for _, w := range m.workers {
		w.Close()
	}
}

func (m *Manager) executeStage(stage models.DeploymentStage) error {
	stageName := string(stage)

	m.logger.LogStageStart(stage)

	results := m.executeWorkersStage(stage)
	stageFailed := false

	for i := 0; i < len(results); i++ {
		var msg string

		result := results[i]
		if result.err != nil {
			stageFailed = true

			msg = fmtStageFailure(stage, result)
		} else {
			if result.skipped {
				msg = fmtStageSkipped(stage, result)
			} else {
				msg = fmtStageSuccess(stage, result)
			}
		}
		m.logger.LogStageResult(msg)
	}

	if stageFailed {
		m.logger.LogStageFail(stage)
		err := fmt.Errorf("Execution of stage %s failed", stageName)
		return err
	}

	m.logger.LogStageSuccess(stage)
	return nil
}

func (m *Manager) executeWorkersStage(stage models.DeploymentStage) []ExecutionResult {
	results := []ExecutionResult{}
	ch := make(chan ExecutionResult)

	exec := func(w *Worker) {
		ch <- w.Execute(stage)
	}

	for _, w := range m.workers {
		go exec(w)
	}

	for i := 0; i < len(m.workers); i++ {
		select {
		case result := <-ch:
			results = append(results, result)
		case <-m.killChan:
			// Received kill first. Log this, add result, wait for worker
			m.logger.LogKillReceived()
			errMsg := fmt.Errorf("Received kill signal")
			results = append(results, ExecutionResult{origin: "applikatoni", err: errMsg})

			result := <-ch
			results = append(results, result)
		}
	}

	return results
}

func (m *Manager) newWorker(h *models.Host, scriptOptions map[string]string) (*Worker, error) {
	roles, err := findHostRoles(h, m.config.Roles)
	if err != nil {
		return nil, err
	}

	rolesScripts := []map[models.DeploymentStage]string{}
	for _, r := range roles {
		s, err := r.RenderScripts(scriptOptions)
		if err != nil {
			return nil, err
		}
		rolesScripts = append(rolesScripts, s)
	}

	mergedScripts := make(map[models.DeploymentStage]string)
	for _, s := range rolesScripts {
		for stage, scriptContent := range s {
			if _, alreadyExists := mergedScripts[stage]; alreadyExists {
				err := fmt.Errorf("merging host scripts failed. script for %s is duplicate", stage)
				return nil, err
			}
			mergedScripts[stage] = scriptContent
		}
	}

	w := &Worker{
		host:      h,
		scripts:   mergedScripts,
		sshConfig: m.sshConfig,
		logger:    m.logger,
	}
	return w, nil
}

func findHostRoles(h *models.Host, roles []*models.Role) ([]*models.Role, error) {
	found := []*models.Role{}

	for _, r := range roles {
		for _, hostRole := range h.Roles {
			if r.Name == hostRole {
				found = append(found, r)
			}
		}
	}

	if len(found) == 0 {
		err := fmt.Errorf("No matching roles for host %s with roles %s found", h.Name, h.Roles)
		return nil, err
	}

	return found, nil
}

func fmtStageSuccess(s models.DeploymentStage, r ExecutionResult) string {
	msg := fmt.Sprintf("%s - execution of stage %s successful (%s)", r.origin, string(s), r.timeTaken)
	return msg
}

func fmtStageFailure(s models.DeploymentStage, r ExecutionResult) string {
	msg := fmt.Sprintf("%s - execution of stage %s failed: %s (%s)", r.origin, string(s), r.err, r.timeTaken)
	return msg
}

func fmtStageSkipped(s models.DeploymentStage, r ExecutionResult) string {
	return fmt.Sprintf("%s - stage %s skipped", r.origin, string(s))
}
