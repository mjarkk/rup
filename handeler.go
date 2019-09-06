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
	if len(data) < metaSize {
		return
	}

	meta := data[:metaSize]
	data = data[metaSize:]

	start, id, from, length := parseMeta(meta)

	if len(id) != 32 || length == 0 {
		// The message id have a valid id
		return
	}

	if start {
		// Createa a new working part
		s.sendConfirm(addr.String(), id, uint64(len(data)))
		c := s.newReq(addr.String(), id, length, data)
		if c == nil {
			return
		}
		c.waitForNewMessages(s)
		return
	}

	// This message is part of another bigger message
	s.reqsLock.Lock()
	context, ok := s.reqs[id]
	s.reqsLock.Unlock()
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

	context.upcommingPartsLock.Lock()
	context.upcommingParts[from] = data
	context.upcommingPartsLock.Unlock()
}

func (c *Context) waitForNewMessages(s *Server) {
	failCount := 0
	for {
		c.upcommingPartsLock.Lock()
		data, ok := c.upcommingParts[c.ReciveSize]
		if ok {
			delete(c.upcommingParts, c.ReciveSize)
			c.upcommingPartsLock.Unlock()
			failCount = 0

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

		c.upcommingPartsLock.Unlock()

		failCount++
		if failCount > 3 {
			s.sendMissingPart(c.From, c.ID, c.ReciveSize)
			failCount = 0
		}

		time.Sleep(time.Millisecond)
	}
}

// sendBuffer sends the current buffer over the stream channel
// The return boolean indicates if this was the last message send to the buffer
func (c *Context) sendBuffer(s *Server) bool {
	c.buffLock.Lock()
	buffToSend := c.buff.Bytes()
	c.buff = bytes.NewBuffer(nil)
	c.buffLock.Unlock()
	if c.Stream == nil {
		s.reqsLock.Lock()
		delete(s.reqs, c.ID)
		s.reqsLock.Unlock()
		c = nil
		return true
	}

	c.Stream <- buffToSend
	if c.ReciveSize < c.MessageSize {
		return false
	}

	// The request is completed it can now be removed
	close(c.Stream)
	s.reqsLock.Lock()
	delete(s.reqs, c.ID)
	s.reqsLock.Unlock()

	for i := 0; i < 3; i++ {
		time.Sleep(time.Millisecond * 2)
		s.sendConfirm(c.From, c.ID, c.MessageSize)
	}

	c = nil

	return true
}
