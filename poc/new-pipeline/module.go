package main

import (
	"sync"
	"time"
	"context"
	"log"
	"crypto/rand"
)

type Module interface {
	Start(in, out chan *Stream, wg *sync.WaitGroup) error
}

type Module1 struct {
}

func (m *Module1) Start(in, out chan *Stream, wg *sync.WaitGroup) (error) {
	go func() {
		defer wg.Done()
		defer close(out)

		relayer := NewRelayer()
		handlerSync:= &sync.WaitGroup{}
		defer handlerSync.Wait()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		incomingC := make(chan interface{}, 0)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			defer close(incomingC)

			LOOP: for {
				select {
					case <- ticker.C:
						conn := make([]byte, 6)
						rand.Read(conn)
						incomingC <- conn
					case <- ctx.Done():
						break LOOP
				}
			}
		}()

		handlerIn := func(relay *Relayer, wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc) {
			defer cancel()
			in := relay.In.C()
			id := relay.In.Id()

			log.Printf("Starting listening on incoming stream %s\n", relay.In.Id())
			LOOP: for {
				select {
					case <- ctx.Done():
						log.Printf("Context canceled, draining in stream %s\n", relay.In.Id())
						go Drain(in, wg)
						break LOOP
					case payload, opened := <- in:
						if ! opened {
							log.Printf("Incoming stream %s closed\n", relay.In.Id())
							wg.Done()
							break LOOP
						}

						err := handlerIn(payload, id)
						if err != nil {
							log.Printf("Error in handlerIn function for stream %s err: %s\n", id, err.Error())
							go Drain(in, wg)
							break LOOP
						}
				}
			}
		}

		handlerOut := func(relay *Relayer, wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc) {
			defer wg.Done()
			defer cancel()

			out := relay.Out.C()
			defer close(out)

			LOOP: for i := 0; i < 10; i++ {
				time.Sleep(100 * time.Millisecond)

				select {
					case <- ctx.Done():
						log.Printf("Canceled closed, stop working stream %s\n", relay.Out.Id())
						break LOOP
					case out <- []byte{byte(i + 48),}:
						log.Printf("Bytes sent to stream %s\n", relay.Out.Id())
				}
			}

			log.Printf("Sending done, closing stream %s\n", relay.Out.Id())
		}

		MainLoop(in, out, relayer, incomingC, 15 * time.Second, handlerIn, handlerOut, ctx, handlerSync)
	}()

	return nil
}

func NewModule1() Module {
	return &Module1{}
}

func handlerIn(payload []byte, id string) error {
	log.Printf("Received bytes %s for stream %s\n", string(payload[:]), id)
	return nil
}
