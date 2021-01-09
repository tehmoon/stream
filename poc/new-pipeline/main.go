package main

import (
	"sync"
	"time"
	"log"
	"runtime"
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

var modules = []Module{NewModule1(), NewModule1(), NewModule1(),}

func main() {
	wg := &sync.WaitGroup{}
	c1 := make(chan *Stream, 0)
	c2 := make(chan *Stream, 0)
	c3 := make(chan *Stream, 0)
	c4 := make(chan *Stream, 0)
	wg.Add(3)
	modules[0].Start(c1, c2, wg)
	modules[1].Start(c2, c3, wg)
	modules[2].Start(c3, c4, wg)

	after := time.After(25 * time.Second)
	LOOP: for {
		select {
			case <- after:
				log.Println("Calling close c1")
				close(c1)
				for stream := range c4 {
					log.Printf("Feedback received for stream %s\n", stream.Id())
					Drain(stream.C(), nil)
				}
				break LOOP
			case stream, opened := <- c4:
				if ! opened {
					close(c1)
					break LOOP
				}

				c1 <- stream
		}
	}

	log.Println("Waiting for work to finish")
	go func() {
		time.Sleep(100000*time.Second)
	}()
	wg.Wait()
	stacktrace := make([]byte, 8192)
	length := runtime.Stack(stacktrace, true)
  log.Println(string(stacktrace[:length]))
}
