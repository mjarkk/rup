package rup

import (
	"bytes"
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

	// ReciveSize is the size of what we currently have recived (this does not count the upcommingParts)
	// If this is equal or more than MessageSize the message is completed
	//
	// This library doesn't check for data integrity thus the
	// ReciveSize can't be trusted to go never above the MessageSize
	ReciveSize uint64

	// upcommingParts are parts for this message
	upcommingPartsWLock sync.RWMutex
	upcommingParts      map[uint64][]byte

	// buff is a small buffer that is counted up before sending it over the rec channel
	// Channels seem somewhat slow so we buffer some parts first before sending them
	// NOTE: probebly subject to change
	buffWLock sync.RWMutex
	buff      *bytes.Buffer

	// Stream is the channel whereover the library will send all message data
	// If there is no data is left (thus the message is completed) this channel will be closed,
	// This can be checked using:
	//   data, ok := <- c.Stream
	//   if !ok {
	//	   fmt.Println("No data left!, Message completed")
	//   }
	Stream chan []byte

	// sendedReqForPart holds the number of the current requested part that is maybe missing
	sendedReqForPart uint64
}

// newReq creates a new s.req
func (s *Server) newReq(From, ID string, MessageSize uint64, startBytes []byte) *Context {
	if ID == "" {
		return nil
	}

	startBytesLen := uint64(len(startBytes))
	requestDune := MessageSize <= startBytesLen

	c := &Context{
		ID:             ID,
		From:           From,
		MessageSize:    MessageSize,
		ReciveSize:     startBytesLen,
		upcommingParts: map[uint64][]byte{},
		buff:           bytes.NewBuffer(nil),
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
		s.sendConfirm(From, ID, startBytesLen)
		return nil
	}

	s.reqsWLock.Lock()
	s.reqs[ID] = c
	s.reqsWLock.Unlock()
	return c
}
