package main

import (
	"sync"
	"context"
)

/*
	Boot order
	- Register modules
	- Pass flags to modules (init)
	- Start modules

	Logic
	- Create prev/next pipes when starting module
	- Each side of the pipes sends a Stream
	- A stream is a channel and a metadata structure
	- The stream is intercepted by a core module that nests the metadata structures:
		- Module1 -> Routine1 -> Module2 -> Routine2 -> (back to Module1)
		- Module1 starts stream: out := utils.NewStream(metadata); next(out)
		- Routine1 receives stream, add metadata to structure, sends the stream to next module

	Message Passing routines
	- know their position in the pipeline
	- control metadata structures

	Metadata
	- Each stream builds a map[string][string]
	- Global module metadata are attached to the structure
	- templates engine after receiving the stream have builtins
		- find parameter in metadata (could be any module)
	- Structure is the following form:
		[module position][<"global" or "stream">.value]
		[2]["global.name"] = "stdin"
		[4]["stream.remote-ip"] = "127.0.0.1"
*/

func start(modules []Module, ctx context.Context, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}

	if wg == nil {
		wg = &sync.WaitGroup{}
		defer wg.Wait()
	}

	if len(modules) == 0 {
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	chans := []chan *Stream{make(chan *Stream, 0),}
	for i, m := range modules {
		chans = append(chans, make(chan *Stream, 0))
		m.Start(chans[i], chans[i+1], ctx, wg)
	}

	LOOP: for {
		select {
			case <- ctx.Done():
				close(chans[0])
				if wg != nil {
					wg.Add(1)
				}
				go Drain(chans[len(chans)-1], wg)
				break LOOP
			case stream, opened := <- chans[len(chans)-1]:
				if ! opened {
					close(chans[0])
					break LOOP
				}

				chans[0] <- stream
		}
	}
}

func main() {
	modules := []Module{NewModule1("127.0.0.1:12345"), NewTCPServer("127.0.0.1:12346"),}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go start(modules, ctx, wg)
	wg.Wait()

	PrintAllStacks()
}
