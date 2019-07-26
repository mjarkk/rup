package rup

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"net"
	"strconv"
	"strings"
)

var c = 0

// handleReq handles all UDP requests
func (s *Server) handleReq(addr net.Addr, data []byte) {

	c++
	if c%4000 == 0 {
		fmt.Println("Recived", c, "requests")
	}

	// Check if there is a followup message
	var followupMsg []byte
	if len(data) == 2000 {
		// This message has a followup message
		slicePart := len(data) - 20
		followupMsg = data[slicePart:]
		data = data[:slicePart]
	}
	followupMsgS := fmt.Sprintf("%x", followupMsg)

	// Get the meta data from the string
	nullByte := bytes.IndexByte(data, 0)
	if nullByte == -1 {
		// If there is no null byte ignore this message
		return
	}
	meta := parseMeta(string(data[:nullByte]))
	data = data[nullByte+1:]

	// This is the start of a new message
	if meta.start {
		s.workingParts.new(s, data, meta.lenght, followupMsgS)
		return
	}

	// Add this data part to the currentParts and wait for a removal
	s.currentParts.add(s, addToCurrentParts{data, fmt.Sprintf("%x", sha1.Sum(data)), followupMsgS})
}

type metaT struct {
	start  bool  // Is this the first message
	lenght int64 // The total message lenght
}

// parseMeta transforms s into a meta type
func parseMeta(s string) metaT {
	meta := metaT{}
	if len(s) == 0 {
		return meta
	}
	partsList := strings.Split(s, ",")
	parts := map[rune]string{}
	for _, part := range partsList {
		keyAndValue := strings.SplitN(part, ":", 2)
		key := ""
		value := ""
		switch len(keyAndValue) {
		case 0:
			continue
		case 1:
			key = keyAndValue[0]
		default:
			key = keyAndValue[0]
			value = keyAndValue[1]
		}
		parts[rune(key[0])] = value
	}
	if _, ok := parts['s']; ok {
		meta.start = true
	}
	if val, ok := parts['l']; ok && val != "" {
		size, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			meta.lenght = size
		}
	}
	return meta
}
