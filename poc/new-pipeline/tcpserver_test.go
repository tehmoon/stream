package main

import (
	"github.com/tehmoon/errors"
	"os"
	"testing"
	"context"
	"time"
	"net"
	"sync"
	"log"
	"io"
)
func TestTCPServerConnOneMax(t *testing.T) {
	config := &TestModuleConfig{
		NewConnEvery: time.Duration(1),
		NewConnMax: 1,
		DelayInPayload: time.Duration(0),
	}

	runTCPServerConnTest(config)
}

func TestTCPServerUntilTimeout(t *testing.T) {
	config := &TestModuleConfig{
		NewConnEvery: time.Duration(1),
		NewConnMax: (1<<63) - 1,
		DelayInPayload: time.Duration(0),
	}

	runTCPServerConnTest(config)
}

func TestTCPServerEveryOneSec(t *testing.T) {
	config := &TestModuleConfig{
		NewConnEvery: time.Duration(1 * time.Second),
		NewConnMax: (1<<63) - 1,
		DelayInPayload: time.Duration(0),
	}

	runTCPServerConnTest(config)
}

func runTCPServerConnTest(config *TestModuleConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	errC := make(chan error, 0)

	a, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:48574")

	modules := []Module{NewTestModule(config), NewTCPServer(a.String()),}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		start(modules, ctx, nil)
	}()

	s := "Hello World!"

	wg.Add(1)
	go func() {
		defer cancel()
		defer wg.Done()
		defer close(errC)

		LOOP: for {
			select {
				case <- ctx.Done():
					break LOOP
				default:
			}

			conn, err := net.DialTCP("tcp", nil, a)
			if err != nil {
				log.Printf("connection error: %s of type: %T\n", err, err)
				time.Sleep(500 * time.Millisecond)
				continue
			}
			defer conn.Close()

			_, err = conn.Write([]byte(s))
			if err != nil {
				if err != io.EOF {
					log.Printf("write error: %s of type: %T\n", err, err)
					time.Sleep(500 * time.Millisecond)
					continue
				}
			}

			buff := make([]byte, 12)
			i, err := io.ReadFull(conn, buff)
			if err != nil {
				log.Printf("read error: %s of type: %T\n", err, err)
				time.Sleep(500 * time.Millisecond)
				continue
			}
			buff = buff[:i]

			if string(buff[:]) != s {
				errC <- errors.New("Cannot assert read/write string")
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	for e := range errC {
		log.Printf("Error received: %s\n", e.Error())
		cancel()
		wg.Wait()
		os.Exit(2)
	}

	wg.Wait()
}
