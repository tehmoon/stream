package main

import (
	"sync"
	"time"
	"context"
	"log"
	"net"
)

type Module interface {
	Start(in, out chan *Stream, ctx context.Context, wg *sync.WaitGroup) error
}

type Module1 struct {
	Addr string
}

func (m Module1) Start(in, out chan *Stream, ctx context.Context, wg *sync.WaitGroup) (error) {

	relayer := NewRelayer()
	ctx, moduleCancel := context.WithCancel(ctx)
	sig := make(chan interface{}, 0)

	conns := make(chan *net.TCPConn, 0)

	handlerIn := func(relay *Relayer, wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc) {
		in := relay.In.C()
		defer Drain(in, wg)
		defer moduleCancel()
		defer cancel()

		addr, _ := net.ResolveTCPAddr("tcp", m.Addr)
		conn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			log.Printf("Error dialing tcp: %s\n", err.Error())
			return
		}

		// todo: cleanup

		select {
			case <- ctx.Done():
				return
			case conns <- conn:
				log.Println("conn sent")
		}

		log.Printf("Starting listening on incoming stream %s\n", relay.In.Id())
		LOOP: for {
			select {
				case <- ctx.Done():
					log.Printf("Context canceled, draining in stream %s\n", relay.In.Id())
					break LOOP
				case payload, opened := <- in:
					if ! opened {
						log.Printf("Incoming stream %s closed\n", relay.In.Id())
						break LOOP
					}

					_, err := conn.Write(payload)
					if err != nil {
						log.Printf("Error writing to conn: %s\n", err.Error())
						break LOOP
					}
			}
		}
	}

	handlerOut := func(relay *Relayer, wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc) {
		defer wg.Done()
		defer moduleCancel()
		defer cancel()

		out := relay.Out.C()
		defer close(out)
		defer log.Println("Closing out stream")

		select {
			case <- ctx.Done():
				return
			case conn, opened := <- conns:
				if ! opened {
					return
				}
				LOOP: for {
					buff := make([]byte, 1)
					i, err := conn.Read(buff)
					if err != nil {
						break LOOP
					}
					select {
						case <- ctx.Done():
							conn.Close()
							break LOOP
						case out <- buff[:i]:
					}
				}
		}

		log.Printf("Sending done, closing stream %s\n", relay.Out.Id())
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(out)
		defer moduleCancel()
		defer close(sig)

		go func() {
			sig <- struct{}{}
		}()

		MainLoop(in, out, relayer, sig, 15 * time.Second, handlerIn, handlerOut, ctx, wg)
	}()

	return nil
}

func NewModule1(addr string ) Module {
	return &Module1{
		Addr: addr,
	}
}
