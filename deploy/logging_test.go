package deploy

import (
	"testing"
	"time"
)

func TestSubscribeDeploymentId(t *testing.T) {
	router := NewLogRouter()
	router.Start()
	defer router.Stop()

	testDone := make(chan struct{})

	router.Announce(8888)

	router.Subscribe(8888, func(ch <-chan LogEntry) {
		logEntry := <-ch
		if logEntry.DeploymentId != 8888 && logEntry.Message != "Hello" {
			t.Errorf("wrong deployment id. expected=%d, got=%d", 8888, logEntry.DeploymentId)
		}

		logEntry = <-ch
		if logEntry.DeploymentId != 8888 && logEntry.Message != "World" {
			t.Errorf("wrong deployment id. expected=%d, got=%d", 8888, logEntry.DeploymentId)
		}
		testDone <- struct{}{}
	})

	go func() {
		// goroutine that simulates two running deployments
		router.Broadcast <- LogEntry{Origin: "example.org", Message: "Hello", DeploymentId: 8888}
		router.Broadcast <- LogEntry{Origin: "example.org", Message: "Hello", DeploymentId: 3333}
		router.Broadcast <- LogEntry{Origin: "example.org", Message: "World", DeploymentId: 8888}
	}()

	<-testDone
}

func TestSubscribeWithoutAnnouncement(t *testing.T) {
	router := NewLogRouter()
	router.Start()
	defer router.Stop()

	err := router.Subscribe(8888, func(ch <-chan LogEntry) {})
	if err != ErrNoDeployment {
		t.Errorf("Subscribe did not return error")
	}
}

func TestSubscribeAll(t *testing.T) {
	router := NewLogRouter()
	router.Start()
	defer router.Stop()

	testDone := make(chan struct{})

	ids := []int{123, 456, 789}

	router.SubscribeAll(func(ch <-chan LogEntry) {
		for i := 0; i < len(ids); i++ {
			logEntry := <-ch
			if logEntry.DeploymentId != ids[i] {
				t.Errorf("wrong deployment id. expected=%d, got=%d", ids[i], logEntry.DeploymentId)
			}
		}
		testDone <- struct{}{}
	})

	go func() {
		// goroutine that simulates three running deployments
		for i := 0; i < len(ids); i++ {
			router.Broadcast <- LogEntry{Origin: "example.org", Message: "Bingo", DeploymentId: ids[i]}
		}
	}()

	<-testDone
}

func TestDoneBroadcasting(t *testing.T) {
	router := NewLogRouter()
	router.Start()
	defer router.Stop()

	testDone := make(chan struct{})

	router.Announce(8888)
	router.Announce(1111)

	router.Subscribe(8888, func(ch <-chan LogEntry) {
		for i := 0; i < 2; i++ {
			logEntry := <-ch
			if logEntry.DeploymentId != 8888 && logEntry.Message != "Bingo" {
				t.Errorf("wrong deployment id. expected=%d, got=%d", 8888, logEntry.DeploymentId)
			}
		}

		// ch should be closed now by the send to router.Done
		_, open := <-ch
		if open {
			t.Errorf("channel still open!")
		}
		testDone <- struct{}{}
	})

	go func() {
		// goroutine that simulates three running deployments
		router.Broadcast <- LogEntry{Origin: "example.org", Message: "one", DeploymentId: 8888}
		router.Broadcast <- LogEntry{Origin: "example.org", Message: "two", DeploymentId: 1111}
		router.Broadcast <- LogEntry{Origin: "example.org", Message: "two", DeploymentId: 8888}
		router.Done <- 8888
		router.Broadcast <- LogEntry{Origin: "example.org", Message: "two", DeploymentId: 1111}
		router.Done <- 1111
	}()

	<-testDone
}

func TestSubscribeAfterBroadcastStart(t *testing.T) {
	router := NewLogRouter()
	router.Start()
	defer router.Stop()

	testDone := make(chan struct{})

	router.Announce(8888)

	router.Broadcast <- LogEntry{Origin: "example.org", Message: "one", DeploymentId: 8888}
	router.Broadcast <- LogEntry{Origin: "example.org", Message: "two", DeploymentId: 8888}

	router.Subscribe(8888, func(ch <-chan LogEntry) {
		logEntry := <-ch
		if logEntry.DeploymentId != 8888 && logEntry.Message != "one" {
			t.Errorf("wrong deployment id. expected=%d, got=%d", 8888, logEntry.DeploymentId)
		}

		logEntry = <-ch
		if logEntry.DeploymentId != 8888 && logEntry.Message != "two" {
			t.Errorf("wrong deployment id. expected=%d, got=%d", 8888, logEntry.DeploymentId)
		}

		logEntry = <-ch
		if logEntry.DeploymentId != 8888 && logEntry.Message != "three" {
			t.Errorf("wrong deployment id. expected=%d, got=%d", 8888, logEntry.DeploymentId)
		}

		testDone <- struct{}{}
	})

	router.Broadcast <- LogEntry{Origin: "example.org", Message: "three", DeploymentId: 8888}
	router.Done <- 8888

	<-testDone
}

func TestSubscribeAfterBroadcastEnd(t *testing.T) {
	router := NewLogRouter()
	router.Start()
	defer router.Stop()

	router.Announce(8888)
	router.Broadcast <- LogEntry{Origin: "example.org", Message: "one", DeploymentId: 8888}
	router.Done <- 8888

	// Announce another deployment to give router some time (otherwise we have a deadlock here)
	router.Announce(9999)
	router.Done <- 9999

	err := router.Subscribe(8888, func(ch <-chan LogEntry) {})
	if err == nil {
		t.Errorf("Subscribe did not return an error")
	}
}

func TestRoutingTimeout(t *testing.T) {
	router := NewLogRouter()
	router.Start()
	defer router.Stop()

	testDone := make(chan struct{})

	router.Announce(8888)

	slowListener := func(ch <-chan LogEntry) {
		<-ch // Receive the first message

		time.Sleep(ListenerTimeout + 100*time.Millisecond)

		// ch should be closed now since we timed out
		_, open := <-ch
		if open {
			t.Errorf("channel still open after timeout!")
		}
		testDone <- struct{}{}
	}

	goodListener := func(ch <-chan LogEntry) {
		<-ch
		<-ch
		testDone <- struct{}{}
	}

	router.Subscribe(8888, slowListener) // times out
	router.Subscribe(8888, goodListener) // should receive both log entries

	go func() {
		router.Broadcast <- LogEntry{Origin: "example.org", Message: "one", DeploymentId: 8888}
		router.Broadcast <- LogEntry{Origin: "example.org", Message: "one", DeploymentId: 8888}
	}()

	<-testDone
	<-testDone
}

func TestRoutingAllTimeoutSubscriptions(t *testing.T) {
	router := NewLogRouter()
	router.Start()
	defer router.Stop()

	testDone := make(chan struct{})

	router.Announce(8888)

	// This test doesn't care about the channels being closed (since we already
	// tested this above). What's tested is the deletion of timed out
	// subscriptions when _every_ subscription times out (which lead to a
	// out-of-bounds panic).
	slowListener := func(ch <-chan LogEntry) {
		time.Sleep(ListenerTimeout*2 + 50*time.Millisecond)
		testDone <- struct{}{}
	}

	router.Subscribe(8888, slowListener)
	router.Subscribe(8888, slowListener)

	go func() {
		router.Broadcast <- LogEntry{Origin: "example.org", Message: "one", DeploymentId: 8888}
	}()

	<-testDone
	<-testDone
}
func TestRoutingBacklogTimeout(t *testing.T) {
	router := NewLogRouter()
	router.Start()
	defer router.Stop()

	testDone := make(chan struct{})

	router.Announce(8888)
	// This gets added to the backlog and routed first
	router.Broadcast <- LogEntry{Origin: "example.org", Message: "one", DeploymentId: 8888}

	slowListener := func(ch <-chan LogEntry) {
		time.Sleep(ListenerTimeout + 100*time.Millisecond)

		// ch should be closed now since we timed out
		_, open := <-ch
		if open {
			t.Errorf("channel still open after timeout when sending broadcast!")
		}
		testDone <- struct{}{}
	}

	goodListener := func(ch <-chan LogEntry) {
		<-ch
		<-ch
		testDone <- struct{}{}
	}

	router.Subscribe(8888, slowListener) // times out
	router.Subscribe(8888, goodListener) // should receive both log entries

	go func() {
		router.Broadcast <- LogEntry{Origin: "example.org", Message: "one", DeploymentId: 8888}
	}()

	<-testDone
	<-testDone
}
