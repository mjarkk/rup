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

	partsDataLenght := s.BufferSize - uint64(metaSize) - 1

	// n is the total written bytes to the
	sendPart := func(from uint64) error {
		startMessage := from == 0
		if from >= uint64(len(msg)) {
			return nil
		}
		time.Sleep(time.Microsecond)

		data := bytes.NewBufferString("d")
		createMeta(data, startMessage, messageID, from, uint64(len(msg)))

		sliceSize := partsDataLenght
		if uint64(len(msg))-from < sliceSize {
			sliceSize = uint64(len(msg)) - from
		}
		data.Write(msg[from : from+sliceSize])

		_, err = s.serv.WriteToUDP(data.Bytes(), addr)
		return err
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
	s.sendingLock.Lock()
	s.sending[messageID] = &sendHandelers{
		confirm: func(to uint64) {
			if to == partsDataLenght {
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
		},
	}
	s.sendingLock.Unlock()

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
			go func(currentPosition uint64) {
				if ended {
					return
				}
				if confirmedRecivedTo > currentPosition {
					currentPosition = confirmedRecivedTo
				}

				if int(currentPosition) >= len(msg) {
					// Req fininshed
					end <- nil
					return
				}

				// fmt.Printf("Sending part: %%%d of the %%100\n", int(math.Round(float64(100)/float64(len(msg))*float64(currentPosition))))

				err := sendPart(currentPosition)
				if err != nil {
					end <- err
				}
			}(currentPosition)

			currentPositionLock.Lock()
			currentPosition += partsDataLenght
			currentPositionLock.Unlock()
		}
	}()

	err = <-end
	ended = true
	return err
}
