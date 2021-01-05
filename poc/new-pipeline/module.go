package main

import (
	"sync"
	"time"
	"context"
	"log"
)

type Module interface {
	Start(in, out chan *Stream, wg *sync.WaitGroup) error
}

type Module1 struct {
}

type ModuleRelayer struct {
	In *Stream
	Out *Stream
}

func (m *Module1) Start(in, out chan *Stream, wg *sync.WaitGroup) (error) {
	go func() {
		defer wg.Done()
		defer close(out)

		timeout := time.NewTicker(15 * time.Second)
		defer timeout.Stop()

		relayer := make(chan *ModuleRelayer, 0)
		streamSync := &sync.WaitGroup{}
		defer streamSync.Wait()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		incomingC := make(chan interface{}, 0)

		go func() {
			defer close(incomingC)

			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			log.Println("Starting incoming conn loop")
			LOOP: for {
				select {
					case <- ctx.Done():
						log.Println("Cancel called1, exiting")
						break LOOP
					case <- ticker.C:
						log.Println("Ticker received")
						incomingC <- struct{}{}
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

						ctx, cancel := context.WithCancel(ctx)
						streamSync.Add(2)
						go func() {
							defer streamSync.Done()
							defer close(relay.Out.C())
							defer cancel()

							out := relay.Out.C()

							LOOP: for i := 0; i < 10; i++ {
								time.Sleep(1 * time.Second)

								select {
									case <- ctx.Done():
										log.Printf("Canceled closed, stop working stream %s\n", relay.Out.Id())
										break LOOP
									case out <- []byte{byte(i + 48),}:
										log.Printf("Bytes sent to stream %s\n", relay.Out.Id())
								}
							}

							log.Printf("Sending done, closing stream %s\n", relay.Out.Id())
						}()

						go func() {
							defer cancel()
							in := relay.In.C()

							log.Printf("Starting listening on incoming stream %s\n", relay.In.Id())
							LOOP: for {
								select {
									case <- ctx.Done():
										log.Printf("Context canceled, draining in stream %s\n", relay.In.Id())
										go func() {
											defer streamSync.Done()

											for range in {}
											log.Printf("Draining done for stream %s\n", relay.In.Id())
										}()
										break LOOP
									case payload, opened := <- in:
										if ! opened {
											log.Printf("Incoming stream %s closed\n", relay.In.Id())
											streamSync.Done()
											break LOOP
										}
										log.Printf("Received bytes %s for stream %s\n", string(payload[:]), relay.In.Id())
								}
							}
						}()
				}
			}
		}()

		streamsIN := make([]*Stream, 0)
		streamsOUT := make([]*Stream, 0)

		LOOP: for {
			select {
				case <- ctx.Done():
					go func() {
						log.Println("Draning inc")
						for range incomingC {}
					}()
					break LOOP
				case _, opened := <- incomingC:
					timeout.Stop()
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

					if len(streamsIN) == 0 {
						streamsOUT = append(streamsOUT, outStream)
						continue
					}

					relay := &ModuleRelayer{
						Out: outStream,
						In: streamsIN[0],
					}

					streamsIN = streamsIN[1:]

					log.Println("Waiting to send the relay")
					select {
						case <- ctx.Done():
							log.Println("Context canceled, exiting")
							break LOOP
						case relayer <- relay:
							log.Println("Relay sent")
					}
				case inStream, opened := <- in:
					timeout.Stop()
					if ! opened {
						log.Println("inStream closed")
						go func() {
							log.Println("Draning inc")
							for range incomingC {}
						}()
						break LOOP
					}

					if len(streamsOUT) == 0 {
						streamsIN = append(streamsIN, inStream)
						continue
					}

					relay := &ModuleRelayer{
						Out: streamsOUT[0],
						In: inStream,
					}

					streamsOUT = streamsOUT[1:]

					log.Printf("Waiting to send the relay for stream %s\n", inStream.Id())
					select {
						case <- ctx.Done():
							log.Println("Context canceled, exiting")
							break LOOP
						case relayer <- relay:
							log.Println("Relay sent")
					}
				case <- timeout.C:
					log.Println("Connexion timeout has been reached")
					break LOOP
			}
		}
	}()

	return nil
}

func NewModule1() Module {
	return &Module1{}
}
