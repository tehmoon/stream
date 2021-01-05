package main

import (
	"crypto/rand"
	"fmt"
)

type Stream struct {
	c chan []byte
	m []map[string]string
	id []byte
}

func NewStream(metadata map[string]string) (*Stream) {
	id := make([]byte, 6)
	//TODO fix
	rand.Read(id)
	return &Stream{
		c: make(chan []byte, 0),
		m: []map[string]string{metadata,},
		id: id,
	}
}

func (s Stream) Id() string {
	return fmt.Sprintf("%x", s.id)
}

func (s Stream) C() (chan []byte) {
	return s.c
}

/*
2021/01/05 05:32:02 Calling close c1
2021/01/05 05:32:02 inStream closed
2021/01/05 05:32:02 Context canceled, draining in stream
2021/01/05 05:32:02 Cancel called, exiting
2021/01/05 05:32:02 Sending bytes...
2021/01/05 05:32:03 Canceled closed, stop working
2021/01/05 05:32:03 Sending done, closing
2021/01/05 05:32:03 Draining done
*/
