package main

import (
	"fmt"
	"io/ioutil"
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
		dataToSend, err := ioutil.ReadFile("./testPhoto.jpg")
		// dataToSend, err := ioutil.ReadFile("./testData.txt")
		if err != nil {
			panic(err)
		}
		for i := 0; i < 1; i++ {
			s.Send(sendTo, dataToSend)
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
				// fmt.Println("recived:", len(data))
				data = append(data, newData...)
			}
			fmt.Println("EOF")
		}
		clientAddr <- s.ServAddr
	}

	fmt.Println("End!")
	return nil
}
