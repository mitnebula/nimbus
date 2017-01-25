package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

var ip = flag.String("ip", "127.0.0.1", "IP to connect to")
var port = flag.String("port", "42424", "Port to connect to/listen on")
var mode = flag.String("mode", "client", "server|client|sender|receiver")
var runtime = flag.Duration("t", time.Duration(180)*time.Second, "runtime in seconds")

var rate = flag.Float64("initRate", 10e6, "initial sending rate")
var msgSize = flag.Int("msgSizeBytes", 5840, "size of each message in bytes")

var r pktops

// overall statistics
var done chan interface{}
var sendCount int64
var recvCount int64
var startTime time.Time
var endTime time.Time

func main() {
	flag.Parse()
	fmt.Printf("%s:%s, %s\n", *ip, *port, *mode)

	//print on ctrl-c
	done = make(chan interface{})
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go exitStats(interrupt)

	flowRate = *rate

	startTime = time.Now()

	var err error
	if *mode == "server" {
		err = Server(*port)

		endTime = startTime.Add(*runtime)
	} else if *mode == "client" {
		err = Client(*ip, *port)

		endTime = startTime.Add(*runtime)
	} else if *mode == "sender" {
		err = Sender(*ip, *port)

		endTime = startTime.Add(*runtime)
	} else if *mode == "receiver" {
		err = Receiver(*port)
	}
	if err != nil {
		fmt.Println(err)
	}

	<-done
}

func exitStats(interrupt chan os.Signal) {
	<-interrupt
	doExit()
}

func doExit() {
	elapsed := time.Since(startTime)
	totalBytes := float64(sendCount * ONE_PACKET)
	fmt.Printf("Sent: throughput %.4vMbps; %v packets in %v\n",
		BpsToMbps(totalBytes/elapsed.Seconds()), sendCount, elapsed)
	totalBytes = float64(recvCount * ONE_PACKET)
	fmt.Printf("Received: throughput %.4vMbps; %v packets in %v\n",
		BpsToMbps(totalBytes/elapsed.Seconds()), recvCount, elapsed)
	done <- struct{}{}
	os.Exit(0)

}
