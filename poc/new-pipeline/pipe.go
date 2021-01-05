package main

type Pipe struct {
	in chan *Stream
	out chan *Stream
}

func NewPipe() *Pipe {
	in, out := make(chan *Stream, 0), make(chan *Stream, 0)

//	go func(in, out make(chan *Stream, 0)) {
//		LOOP: for {
//			select {
//				case stream, opened := <- in:
//					if ! opened {
//						close(out)
//						break LOOP
//					}
//
//					out <- stream
//			}
//		}
//	}(in, out)

	return &Pipe{
		in: in,
		out: out,
	}
}

func (p Pipe) Chans() (chan *Stream, chan *Stream) {
	return p.in, p.out
}
