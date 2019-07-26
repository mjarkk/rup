package rup

import "net"

// Start creates a server instace
// When provided no address address it will take a ramdom poort on 0.0.0.0
// If there are more than 1 addresess defined the program will return an error
func Start(options StartOptions) (*Server, error) {
	server := Server{
		workingParts: workingParts{
			parts: []*Context{},
		},
	}

	// // Uncomment the data underhere later
	//
	// if options.RSAPrivKey == nil {
	// 	priv, err := crypt.RSAGenKey(4096)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	server.rsaPrivKey = priv
	// } else {
	// 	server.rsaPrivKey = options.RSAPrivKey
	// }
	// server.rsaPrivKey.Precompute()

	// err := seedRand()
	// if err != nil {
	// 	return nil, err
	// }

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

	server.listen()

	return &server, nil
}

// listen starts listening on the UDP connection
func (s *Server) listen() {
	type handelerT struct {
		data []byte
		addr net.Addr
	}
	handeler := make(chan handelerT)
	for i := 0; i < 5; i++ {
		go func() {
			for {
				buff := make([]byte, 2048)
				n, addr, err := s.serv.ReadFrom(buff)
				if err != nil {
					return
				}
				data := buff[:n]
				handeler <- handelerT{data, addr}
			}
		}()
	}
	for i := 0; i < 10000; i++ {
		go func() {
			for {
				r := <-handeler
				s.handleReq(r.addr, r.data)
			}
		}()
	}
}
