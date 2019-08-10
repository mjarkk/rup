package rup

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// sendConfirm sends a confirm message back to the sender
//
// Message layout:
// 63 {utf8 messageID} 00 {utf8 msg number} 00 {utf8 msg number} 00 {utf8 msg number} 00 {utf8 msg number} ...
// 63 = "c"
func (s *Server) sendConfirm(to, messageID string, msgNumbers []uint64) error {
	addr, err := net.ResolveUDPAddr("udp", to)
	if err != nil {
		return err
	}

	parts := [][]byte{}
	for _, msgNum := range msgNumbers {
		parts = append(parts, []byte(strconv.FormatUint(msgNum, 10)))
	}
	msgToSend := bytes.Join(parts, []byte{0x00})                      // Add the confirmed ID's
	msgToSend = append(append([]byte(messageID), 0x00), msgToSend...) // Add the message ID
	msgToSend = append([]byte("c"), msgToSend...)                     // Add the message type
	_, err = s.serv.WriteToUDP(msgToSend, addr)
	return err
}

// sendReqParts sends a req part to the sender
// the ids a dubble slice because this allows for request ranges
// for example:
// []uint64{[]uint64{554, 700}}
//   requests for package 554 to 700
// []uint64{[]uint64{554}}
//   request only for package 554
// []uint64{[]uint64{400, 500}, []uint64{502, 600}}
//   request for the packges from 400 till 500 and from 502 till 600
// []uint64{[]uint64{554}, []uint64{557}}
//   request just 2 packages
//
// There are also ilegal request
// These won't cause an error but will have unsubspected behavior when executing
// []uint64{[]uint64{100, 200, 300}}
//   you cant request a range from 100 to 200 and then 300
//
// Message layout:
// 72 {utf8 messageID} 00 {utf8 msg number} 2d {utf8 msg number} 00 {utf8 msg number} 2d {utf8 msg number} 00 {utf8 msg number} 2d {utf8 msg number}...
// OR
// 72 {utf8 messageID} 00 {utf8 msg number} 00 {utf8 msg number} 00 {utf8 msg number}...
// 72 = "r"
// 2d = "-"
func (s *Server) sendReqParts(to, messageID string, msgNums [][]uint64) error {
	addr, err := net.ResolveUDPAddr("udp", to)
	if err != nil {
		return err
	}

	reqeustArr := [][]byte{}
	for _, part := range msgNums {
		toAdd := []string{}
		for _, num := range part {
			toAdd = append(toAdd, strconv.FormatUint(num, 10))
		}
		reqeustArr = append(reqeustArr, []byte(strings.Join(toAdd, "-")))
	}
	reqeust := bytes.Join(reqeustArr, []byte{0x00})
	reqeust = append(append([]byte(messageID), 0x00), reqeust...)
	reqeust = append([]byte("r"), reqeust...)

	_, err = s.serv.WriteToUDP(reqeust, addr)
	return err
}

type reLoopReq struct {
	start int
	end   int // use 0 or -1 to ignore
}

// Send sends a message
func (s *Server) Send(to string, msg []byte) error {
	addr, err := net.ResolveUDPAddr("udp", to)
	if err != nil {
		return err
	}

	// Generate the message it's id
	uuidV4, err := uuid.NewV4()
	if err != nil {
		return err
	}
	messageID := strings.Replace(uuidV4.String(), "-", "", -1)

	msgReqLoop := make(chan reLoopReq)
	defer func() {
		msgReqLoop = nil
		s.sendingWLock.Lock()
		delete(s.sending, messageID)
		s.sendingWLock.Unlock()
	}()

	leftOverMsgToSmall := errors.New("buffer size to small")
	sendPart := func(messageNum int, startMessage bool) error {
		if (messageNum)*s.BufferSize > len(msg) {
			return leftOverMsgToSmall
		}
		time.Sleep(time.Microsecond)

		var meta []byte
		if startMessage {
			meta = addMeta(metaT{
				start:      true,
				id:         messageID,
				length:     uint64(len(msg)),
				messageNum: messageNum,
			})
		} else {
			meta = addMeta(metaT{
				id:         messageID,
				messageNum: messageNum,
			})
		}
		meta = append([]byte("d"), meta...)

		sliceSize := s.BufferSize - len(meta)
		if len(msg)-(s.BufferSize*messageNum) < sliceSize {
			sliceSize = len(msg) - (s.BufferSize * messageNum)
		}

		toAdd := msg[s.BufferSize*messageNum : s.BufferSize*messageNum+sliceSize]

		_, err = s.serv.WriteToUDP(append(meta, toAdd...), addr)
		return err
	}

	sendPart(1, true)

	recivedFirstRequst := false
	s.sendingWLock.Lock()
	s.sending[messageID] = &sendHandelers{
		confirm: func(msgNumbers []uint64) {
			if len(msgNumbers) > 0 && msgNumbers[0] == 1 {
				recivedFirstRequst = true
			}
		},
		req: func(msgNumbers [][]uint64) {
			fmt.Println("recived req for msgNumbers:", msgNumbers)
		},
	}
	s.sendingWLock.Unlock()

	go func() {
		msgReqLoop <- reLoopReq{
			start: 1,
			end:   -1,
		}
		time.Sleep(time.Millisecond * 5)
		if !recivedFirstRequst {
			fmt.Println("resending..")
			sendPart(1, true)
		}
	}()

	for {
		reLoopData, ok := <-msgReqLoop
		if !ok {
			return nil
		}
		messageNum := reLoopData.start
	msgLoop:
		for {
			messageNum++
			if messageNum == reLoopData.end {
				continue
			}
			err := sendPart(messageNum, false)
			switch err {
			case nil:
				continue msgLoop
			case leftOverMsgToSmall:
				break msgLoop
			default:
				return err
			}
		}
	}
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
