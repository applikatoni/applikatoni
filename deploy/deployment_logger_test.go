package deploy

import (
	"testing"
	"time"

	"github.com/applikatoni/applikatoni/models"
)

var testId int = 999999
var deployment *models.Deployment = &models.Deployment{Id: testId}

var testLogEntry LogEntry = LogEntry{
	Origin:    "example.org",
	Message:   "Hello World",
	EntryType: COMMAND_START,
	Timestamp: time.Now(),
}

func TestBroadcasting(t *testing.T) {
	router := NewLogRouter()
	router.Announce(testId)

	logger := NewDeploymentLogger(deployment, router)
	logger.BroadcastLogs()
	logger.Log(testLogEntry)

	entry := <-router.Broadcast

	if entry.DeploymentId != testId {
		t.Errorf("Logger did not add deployment id. expected=%d, got=%d", testId, entry.DeploymentId)
	}

	if entry.EntryType != testLogEntry.EntryType {
		t.Errorf("wrong entrytype. expected=%s, got=%s", testLogEntry.EntryType, entry.EntryType)
	}

	if entry.Origin != testLogEntry.Origin {
		t.Errorf("wrong origin. expected=%s, got=%s", testLogEntry.Origin, entry.Origin)
	}

	if entry.Message != testLogEntry.Message {
		t.Errorf("wrong message. expected=%s, got=%s", testLogEntry.Message, entry.Message)
	}
}

func TestFlush(t *testing.T) {
	router := NewLogRouter()
	router.Announce(testId)

	testDone := make(chan struct{})

	logger := NewDeploymentLogger(deployment, router)
	logger.BroadcastLogs()

	logger.Log(testLogEntry)
	logger.Log(testLogEntry)
	logger.Log(testLogEntry)

	go func() {
		// Simulating the router here by fetching the logs in a goroutine
		for i := 0; i < 3; i++ {
			entry := <-router.Broadcast

			if entry.DeploymentId != testId {
				t.Errorf("Logger did not add deployment id. expected=%d, got=%d", testId, entry.DeploymentId)
			}

			if entry.EntryType != COMMAND_START {
				t.Errorf("wrong entrytype. expected=%s, got=%s", COMMAND_START, entry.EntryType)
			}

			if entry.Origin != "example.org" {
				t.Errorf("wrong origin. expected=%s, got=%s", "example.org", entry.Origin)
			}

			if entry.Message != "Hello World" {
				t.Errorf("wrong message. expected=%s, got=%s", "whoami", entry.Message)
			}
		}

		doneId := <-router.Done
		if doneId != testId {
			t.Errorf("Flushing did not send the correct deployment id. expected=%d, got=%d", testId, doneId)
		}

		testDone <- struct{}{}
	}()

	// This sends the deployment id to the router and makes sure all logs are delivered
	logger.Flush()

	<-testDone
}

func TestLogCmdStart(t *testing.T) {
	router := NewLogRouter()
	router.Announce(testId)

	logger := NewDeploymentLogger(deployment, router)
	logger.BroadcastLogs()
	logger.LogCmdStart("example.org", "whoami")

	entry := <-router.Broadcast

	if entry.DeploymentId != testId {
		t.Errorf("wrong DeploymentId. expected=%d, got=%d", testId, entry.DeploymentId)
	}

	if entry.EntryType != COMMAND_START {
		t.Errorf("wrong entrytype. expected=%s, got=%s", COMMAND_START, entry.EntryType)
	}

	if entry.Origin != "example.org" {
		t.Errorf("wrong origin. expected=%s, got=%s", "example.org", entry.Origin)
	}

	if entry.Message != "whoami" {
		t.Errorf("wrong message. expected=%s, got=%s", "whoami", entry.Message)
	}
}
