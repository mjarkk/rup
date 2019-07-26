package rup

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"net"
	"time"
)

// Send sends a message to the *server
func (s *Server) Send(to string, msg []byte) error {
	msgInfo := fmt.Sprintf("s,l:%v", len(msg))
	firstAppendLen := 1979 - len(msgInfo)
	if len(msg) < firstAppendLen {
		firstAppendLen = len(msg)
	}
	firstAppend := msg[:firstAppendLen]
	msg = msg[firstAppendLen:]

	// splittedMsg is every part of the message to send
	splittedMsg := [][]byte{
		bytes.Join([][]byte{[]byte(msgInfo), firstAppend}, []byte{0}),
	}

	for {
		if len(msg) == 0 {
			break
		}
		sliceSize := 1979
		if len(msg) < sliceSize {
			sliceSize = len(msg)
		}
		toAdd := msg[:sliceSize]
		toAddHash := sha1.Sum(toAdd)
		msg = msg[sliceSize:]

		splittedMsg[len(splittedMsg)-1] = bytes.Join([][]byte{splittedMsg[len(splittedMsg)-1], toAddHash[:]}, []byte{})

		splittedMsg = append(splittedMsg, bytes.Join([][]byte{[]byte{0}, toAdd}, []byte{}))
	}

	addr, err := net.ResolveUDPAddr("udp", to)
	if err != nil {
		return err
	}

	fmt.Println("Sending", len(splittedMsg), "parts")

	for _, part := range splittedMsg {
		_, err = s.serv.WriteToUDP(part, addr)
		if err != nil {
			return err
		}
		time.Sleep(time.Microsecond * 2)
	}
	return nil
}
