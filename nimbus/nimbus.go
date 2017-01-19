package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"time"
)

var ip = flag.String("ip", "127.0.0.1", "IP to connect to")
var port = flag.String("port", "42424", "Port to connect to/listen on")
var mode = flag.String("mode", "client", "server|client|sender|receiver")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
var runtime = flag.Int("t", 180, "runtime in seconds")

var r pktops

// overall statistics
var done chan interface{}
var sendCount int64
var recvCount int64
var startTime time.Time
var endTime time.Time

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Println("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Println("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	fmt.Printf("%s:%s, %s\n", *ip, *port, *mode)

	//print on ctrl-c
	done = make(chan interface{})
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go exitStats(interrupt)

	startTime = time.Now()

	var err error
	if *mode == "server" {
		err = Server(*port)
	} else if *mode == "client" {
		err = Client(*ip, *port)
	} else if *mode == "sender" {
		err = Sender(*ip, *port)

		rt := time.Duration(*runtime) * time.Second
		endTime = startTime.Add(rt)
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
	fmt.Printf("Sent: throughput %.4v; %v packets in %v\n", totalBytes/elapsed.Seconds(), sendCount, elapsed)
	totalBytes = float64(recvCount * ONE_PACKET)
	fmt.Printf("Received: throughput %.4v; %v packets in %v\n", totalBytes/elapsed.Seconds(), recvCount, elapsed)
	done <- struct{}{}
	os.Exit(0)

}
