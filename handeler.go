package rup

import (
	"bytes"
	"net"
	"strconv"
	"time"
)

func (s *Server) handleMissingPart(addr net.Addr, data []byte) {
	parts := bytes.Split(data, []byte{0x00})
	if len(parts) != 2 {
		return
	}

	sending, ok := s.sending[string(parts[0])]
	if !ok {
		return
	}

	from, err := strconv.ParseUint(string(parts[1]), 10, 64)
	if err != nil {
		return
	}

	sending.req(from)
}

func (s *Server) handleConfirm(addr net.Addr, data []byte) {
	parts := bytes.Split(data, []byte{0x00})
	if len(parts) != 2 {
		return
	}
	sending, ok := s.sending[string(parts[0])]
	if !ok {
		return
	}

	end, err := strconv.ParseUint(string(parts[1]), 10, 64)
	if err != nil {
		return
	}
	sending.confirm(end)
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
	from, fromOk := meta.from.(uint64)
	if !idOk || ID == "" {
		// The message doesn't have a valid id
		return
	}
	if lengthOk && length > 0 && startOk && start {
		// Createa a new working part
		s.sendConfirm(addr.String(), ID, uint64(len(data)))
		c := s.newReq(addr.String(), ID, length, data)
		if c == nil {
			return
		}
		c.waitForNewMessages(s)
		return
	} else if startOk && start {
		// Invalid start message
		return
	}

	if !fromOk {
		return
	}

	// This message is part of another bigger message
	s.reqsWLock.Lock()
	context, ok := s.reqs[ID]
	s.reqsWLock.Unlock()
	if !ok {
		return
	}
	saddr := addr.String()
	if context.From != saddr {
		// The sender address changed
		context.From = saddr
	}

	if context.ReciveSize >= context.MessageSize || from < context.ReciveSize {
		// This request is already finished no need to handle this reqeust
		// Or the id is lower that what wa already have in our cache
		return
	}

	context.upcommingPartsWLock.Lock()
	context.upcommingParts[from] = data
	context.upcommingPartsWLock.Unlock()
}

func (c *Context) waitForNewMessages(s *Server) {
	failCount := 0
	for {
		data, ok := c.upcommingParts[c.ReciveSize]
		if ok {
			failCount = 0
			delete(c.upcommingParts, c.ReciveSize)
			c.buff.Write(data)
			c.ReciveSize += uint64(len(data))
			if c.ReciveSize >= c.MessageSize || c.buff.Len() > 5000 {
				// The buffer is full or this is the end of the message,
				// We can search the buffer to the channel
				if c.sendBuffer(s) {
					return
				}
			}
			continue
		}
		failCount++
		if failCount > 3 {
			s.sendMissingPart(c.From, c.ID, c.ReciveSize)
			failCount = 0
			time.Sleep(time.Millisecond * 2)
		}

		time.Sleep(time.Millisecond)
	}
}

// sendBuffer sends the current buffer over the stream channel
// The return boolean indicates if this was the last message send to the buffer
func (c *Context) sendBuffer(s *Server) bool {
	c.buffWLock.Lock()
	buffToSend := c.buff.Bytes()
	c.buff = bytes.NewBuffer(nil)
	c.buffWLock.Unlock()
	if c.Stream == nil {
		return true
	}
	c.Stream <- buffToSend
	if c.ReciveSize < c.MessageSize {
		return false
	}

	// The request is completed it can now be removed
	close(c.Stream)
	s.reqsWLock.Lock()
	delete(s.reqs, c.ID)
	s.reqsWLock.Unlock()
	c = nil
	return true
}

type metaT struct {
	start  interface{} // s (bool) is this the starting message
	id     interface{} // i (string) The message set id
	from   interface{} // n (uint64) The message it's nummber
	length interface{} // l (uint64) The total message lenght
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
		'f': {&meta.from, "uint64"},
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
