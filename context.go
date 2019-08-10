package rup

import (
	"sync"
	"time"
)

// Context contains all information
type Context struct {
	// The request id
	ID string

	// From tells from who the message was
	// net.Addr.String()
	From string

	// MessageSize is the size of the message in amound of bytes
	// This can be used for tracking the recive process
	MessageSize uint64

	// ReciveSize is the size of what we currently have recived
	ReciveSize uint64

	// upcommingParts are parts for this message
	upcommingPartsWLock sync.RWMutex
	upcommingParts      map[uint64][]byte

	// buff is a small buffer that is counted up before sending it over the rec channel
	// Channels seem somewhat slow so we buffer some parts first before sending them
	// NOTE: probebly subject to change
	buffWLock sync.RWMutex
	buff      []byte

	// expectNext is the id of the next expected block
	// NOTE: probebly subject to change
	expectNext uint64

	// Stream is the channel whereover the library will send all message data
	// If there is no data is left (thus the message is completed) this channel will be closed,
	// This can be checked using:
	//   data, ok := <- c.Stream
	//   if !ok {
	//	   fmt.Println("No data left!, Message completed")
	//   }
	Stream chan []byte
}

// newReq creates a new s.req
func (s *Server) newReq(From, ID string, MessageSize uint64, startBytes []byte) {
	if ID == "" {
		return
	}

	startBytesLen := uint64(len(startBytes))
	requestDune := MessageSize <= startBytesLen

	// all messages start with 1 because 0 means there is no next message expected
	// that means that if this is the first (1) message than the next will be (2)
	expectNext := 2
	if requestDune {
		expectNext = 0
	}

	c := &Context{
		ID:             ID,
		From:           From,
		MessageSize:    MessageSize,
		ReciveSize:     startBytesLen,
		upcommingParts: map[uint64][]byte{},
		buff:           []byte{},
		expectNext:     uint64(expectNext),
		Stream:         make(chan []byte),
	}

	if s.Reciver != nil {
		go s.Reciver(c)
		c.Stream <- startBytes
	}

	// If this udp package contains the full message close the channel
	if requestDune {
		close(c.Stream)
		c = nil
		time.Sleep(time.Millisecond * 5)
		s.sendConfirm(From, ID, []uint64{1})
		return
	}

	s.reqsWLock.Lock()
	s.reqs[ID] = c
	s.reqsWLock.Unlock()
}
