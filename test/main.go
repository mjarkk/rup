package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mjarkk/rup"
)

var clientAddr = make(chan string)
var end = make(chan struct{})

func main() {
	go func() {
		err := createServer(true)
		if err != nil {
			fmt.Println("CreateServer error:", err)
			os.Exit(1)
		}
	}()
	go func() {
		err := createServer(false)
		if err != nil {
			fmt.Println("CreateServer error:", err)
			os.Exit(1)
		}
	}()
	<-end
}

func createServer(isSender bool) error {
	s, err := rup.Start(rup.StartOptions{})
	if err != nil {
		return err
	}

	if isSender {
		sendTo := <-clientAddr
		s.Send(sendTo, "testMessage", []byte("test"))
		time.Sleep(time.Second * 2)
		end <- struct{}{}
	} else {
		clientAddr <- s.ServAddr
	}

	return nil
}
