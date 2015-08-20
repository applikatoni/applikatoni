package main

import (
	"fmt"
	"sync"
)

type KillRegistry struct {
	sync.RWMutex
	m map[int]chan struct{}
}

func NewKillRegistry() *KillRegistry {
	return &KillRegistry{
		m: make(map[int]chan struct{}),
	}
}

func (kr *KillRegistry) Add(deploymentId int) chan struct{} {
	c := make(chan struct{})

	kr.Lock()
	kr.m[deploymentId] = c
	kr.Unlock()

	return c
}

func (kr *KillRegistry) Remove(deploymentId int) {
	kr.Lock()
	if c, ok := kr.m[deploymentId]; ok {
		delete(kr.m, deploymentId)
		close(c)
	}
	kr.Unlock()
}

func (kr *KillRegistry) Get(deploymentId int) (chan struct{}, error) {
	kr.RLock()
	defer kr.RUnlock()

	c, ok := kr.m[deploymentId]
	if !ok {
		return nil, fmt.Errorf("no kill channel for deployment id %d found", deploymentId)
	}
	return c, nil
}
