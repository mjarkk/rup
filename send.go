package rup

import (
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// Send sends a message to the *server
func (s *Server) Send(to string, msg []byte) error {
	addr, err := net.ResolveUDPAddr("udp", to)
	if err != nil {
		return err
	}

	uuidV4, err := uuid.NewV4()
	if err != nil {
		return err
	}
	messageID := strings.Replace(uuidV4.String(), "-", "", -1)
	messageNum := uint64(1)

	meta := addMeta(metaT{
		start:      true,
		id:         messageID,
		length:     uint64(len(msg)),
		messageNum: messageNum,
	})

	firstAppendLen := s.BufferSize - len(meta)
	if len(msg) < firstAppendLen {
		firstAppendLen = len(msg)
	}
	firstAppend := msg[:firstAppendLen]
	msg = msg[firstAppendLen:]

	_, err = s.serv.WriteToUDP(append(meta, firstAppend...), addr)
	if err != nil {
		return err
	}

	for {
		if len(msg) == 0 {
			break
		}
		messageNum++
		time.Sleep(time.Microsecond * 1)

		meta = addMeta(metaT{
			id:         messageID,
			messageNum: messageNum,
		})

		sliceSize := s.BufferSize - len(meta)
		if len(msg) < sliceSize {
			sliceSize = len(msg)
		}

		toAdd := msg[:sliceSize]
		msg = msg[sliceSize:]

		_, err = s.serv.WriteToUDP(append(meta, toAdd...), addr)
		if err != nil {
			return err
		}
	}

	return nil
}

func addMeta(meta metaT) []byte {
	parts := []string{}
	add := func(s string) {
		parts = append(parts, s)
	}
	if val, ok := meta.id.(string); ok {
		add("i:" + val)
	}
	if val, ok := meta.start.(bool); ok && val == true {
		add("s")
	}
	if val, ok := meta.messageNum.(uint64); ok {
		add("n:" + strconv.Itoa(int(val)))
	}
	if val, ok := meta.length.(uint64); ok {
		add("l:" + strconv.Itoa(int(val)))
	}
	return append([]byte(strings.Join(parts, ",")), 0x00)
}
