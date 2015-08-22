package deploy

import (
	"fmt"
	"sync"
	"time"

	"github.com/applikatoni/applikatoni/models"
)

type DeploymentLogger struct {
	deployment *models.Deployment
	router     *LogRouter

	// A buffered channel to which workers/managers can send their logs
	// It's buffered in order to not block the workers while they do their
	// work.
	ch chan LogEntry

	// Used internally to Flush() the logs to the router. Incremented when
	// LogEntry is added to the queue and decremented when successfully sent to
	// the router. Only returns on Wait() if all logs have been sent to the
	// router. Used in Flush().
	wg sync.WaitGroup
}

func NewDeploymentLogger(d *models.Deployment, r *LogRouter) *DeploymentLogger {
	return &DeploymentLogger{
		deployment: d,
		router:     r,
		ch:         make(chan LogEntry, 100),
		wg:         sync.WaitGroup{},
	}
}

func (l *DeploymentLogger) BroadcastLogs() {
	l.router.Announce(l.deployment.Id)

	go func() {
		for entry := range l.ch {
			entry.DeploymentId = l.deployment.Id
			l.router.Broadcast <- entry
			l.wg.Done()
		}
	}()
}

func (l *DeploymentLogger) Log(entry LogEntry) {
	l.wg.Add(1)
	l.ch <- entry
}

func (l *DeploymentLogger) Flush() {
	l.wg.Wait() // Wait for `ch` to drain
	close(l.ch)
	l.router.Done <- l.deployment.Id
}

func (l *DeploymentLogger) LogCmdStart(origin, cmd string) {
	entry := LogEntry{
		Origin:    origin,
		EntryType: COMMAND_START,
		Message:   cmd,
		Timestamp: time.Now(),
	}

	l.Log(entry)
}

func (l *DeploymentLogger) LogCmdFail(origin, cmd string, err error) {
	entry := LogEntry{
		Origin:    origin,
		EntryType: COMMAND_FAIL,
		Message:   fmt.Sprintf("cmd=\"%s\", error=\"%s\"", cmd, err),
		Timestamp: time.Now(),
	}

	l.Log(entry)
}

func (l *DeploymentLogger) LogCmdSuccess(origin, cmd string) {
	entry := LogEntry{
		Origin:    origin,
		EntryType: COMMAND_SUCCESS,
		Message:   fmt.Sprintf("\"%s\"", cmd),
		Timestamp: time.Now(),
	}

	l.Log(entry)
}

func (l *DeploymentLogger) LogStageStart(stage models.DeploymentStage) {
	entry := LogEntry{
		Origin:    "applikatoni",
		EntryType: STAGE_START,
		Message:   string(stage),
		Timestamp: time.Now(),
	}

	l.Log(entry)
}

func (l *DeploymentLogger) LogStageResult(msg string) {
	entry := LogEntry{
		Origin:    "applikatoni",
		EntryType: STAGE_RESULT,
		Message:   msg,
		Timestamp: time.Now(),
	}

	l.Log(entry)
}

func (l *DeploymentLogger) LogStageFail(stage models.DeploymentStage) {
	entry := LogEntry{
		Origin:    "applikatoni",
		EntryType: STAGE_FAIL,
		Message:   string(stage),
		Timestamp: time.Now(),
	}

	l.Log(entry)
}

func (l *DeploymentLogger) LogStageSuccess(stage models.DeploymentStage) {
	entry := LogEntry{
		Origin:    "applikatoni",
		EntryType: STAGE_SUCCESS,
		Message:   string(stage),
		Timestamp: time.Now(),
	}

	l.Log(entry)
}

func (l *DeploymentLogger) LogDeploymentStart() {
	entry := LogEntry{
		Origin:    "applikatoni",
		EntryType: DEPLOYMENT_START,
		Message:   fmt.Sprintf("deployment_id=%d", l.deployment.Id),
		Timestamp: time.Now(),
	}

	l.Log(entry)
}

func (l *DeploymentLogger) LogDeploymentSuccess() {
	entry := LogEntry{
		Origin:    "applikatoni",
		EntryType: DEPLOYMENT_SUCCESS,
		Message:   fmt.Sprintf("deployment_id=%d", l.deployment.Id),
		Timestamp: time.Now(),
	}

	l.Log(entry)
}

func (l *DeploymentLogger) LogDeploymentFail(err error) {
	entry := LogEntry{
		Origin:    "applikatoni",
		EntryType: DEPLOYMENT_FAIL,
		Message:   fmt.Sprintf("deployment_id=%d, err=%s", l.deployment.Id, err),
		Timestamp: time.Now(),
	}

	l.Log(entry)
}

func (l *DeploymentLogger) LogKillReceived() {
	entry := LogEntry{
		Origin:    "applikatoni",
		EntryType: KILL_RECEIVED,
		Message:   "deployment will be stopped after current stage",
		Timestamp: time.Now(),
	}

	l.Log(entry)
}
