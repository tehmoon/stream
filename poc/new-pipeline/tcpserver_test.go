package main

import (
	"testing"
	"context"
	"time"
	"net"
	"sync"
)

func Test1(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	config := &TestModuleConfig{
		NewConnEvery: time.Duration(1),
		NewConnMax: (1 << 63) - 1,
		DelayInPayload: time.Duration(0),
	}

	a, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:48574")
	modules := []Module{NewTestModule(config), NewTCPServer(a.String()),}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go start(modules, ctx, wg)

	wg.Add(1)
	go func() {
		defer cancel()
		conn, err := 
	}()

	wg.Wait()
}
