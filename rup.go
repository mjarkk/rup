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
	reqsWLock       sync.RWMutex
	reqs            map[string]*Context
	sendingWLock    sync.RWMutex
	sending         map[string]*sendHandelers
	rsaPrivKey      *rsa.PrivateKey
	rsaPubKeyString string
	BufferSize      int
}

// StartOptions are the options for the start function
type StartOptions struct {
	// Address is a *ustom address use
	// If empty the program will select one for you
	Address string

	// RSAPrivateKey can be set to use a self defined rsa private key
	RSAPrivKey *rsa.PrivateKey

	// BufferSize is the buffer size used for the udp pacakge
	// The later they are the less cpu intensive they are butt they will be less reliable
	// Default (2048) is used when BufferSize <= 0
	BufferSize int
}
