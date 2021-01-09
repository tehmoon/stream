package main

type Relayer struct {
	In *Stream
	Out *Stream
	V interface{}
}

func NewRelayer() (chan *Relayer) {
	return make(chan *Relayer)
}
