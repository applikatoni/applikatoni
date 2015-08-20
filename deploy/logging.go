package deploy

import (
	"errors"
	"log"
	"sync"
	"time"
)

type LogEntryType string

type Listener func(<-chan LogEntry)

var ErrNoDeployment = errors.New("no deployment with this ID found")
var ErrTimeout = errors.New("sending to listener timed out")
var ListenerTimeout = 200 * time.Millisecond

const (
	COMMAND_STDOUT_OUTPUT LogEntryType = "COMMAND_STDOUT_OUTPUT"
	COMMAND_STDERR_OUTPUT LogEntryType = "COMMAND_STDERR_OUTPUT"
	COMMAND_START         LogEntryType = "COMMAND_START"
	COMMAND_FAIL          LogEntryType = "COMMAND_FAIL"
	COMMAND_SUCCESS       LogEntryType = "COMMAND_SUCCESS"
	STAGE_START           LogEntryType = "STAGE_START"
	STAGE_FAIL            LogEntryType = "STAGE_FAIL"
	STAGE_SUCCESS         LogEntryType = "STAGE_SUCCESS"
	STAGE_RESULT          LogEntryType = "STAGE_RESULT"
	DEPLOYMENT_START      LogEntryType = "DEPLOYMENT_START"
	DEPLOYMENT_SUCCESS    LogEntryType = "DEPLOYMENT_SUCCESS"
	DEPLOYMENT_FAIL       LogEntryType = "DEPLOYMENT_FAIL"
	KILL_RECEIVED         LogEntryType = "KILL_RECEIVED"
)

type LogEntry struct {
	Id           int          `json:"id"`
	Timestamp    time.Time    `json:"timestamp"`
	DeploymentId int          `json:"deployment_id"`
	Origin       string       `json:"origin"`
	EntryType    LogEntryType `json:"entry_type"`
	Message      string       `json:"message"`
}

type subscription struct {
	DeploymentId int
	Target       chan<- LogEntry
}

type LogRouter struct {
	// LogEntries on this channel will be routed to registered listeners
	Broadcast chan LogEntry

	// If a deployment is done, the DeploymentId needs to be sent on this channel
	// so the router can close the listeners target channels
	// Be sure to only sent here after all Broadcast sends
	Done chan int

	// Send a ListenRequest on this channel to register for LogEntries
	subscribe chan subscription

	stop chan struct{}

	// The mutex around `subscriptions`
	mu            *sync.Mutex
	subscriptions map[int][]subscription
	backlog       map[int][]LogEntry
}

func NewLogRouter() *LogRouter {
	return &LogRouter{
		Broadcast:     make(chan LogEntry),
		Done:          make(chan int),
		subscribe:     make(chan subscription),
		stop:          make(chan struct{}),
		mu:            &sync.Mutex{},
		subscriptions: make(map[int][]subscription),
		backlog:       make(map[int][]LogEntry),
	}
}

func (r *LogRouter) Start() {
	go func() {
		for {
			select {
			case sub := <-r.subscribe:
				err := r.sendBacklog(sub)
				if err != nil && err == ErrTimeout {
					log.Println("timeout when sending backlog, not adding subscription")
					close(sub.Target)
					continue
				}
				r.addSubscription(sub)
			case logEntry := <-r.Broadcast:
				r.saveLogEntry(logEntry)
				r.routeLogEntry(logEntry)
			case deploymentId := <-r.Done:
				r.deleteSubscriptions(deploymentId)
				r.deleteBacklog(deploymentId)
			case <-r.stop:
				return
			}
		}
	}()
}

func (r *LogRouter) Stop() {
	r.stop <- struct{}{}
}

func (r *LogRouter) Announce(deploymentId int) {
	r.mu.Lock()
	r.subscriptions[deploymentId] = []subscription{}
	r.mu.Unlock()
}

func (r *LogRouter) Subscribe(deploymentId int, l Listener) error {
	r.mu.Lock()
	_, ok := r.subscriptions[deploymentId]
	if !ok && deploymentId != 0 {
		r.mu.Unlock()
		return ErrNoDeployment
	}
	r.mu.Unlock()

	ch := make(chan LogEntry)
	r.subscribe <- subscription{Target: ch, DeploymentId: deploymentId}
	go l(ch)

	return nil
}

func (r *LogRouter) SubscribeAll(l Listener) {
	r.Subscribe(0, l)
}

func (r *LogRouter) addSubscription(sub subscription) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := sub.DeploymentId
	r.subscriptions[id] = append(r.subscriptions[id], sub)
}

func (r *LogRouter) saveLogEntry(logEntry LogEntry) {
	id := logEntry.DeploymentId
	r.backlog[id] = append(r.backlog[id], logEntry)
}

func (r *LogRouter) sendBacklog(sub subscription) error {
	for _, logEntry := range r.backlog[sub.DeploymentId] {
		err := r.sendWithTimeout(sub, logEntry)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *LogRouter) routeLogEntry(logEntry LogEntry) {
	id := logEntry.DeploymentId
	if id == 0 {
		log.Println("ERROR routing LogEntry: DeploymentId is 0")
		return
	}

	success := []subscription{}

	for _, sub := range r.subscriptions[id] {
		err := r.sendWithTimeout(sub, logEntry)
		if err != nil && err == ErrTimeout {
			log.Println("timeout when routing log entry, deleting subscription")
			close(sub.Target)
		} else {
			success = append(success, sub)
		}
	}

	r.subscriptions[id] = success

	for _, sub := range r.subscriptions[0] {
		sub.Target <- logEntry
	}
}

func (r *LogRouter) deleteSubscriptions(deploymentId int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, sub := range r.subscriptions[deploymentId] {
		close(sub.Target)
	}
	delete(r.subscriptions, deploymentId)
}

func (r *LogRouter) deleteBacklog(deploymentId int) {
	delete(r.backlog, deploymentId)
}

func (r *LogRouter) sendWithTimeout(s subscription, logEntry LogEntry) error {
	select {
	case s.Target <- logEntry:
	case <-time.After(ListenerTimeout):
		return ErrTimeout
	}
	return nil
}
