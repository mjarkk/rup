package rup

import (
	"crypto/rand"
	"math/big"
	mrand "math/rand"
	"net"
	"strconv"
)

// genServer returns creates a *net.UDPConn
func genServer(addr string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	return net.ListenUDP("udp", udpAddr)
}

// randomPortNum returns a port number between 10000 and 65535
// These port numbers are usualy not used
func randomPortNum() string {
	return ":" + strconv.Itoa(mrand.Intn(65535-10000)+10000)
}

var seedRandIsCalled = false

// SeedRand seeds the math/rand package so the random numbers
// are "actually random" *1
// 1: random as in the value is diffrent every time a function from the math/rand package is called
func seedRand() error {
	if seedRandIsCalled {
		return nil
	}
	num, err := rand.Int(rand.Reader, big.NewInt(9223372036854775806)) // https://golang.org/ref/spec#Numeric_types
	if err != nil {
		return err
	}
	mrand.Seed(num.Int64())
	seedRandIsCalled = true
	return nil
}
