package main

import (
	"time"
	"context"
	"sync"
	"log"
)

type TestModuleConfig struct {
	NewConnEvery time.Duration
	NewConnMax int64
	DelayInPayload time.Duration
	NewConnFunc func()
}

type TestModule struct {
	config *TestModuleConfig
}

func (m TestModule) Start(in, out chan *Stream, ctx context.Context, wg *sync.WaitGroup) error {
	relayer := NewRelayer()
	ctx, moduleCancel := context.WithCancel(context.Background())
	sig := make(chan interface{}, 0)

	handler := func(relay *Relayer, wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc) {
		defer Drain(relay.In.C(), wg)
		defer cancel()
		defer close(relay.Out.C())

		LOOP: for {
			select {
				case <- ctx.Done():
					break LOOP
				case payload, opened := <- relay.In.C():
					if ! opened {
						break LOOP
					}

					time.Sleep(m.config.DelayInPayload)

					select {
						case <- ctx.Done():
							break LOOP
						case relay.Out.C() <- payload:
					}
			}
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer moduleCancel()
		defer close(sig)

		ticker := time.NewTicker(m.config.NewConnEvery)
		defer ticker.Stop()

		log.Println("Conn loop started")
		LOOP: for i := int64(0) ; i < m.config.NewConnMax; i++ {
			select {
				case <- ctx.Done():
					log.Println("cancelling ctx in loop")
					break LOOP
				case <- ticker.C:
					log.Println("ticker received in loop")
					if f := m.config.NewConnFunc; f != nil {
						f()
					}

					select {
						case <- ctx.Done():
							log.Println("cancelling ctx in inner loop")
							break LOOP
						case sig <- struct{}{}:
							log.Println("payload relayed")
					}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer moduleCancel()
		defer close(out)

		MainLoop(in, out, relayer, sig, time.Duration(0), handler, nil, ctx, wg)
	}()

	return nil
}

func NewTestModule(config *TestModuleConfig) Module {
	return &TestModule{
		config: config,
	}
}
