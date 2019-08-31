package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/mjarkk/rup"
)

var clientAddr chan string
var end chan struct{}

func init() {
	end = make(chan struct{})
	clientAddr = make(chan string)
}

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
	time.Sleep(time.Millisecond * 500)
	fmt.Println("End!")
}

func createServer(isSender bool) error {
	s, err := rup.Start(rup.StartOptions{})
	if err != nil {
		return err
	}

	if isSender {
		sendTo := <-clientAddr
		// dataToSend, err := ioutil.ReadFile("./testPhoto.jpg")
		// dataToSend, err := ioutil.ReadFile("./testData.txt")
		dataToSend, err := ioutil.ReadFile("./largeTestData.txt")
		if err != nil {
			panic(err)
		}
		for i := 0; i < 1; i++ {
			s.Send(sendTo, dataToSend)
			fmt.Println("SEND END")
			time.Sleep(time.Millisecond * 250)
		}
		end <- struct{}{}
	} else {
		s.Reciver = func(c *rup.Context) {
			data := []byte{}
			for {
				newData, ok := <-c.Stream
				if !ok {
					break
				}
				data = append(data, newData...)
			}
			fmt.Printf("EOF: %x\n", sha1.Sum(data))
			end <- struct{}{}
		}
		clientAddr <- s.ServAddr
	}
	return nil
}
