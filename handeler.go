package rup

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"time"
)

func (s *Server) handleReqParts(addr net.Addr, data []byte) {
	requests := bytes.Split(data, []byte{0x00})
	fmt.Printf("reqeust for %v parts\n", len(requests))
}

func (s *Server) handleConfirm(addr net.Addr, data []byte) {
	parts := bytes.Split(data, []byte{0x00})
	if len(parts) == 0 {
		return
	}
	sending, ok := s.sending[string(parts[0])]
	if !ok {
		return
	}
	ids := []uint64{}
	for _, id := range parts[1:] {
		out, err := strconv.ParseUint(string(id), 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, out)
	}
	sending.confirm(ids)
}

// handleReq handles all UDP requests
func (s *Server) handleReq(addr net.Addr, data []byte) {
	// Get the meta data from the string
	nullByte := bytes.IndexByte(data, 0x00)
	if nullByte == -1 {
		// If there is no null byte ignore this message
		return
	}
	meta := parseMeta(data[:nullByte])
	data = data[nullByte+1:]

	length, lengthOk := meta.length.(uint64)
	start, startOk := meta.start.(bool)
	ID, idOk := meta.id.(string)
	msgNum, msgNumOk := meta.messageNum.(uint64)
	if !idOk || ID == "" {
		// The message doesn't have a valid id
		return
	}
	if lengthOk && length > 0 && startOk && start {
		// Createa a new working part
		s.sendConfirm(addr.String(), ID, []uint64{1})
		s.newReq(addr.String(), ID, length, data)
		return
	} else if startOk && start {
		// Invalid start message
		return
	}

	if !msgNumOk || msgNum < 2 {
		// append messages only start with
		return
	}

	// This message is part of another bigger message
	value, ok := s.reqs[ID]
	if !ok {
		return
	}
	saddr := addr.String()
	if value.From != saddr {
		// The sender address changed
		value.From = saddr
	}

	if value.expectNext == 0 {
		// This request is already finished
		return
	} else if msgNum > value.expectNext {
		// add the data to the upcomming parts and die
		value.upcommingPartsWLock.Lock()
		value.upcommingParts[msgNum] = data
		value.upcommingPartsWLock.Unlock()
		return
	} else if msgNum < value.expectNext {
		// The id is lower that what wa already have in our cache
		return
	}
	value.addData(s, data)
}

func (c *Context) addData(s *Server, data []byte) {
	c.buffWLock.Lock()
	c.buff = append(c.buff, data...)
	c.buffWLock.Unlock()
	c.ReciveSize += uint64(len(data))
	eof := c.ReciveSize == c.MessageSize
	if eof {
		c.expectNext = 0
	} else {
		c.expectNext++
	}

	if eof || len(c.buff) > 5000 {
		// The buffer is full or this is the end of the message,
		// We can search the buffer to the channel
		if c.sendBuffer(s) {
			return
		}
	}

	time.Sleep(time.Microsecond * 2)
	// Check if there is a followup message in the upcommingParts map
	c.upcommingPartsWLock.Lock()
	data, ok := c.upcommingParts[c.expectNext]
	c.upcommingPartsWLock.Unlock()
	if ok {
		c.addData(s, data)
		return
	}
	time.Sleep(time.Millisecond)
}

// sendBuffer sends the current buffer over the stream channel
// The return boolean indicates if this was the last message send to the buffer
func (c *Context) sendBuffer(s *Server) bool {
	c.buffWLock.Lock()
	buffToSend := c.buff
	c.buff = []byte{}
	c.buffWLock.Unlock()
	c.Stream <- buffToSend
	if c.expectNext != 0 {
		return false
	}
	// WHAA it's the end kill the request
	// this is really dark
	close(c.Stream)
	delete(s.reqs, c.ID)
	c = nil
	return true
}

type metaT struct {
	start      interface{} // s (bool) is this the starting message
	id         interface{} // i (string) The message set id
	messageNum interface{} // n (uint64) The message it's nummber
	length     interface{} // l (uint64) The total message lenght
}

// parseMeta transforms s into a meta type
func parseMeta(s []byte) metaT {
	meta := metaT{}
	bindings := map[rune]struct {
		binding   *interface{}
		valueType string
	}{
		's': {&meta.start, "bool"},
		'i': {&meta.id, "string"},
		'n': {&meta.messageNum, "uint64"},
		'l': {&meta.length, "uint64"},
	}

	if len(s) == 0 {
		return meta
	}
	partsList := bytes.Split(s, []byte(","))

	for _, part := range partsList {
		keyAndValue := bytes.SplitN(part, []byte(":"), 2)
		var key rune
		var value []byte
		switch len(keyAndValue) {
		case 0:
			continue
		case 1:
			key = rune(keyAndValue[0][0])
			value = []byte{}
		default:
			key = rune(keyAndValue[0][0])
			value = keyAndValue[1]
		}
		binding, ok := bindings[rune(key)]
		if !ok {
			continue
		}
		switch binding.valueType {
		case "bool":
			*binding.binding = true
		case "string":
			*binding.binding = string(value)
		case "uint64":
			if len(value) != 0 {
				parsedVal, err := strconv.ParseUint(string(value), 10, 64)
				if err == nil {
					*binding.binding = parsedVal
				}
			}
		}
	}

	return meta
}
