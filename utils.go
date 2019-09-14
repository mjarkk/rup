package rup

import (
	"bytes"
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

// parseMeta meta parses the meta data
func parseMeta(data []byte) (start bool, id string, from, length uint64) {
	start = data[0] == 1

	id = string(data[1:33])

	from = uint64(data[33])
	from |= uint64(data[34]) << 8
	from |= uint64(data[35]) << 16
	from |= uint64(data[36]) << 24

	length = uint64(data[37])
	length |= uint64(data[38]) << 8
	length |= uint64(data[39]) << 16
	length |= uint64(data[40]) << 24

	return
}

// The default meta size
var metaSize = 41

// createMeta adds the meta data to a message
// the meta data is always a static size of 41 bytes
//
// Meta layout:
// 0-1   = start (1/0)
// 1-33  = message id (uuid without the "-")
// 33-37 = send from bytes ([from % 255, from / (255^1) % 255, from / (255^2) % 255, from / (255^3) % 255])
// 37-41 = message length ([length % 255, length / (255^1) % 255, length / (255^2) % 255, length / (255^3) % 255])
func createMeta(buff *bytes.Buffer, start bool, id string, from, length uint64) {
	addedBytes := 1

	if start {
		buff.WriteByte(0)
	} else {
		buff.WriteByte(1)
	}

	addedBytes += len(id)
	buff.WriteString(id)

	addedBytes += 8
	fromBytes := make([]byte, 4)
	fromBytes[0] = byte(from % 255)
	from /= 255
	fromBytes[1] = byte(from % 255)
	from /= 255
	fromBytes[2] = byte(from % 255)
	from /= 255
	fromBytes[3] = byte(from % 255)
	buff.Write(fromBytes)

	lengthBytes := make([]byte, 4)
	lengthBytes[0] = byte(length % 255)
	length /= 255
	lengthBytes[1] = byte(length % 255)
	length /= 255
	lengthBytes[2] = byte(length % 255)
	length /= 255
	lengthBytes[3] = byte(length % 255)
	buff.Write(lengthBytes)
}
