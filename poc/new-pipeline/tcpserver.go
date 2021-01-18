package main

import (
	"time"
	"sync"
	"context"
	"net"
	"log"
)

type TCPServer struct {
	Addr string
}

func (m TCPServer) Start(in, out chan *Stream, ctx context.Context, wg *sync.WaitGroup) (error) {
	relayer := NewRelayer()
	ctx, moduleCancel := context.WithCancel(context.Background())
	sig := make(chan interface{}, 0)

	handlerIn := func(relay *Relayer, wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc) {
		defer Drain(relay.In.C(), wg)
		defer cancel()

		conn := relay.V.(net.Conn)

		log.Println("tcp server handler in started")
		LOOP: for {
			select {
				case <- ctx.Done():
					break LOOP
				case payload, opened := <- relay.In.C():
					if ! opened {
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
		defer cancel()
		defer close(relay.Out.C())

		conn := relay.V.(net.Conn)
		defer conn.Close()

		log.Println("tcp server handler out started")
		LOOP: for {
			select {
				case <- ctx.Done():
					break LOOP
				default:
					buff := make([]byte, 1)
					i, err := conn.Read(buff)
					if err != nil {
						break LOOP
					}

					select {
						case relay.Out.C() <- buff[:i]:
						case <- ctx.Done():
							break LOOP
					}
			}
		}
	}

	// Handle err
	addr, _ := net.ResolveTCPAddr("tcp", m.Addr)
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Printf("Error listening tcp: %s\n", err.Error())
	}
	log.Println("Server is listening")

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(sig)
		defer moduleCancel()

		LOOP: for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Error accepting connection: %s\n", err.Error())
				break LOOP
			}

			log.Println("Connection accepted")
			select {
				case sig <- conn:
				case <- ctx.Done():
					break LOOP
			}
		}
	}()

	// Main loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(out)
		defer listener.Close()
		defer moduleCancel()

		MainLoop(in, out, relayer, sig, 3 * time.Second, handlerIn, handlerOut, ctx, wg)
	}()

	return nil
}

func NewTCPServer(addr string) Module {
	return &TCPServer{
		Addr: addr,
	}
}
