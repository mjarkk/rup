package rup

import (
	"crypto/rsa"
	"net"
	"sync"
)

// Server is the global type for dealing with the server
type Server struct {
	serv            *net.UDPConn
	ServAddr        string
	Reciver         func(*Context)
	reqsLock        sync.RWMutex
	reqs            map[string]*Context
	sendingLock     sync.RWMutex
	sending         map[string]*sendHandelers
	rsaPrivKey      *rsa.PrivateKey
	rsaPubKeyString string
	BufferSize      uint64
}

// StartOptions are the options for the start function
type StartOptions struct {
	// Address is a *ustom address use
	// If empty the program will select one for you
	Address string

	// BufferSize is the buffer size used for the udp pacakge
	// The later they are the less cpu intensive they are butt they will be less reliable
	// Default (8192) is used when BufferSize <= 0
	BufferSize uint64
}
