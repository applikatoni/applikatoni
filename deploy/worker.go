package deploy

import (
	"bufio"
	"io"
	"log"
	"strings"
	"time"

	"code.google.com/p/go.crypto/ssh"
	"github.com/flinc/applikatoni/models"
)

type ExecutionResult struct {
	timeTaken time.Duration
	origin    string
	err       error
	skipped   bool
}

type Worker struct {
	sshConfig *ssh.ClientConfig
	sshClient *ssh.Client
	host      *models.Host
	logger    *DeploymentLogger
	scripts   map[models.DeploymentStage]string // No ScriptTemplate here, we need the rendered one
}

func (w *Worker) Connect() error {
	client, err := newSSHClient(w.host.Name, w.sshConfig)
	if err != nil {
		return err
	}
	w.sshClient = client
	return nil
}

func (w *Worker) Close() error {
	if w.sshClient != nil {
		return w.sshClient.Close()
	}
	return nil
}

func (w *Worker) Execute(stage models.DeploymentStage) ExecutionResult {
	script, present := w.scripts[stage]
	if !present {
		return ExecutionResult{origin: w.host.Name, skipped: true}
	}

	start := time.Now()
	err := w.executeScript(script)
	timeTaken := time.Since(start)

	return ExecutionResult{origin: w.host.Name, err: err, timeTaken: timeTaken}
}

func (w *Worker) executeScript(script string) error {
	r := strings.NewReader(script)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		w.logCommandStart(line)

		err := w.runCommand(line)
		if err != nil {
			w.logCommandFail(line, err)
			return err
		}
		w.logCommandSuccess(line)
	}

	if err := scanner.Err(); err != nil {
		log.Println("Scanning lines of script failed", err)
		return err
	}

	return nil
}

func (w *Worker) runCommand(cmd string) error {
	session, err := w.sshClient.NewSession()
	if err != nil {
		log.Println("could not create new SSH session", err)
		return err
	}
	defer session.Close()

	sessionStderr, err := session.StderrPipe()
	if err != nil {
		log.Println("could not create new stderr pipe")
		return err
	}
	go w.logOutput(COMMAND_STDERR_OUTPUT, sessionStderr)

	sessionStdout, err := session.StdoutPipe()
	if err != nil {
		log.Println("could not create new stdout pipe")
		return err
	}
	go w.logOutput(COMMAND_STDOUT_OUTPUT, sessionStdout)

	if err = session.Start(cmd); err != nil {
		log.Println("Start failed")
		return err
	}

	return session.Wait()
}

func (w *Worker) logOutput(entryType LogEntryType, r io.Reader) {
	reader := bufio.NewReader(r)

	for {
		line, err := reader.ReadBytes('\n')
		if s := string(line); s != "" {
			entry := LogEntry{
				Origin:    w.host.Name,
				EntryType: entryType,
				Message:   s,
				Timestamp: time.Now(),
			}

			w.logger.Log(entry)
		}
		if err != nil {
			break
		}
	}
}

func (w *Worker) logCommandStart(cmd string) {
	w.logger.LogCmdStart(w.host.Name, cmd)
}

func (w *Worker) logCommandFail(cmd string, err error) {
	w.logger.LogCmdFail(w.host.Name, cmd, err)
}

func (w *Worker) logCommandSuccess(cmd string) {
	w.logger.LogCmdSuccess(w.host.Name, cmd)
}
