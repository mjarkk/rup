package rup

import (
	"bytes"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
)

var (
	// ErrReqTimedOut tells if the request is timed out
	ErrReqTimedOut = errors.New("Request timed out")
)

// sendConfirm sends a confirm message back to the sender
//
// Message layout:
// 63 {utf8 messageID} 00 {utf8 range end}
// 63 = "c"
func (s *Server) sendConfirm(to, messageID string, rangeEnd uint64) error {
	addr, err := net.ResolveUDPAddr("udp", to)
	if err != nil {
		return err
	}

	toSend := bytes.NewBufferString("c" + messageID)
	toSend.Write([]byte{0x00})
	toSend.WriteString(strconv.FormatUint(rangeEnd, 10))

	_, err = s.serv.WriteToUDP(toSend.Bytes(), addr)
	return err
}

// sendMissingPart sends a req to start sending again from a spesific part
//
// Message layout:
// 0x72 {utf8 messageID} 0x00 {utf8 from}
// 0x72 = "r"
// 0x2d = "-"
func (s *Server) sendMissingPart(to, messageID string, from uint64) error {
	addr, err := net.ResolveUDPAddr("udp", to)
	if err != nil {
		return err
	}

	toSend := bytes.NewBufferString("r" + messageID)
	toSend.Write([]byte{0x00})
	toSend.WriteString(strconv.FormatUint(from, 10))

	_, err = s.serv.WriteToUDP(toSend.Bytes(), addr)
	return err
}

func getAddr(addr string) (*net.UDPAddr, error) {
	return net.ResolveUDPAddr("udp", addr)
}

func genMsgID() (string, error) {
	uuidV4, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return strings.Replace(uuidV4.String(), "-", "", -1), nil
}

// Send sends a message
func (s *Server) Send(to string, msg []byte) error {

	// Fix for an anoying windows bug:
	// > https://github.com/golang/go/issues/15216
	if strings.HasPrefix(to, ":") {
		to = "127.0.0.1" + to
	}

	// Get the address
	addr, err := getAddr(to)
	if err != nil {
		return err
	}

	// Generate the message it's id
	messageID, err := genMsgID()
	if err != nil {
		return err
	}

	var firstPartLenght uint64

	// n is the total written bytes to the
	sendPart := func(from uint64) (n uint64, err error) {
		startMessage := from == 0
		if from >= uint64(len(msg)) {
			return 0, nil
		}
		time.Sleep(time.Microsecond)

		data := bytes.NewBufferString("d")
		metaData := metaT{
			id:   messageID,
			from: from,
		}
		if startMessage {
			metaData.start = true
			metaData.length = uint64(len(msg))
		}
		addMeta(data, metaData)

		sliceSize := s.BufferSize - uint64(data.Len())
		if uint64(len(msg))-from < sliceSize {
			sliceSize = uint64(len(msg)) - from
		}
		if startMessage {
			firstPartLenght = sliceSize
		}

		data.Write(msg[from : from+sliceSize])

		_, err = s.serv.WriteToUDP(data.Bytes(), addr)
		return sliceSize, err
	}

	// vars for the main for loop
	end := make(chan error)
	var (
		ended               bool
		confirmedRecivedTo  uint64
		currentPosition     uint64
		currentPositionLock sync.RWMutex
	)

	// Setting up all functions that need to run in a goroutine
	recivedFirstRequst := false
	s.sendingWLock.Lock()
	s.sending[messageID] = &sendHandelers{
		confirm: func(to uint64) {
			if to == firstPartLenght {
				recivedFirstRequst = true
			}
			if to > confirmedRecivedTo {
				confirmedRecivedTo = to
			}
		},
		req: func(from uint64) {
			if from < confirmedRecivedTo {
				return
			}
			confirmedRecivedTo = from
			currentPositionLock.Lock()
			currentPosition = from
			currentPositionLock.Unlock()
		},
	}
	s.sendingWLock.Unlock()

	go func() {
		checkNum := 0
		for {
			if ended {
				return
			}
			if checkNum == 5 {
				end <- ErrReqTimedOut
				break
			}
			checkNum++
			time.Sleep(time.Millisecond * 5)
			if recivedFirstRequst {
				break
			}
			currentPositionLock.Lock()
			currentPosition = 0
			currentPositionLock.Unlock()
		}
	}()

	go func() {
		// This is the main for loop that sends all parts
		for {
			if ended {
				return
			}
			if int(confirmedRecivedTo) >= len(msg) {
				// Req fininshed
				end <- nil
				break
			}

			currentPositionLock.Lock()
			n, err := sendPart(currentPosition)
			currentPosition += n
			currentPositionLock.Unlock()

			if err != nil {
				end <- err
				break
			}

			time.Sleep(time.Microsecond)
		}
	}()

	err = <-end
	ended = true
	return err
}

func addMeta(buff *bytes.Buffer, meta metaT) {
	items := map[rune]interface{}{
		'i': meta.id,
		's': meta.start,
		'f': meta.from,
		'l': meta.length,
	}

	writtenFirstPart := false
	for key, val := range items {
		keyValue := ""
		switch val.(type) {
		case string:
			keyValue = val.(string)
		case bool:
		case uint64:
			keyValue = strconv.Itoa(int(val.(uint64)))
		case int:
			keyValue = strconv.Itoa(val.(int))
		default:
			continue
		}
		if writtenFirstPart {
			buff.WriteRune(',')
		}
		buff.WriteRune(key)
		if keyValue != "" {
			buff.WriteString(":" + keyValue)
		}
		writtenFirstPart = true
	}

	buff.Write([]byte{0x00})
	return
}
