package main

import (
	"sync"
	"fmt"
	"log"
	"context"
	"time"
)

/*
	- handler in
	- handler out
	- relayer
	- ticker (can be whatever kind of channel)
	- incomingC channel
*/

func Drain(v interface{}, wg *sync.WaitGroup) {
	switch c := v.(type) {
		case chan []byte:
			for range c {}
		case chan interface{}:
			for range c {}
		default:
			panic(fmt.Sprintf("Type %T is not valid to drain\n", c))
	}

	if wg != nil {
		wg.Done()
	}
}

func MainLoop(in, out chan *Stream, relayer chan *Relayer, sig chan interface{}, timeout time.Duration, handlerIn Handler, handlerOut Handler, ctx context.Context, wg *sync.WaitGroup) {
	relaySig := make(chan interface{}, 0)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(relaySig)

		wg.Add(1)
		defer Drain(sig, wg)

		log.Println("Starting incoming conn loop")
		LOOP: for {
			select {
				case <- ctx.Done():
					log.Println("Cancel called1, exiting")
					break LOOP
				case s, opened := <- sig:
					if ! opened {
						log.Println("Signal closed, meaning that there won't be any new conn")
						break LOOP
					}

					log.Printf("Ticker received\n")
					relaySig <- s
			}

			select {
				case <- ctx.Done():
					log.Println("Cancel called2, exiting")
					break LOOP
				case relay, opened := <- relayer:
					log.Println("Relay received")
					if ! opened {
						break LOOP
					}

					log.Printf("Starting work for stream in %s, out %s\n", relay.In.Id(), relay.Out.Id())

					// Wraps context
					ctx, cancel := context.WithCancel(ctx)
					if handlerIn != nil {
						wg.Add(1)
						go handlerIn(relay, wg, ctx, cancel)
					}

					if handlerOut != nil {
						wg.Add(1)
						go handlerOut(relay, wg, ctx, cancel)
					}
			}
		}
	}()

	ins := make([]*Stream, 0)
	outs := make([]*Stream, 0)
	vs := make([]interface{}, 0)

	if timeout <= 0 {
		log.Println("Negative timeout, set to max limit")
		timeout = (1<<63) - 1 // will tick in ~294 years
	}

	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	if timeout <= 0 {
		log.Println("Stoping timeout negative")
		ticker.Stop()
	}

	log.Printf("Timeout set to %v %v\n", timeout, ticker)
	LOOP: for {
		select {
			case <- ctx.Done():
				log.Println("123blih")
				wg.Add(1)
				go Drain(relaySig, wg)
				break LOOP
			case v, opened := <- relaySig:
				log.Println("124blih")
				ticker.Stop()
				if ! opened {
					break LOOP
				}

				outStream := NewStream(make(map[string]string))
				select {
					case out <- outStream:
						log.Printf("Out stream sent %s\n", outStream.Id())
					case <- ctx.Done():
						log.Printf("Context canceled, exiting stream %s\n", outStream.Id())
						break LOOP
				}

				if len(ins) == 0 {
					outs = append(outs, outStream)
					vs = append(vs, v)
					continue
				}

				relay := &Relayer{
					Out: outStream,
					In: ins[0],
					V: v,
				}

				ins = ins[1:]

				log.Println("Waiting to send the relay")
				select {
					case <- ctx.Done():
						log.Println("Context canceled, exiting")
						break LOOP
					case relayer <- relay:
						log.Println("Relay sent")
				}
			case inStream, opened := <- in:
				log.Println("received stream")
				if ! opened {
					log.Println("inStream closed")
					wg.Add(1)
					go Drain(relaySig, wg)
					break LOOP
				}

				if len(outs) == 0 {
					ins = append(ins, inStream)
					continue
				}

				relay := &Relayer{
					Out: outs[0],
					V: vs[0],
					In: inStream,
				}

				outs = outs[1:]
				vs = vs[1:]

				log.Printf("Waiting to send the relay for stream %s\n", inStream.Id())
				select {
					case <- ctx.Done():
						log.Println("Context canceled, exiting")
						break LOOP
					case relayer <- relay:
						log.Println("Relay sent")
				}
			case <- ticker.C:
				log.Println("Connexion timeout has been reached")
				break LOOP
		}
	}

	for _, stream := range ins {
		log.Printf("Draining stream in %s after main loop\n", stream.Id())
		wg.Add(1)
		go Drain(stream.C(), wg)
	}

	for _, stream := range outs {
		log.Printf("Closing stream out %s after main loop\n", stream.Id())
		close(stream.C())
	}
}

type Handler func(relay *Relayer, wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc)
