package rup

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	"net"

	"github.com/mjarkk/rup/crypt"
)

var (
// ReqRequestPublicKey is a request for the public key
// ReqRequestPublicKey = []byte("rp")

// SendPublicKey is the messate data for sending a public key to a client
// SendPublicKey = []byte("sp")

// Message is a normal message
// Message = []byte("m")

// GlobalLimits is a list of pre defined request limits for certen requests
// GlobalLimits = map[string]uint32{
// 	string(ReqRequestPublicKey): 4,
// 	string(SendPublicKey):       4,
// }

)

// Context is the meta data passes to the Rec function
type Context struct {
	Title string
	Msg   []byte
}

// Server is the global type for dealing with the server
type Server struct {
	serv            *net.UDPConn
	ServAddr        string
	recivers        map[string]func(*Context)
	currentParts    map[string][]byte // The current parts are the parts that are recived but do not (yet) have a goal in live
	workingParts    map[string]workPart
	rsaPrivKey      *rsa.PrivateKey
	rsaPubKeyString string
}

// Work part is a request that is still running
type workPart struct {
	expectNext string
	buff       bytes.Buffer
}

type req struct {
	buff []byte
	n    int
	addr net.Addr
	err  error
}

// StartOptions are the options for the start function
type StartOptions struct {
	// Address is a custom address use
	// If empty the program will select one for you
	Address string

	// RSAPrivateKey can be set to use a self defined rsa private key
	RSAPrivKey *rsa.PrivateKey
}

// Send sends a message to the *server
func (s *Server) Send(to string, title string, msg []byte) error {
	addr, err := net.ResolveUDPAddr("udp", to)
	if err != nil {
		return err
	}
	_, err = s.serv.WriteToUDP(msg, addr)
	return err
}

// Rec handles the recive of data
func (s *Server) Rec(title string, handeler func(*Context)) {
	s.recivers[title] = handeler
}

// listen starts listening on the UDP connection
func (s *Server) listen() {
	reqChan := make(chan req)
	// The actual listener
	go func() {
		for {
			buff := make([]byte, 2048)
			n, addr, err := s.serv.ReadFrom(buff)
			reqChan <- req{buff, n, addr, err}
		}
	}()
	// handle one request
	postHandeler := func() {
		for {
			req := <-reqChan
			if req.err != nil {
				return
			}
			data := req.buff[:req.n]
			s.handleReq(req.addr, data)
		}
	}
	// The handelers
	for i := 0; i < 10; i++ {
		go postHandeler()
	}
}

// handleReq handles 1 udp request
func (s *Server) handleReq(addr net.Addr, data []byte) {
	fmt.Println(addr.String(), string(data))
}

// Start creates a server instace
// When provided no address address it will take a ramdom poort on 0.0.0.0
// If there are more than 1 addreses defined the program will return an error
func Start(options StartOptions) (*Server, error) {
	server := Server{}

	if options.RSAPrivKey == nil {
		priv, err := crypt.RSAGenKey(4096)
		if err != nil {
			return nil, err
		}
		server.rsaPrivKey = priv
	} else {
		server.rsaPrivKey = options.RSAPrivKey
	}
	server.rsaPrivKey.Precompute()

	err := seedRand()
	if err != nil {
		return nil, err
	}

	switch len(options.Address) {
	case 0:
		loopItr := -1
		for {
			loopItr++
			port := randomPortNum()
			serv, err := genServer(port)
			if err != nil {
				if loopItr < 3 {
					continue
				}
				return nil, err
			}
			server.serv = serv
			server.ServAddr = port
			break
		}
	default:
		serv, err := genServer(options.Address)
		if err != nil {
			return nil, err
		}
		server.serv = serv
		server.ServAddr = options.Address
	}

	go server.listen()

	return &server, nil
}
