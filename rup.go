package rup

import (
	"bytes"
	"crypto/rsa"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// Server is the global type for dealing with the server
type Server struct {
	serv            *net.UDPConn
	ServAddr        string
	Reciver         func(*Context)
	currentParts    currentParts // The recived parts that do not yet have a destination
	workingParts    workingParts // Working parts as in "parts that work", these are buffers with actual data
	rsaPrivKey      *rsa.PrivateKey
	rsaPubKeyString string
}

type workingParts struct {
	lock  sync.RWMutex
	parts []*Context
}

type currentParts struct {
	syncReqReRun bool
	syncRunning  bool
	m            sync.Map
}

type addToCurrentParts struct {
	data []byte
	hash string
	next string
}

type currentPart struct {
	data  []byte // data is the data of this part
	count uint16 // the amound of times this is recived
	next  string
}

// Context contains all information
type Context struct {
	// MessageSize is the size of the message in amound of bytes
	// This can be used for tracking the recive process
	MessageSize int64

	// ReciveSize is the size of what we currently have recived
	ReciveSize int64

	// expectNext is the hash of the next expected message to append on this one
	// If this is empty the message is completed and the channel can be closed
	// Only for internal use
	expectNext string

	// Stream is the channel whereover the library will send all message data
	// If there is no data is left (thus the message is completed) this channel will be closed,
	// This can be checked using:
	//   data, ok := <- c.Stream
	//   if !ok {
	//	   fmt.Println("No data left!, Message completed")
	//   }
	Stream chan []byte
}

type addToWorkingPart struct {
	nextHash, dataHash string
	data               []byte
}

// publishContext sends a *Context object to the user there handeler
func (s *Server) publishContext(c *Context, startData []byte) error {
	if s.Reciver == nil {
		return errors.New("No reciver defined, code will deadlock without one")
	}
	s.workingParts.lock.Lock()
	s.workingParts.parts = append(s.workingParts.parts, c)
	s.workingParts.lock.Unlock()
	go s.Reciver(c)
	c.Stream <- startData
	return nil
}

// new creates a new working part
func (w *workingParts) new(s *Server, data []byte, totalSize int64, next string) {
	err := s.publishContext(&Context{
		MessageSize: totalSize,
		expectNext:  next,
		Stream:      make(chan []byte),
	}, data)

	if err != nil {
		// It's useless to sync if s.publishContext returned an error
		return
	}

	if len(next) != 0 {
		s.currentParts.sync(s)
	}
}

func (w *workingParts) update(s *Server, update addToWorkingPart) {
	toRemove := []string{}

	w.lock.Lock()
	for _, part := range w.parts {
		if part.expectNext == "" || part.expectNext != update.dataHash {
			continue
		}

		// We found a matching part, add it to the buffer
		part.expectNext = update.nextHash
		part.Stream <- update.data
		// fmt.Println("Working part:", update.dataHash, "Buff-len:", w.parts[i].buff.Len())

		if update.nextHash == "" {
			close(part.Stream)
			fmt.Println("working parts completed")
		}

		toRemove = append(toRemove, update.dataHash)
	}
	w.lock.Unlock()

	if len(toRemove) == 0 {
		return
	}

	for _, hash := range toRemove {
		s.currentParts.remove(hash)
	}
	s.currentParts.sync(s)
}

func (c *currentParts) add(s *Server, add addToCurrentParts) {
	// Check if there is already a part with this request it's data
	rawData, ok := c.m.Load(add.hash)
	var data currentPart
	if !ok {
		data.data = add.data
		data.next = add.next
	} else {
		data = rawData.(currentPart)
	}
	data.count++
	c.m.Store(add.hash, data)

	go func() {
		c.m.Range(func(rawKey, rawData interface{}) bool {
			data := rawData.(currentPart)
			if data.next != add.hash {
				return true
			}
			data.next = add.next
			data.data = bytes.Join([][]byte{data.data, add.data}, []byte{})
			c.m.Store(rawKey, data)
			return false
		})

		c.sync(s)
		time.Sleep(time.Second * 3)
		c.remove(add.hash)
	}()
}

func (c *currentParts) remove(hash string) {
	rawData, ok := c.m.Load(hash)
	if !ok {
		return
	}
	data := rawData.(currentPart)
	if data.count <= 1 {
		c.m.Delete(hash)
		return
	}
	data.count--
	c.m.Store(hash, data)
}

var syncCount uint32

func (c *currentParts) sync(s *Server, force ...bool) {
	if c.syncRunning && len(force) == 0 {
		c.syncReqReRun = true
		return
	}
	c.syncRunning = true

	syncCount++
	if syncCount%10 == 0 {
		fmt.Printf("ran a sync #%v\n", syncCount)
	}

	c.m.Range(func(rawKey, rawData interface{}) bool {
		data := rawData.(currentPart)
		s.workingParts.update(s, addToWorkingPart{data.next, rawKey.(string), data.data})
		return true
	})

	// time.Sleep(time.Millisecond)
	// time.Sleep(time.Microsecond * 50)
	if c.syncReqReRun {
		c.syncReqReRun = false
		c.sync(s, true)
		return
	}
	c.syncRunning = false
}

// StartOptions are the options for the start function
type StartOptions struct {
	// Address is a custom address use
	// If empty the program will select one for you
	Address string

	// RSAPrivateKey can be set to use a self defined rsa private key
	RSAPrivKey *rsa.PrivateKey
}
