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
var mode = flag.String("mode", "client", "server or client")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")

// overall statistics
var done chan interface{}
var sendCount int64
var recvCount int64
var startTime time.Time

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

	if *mode == "server" {
		err := Server(*port)
		if err != nil {
			fmt.Println(err)
		}
	} else if *mode == "client" {
		err := Client(*ip, *port)
		if err != nil {
			fmt.Println(err)
		}
	}

	<-done
}

func exitStats(interrupt chan os.Signal) {
	<-interrupt
	elapsed := time.Since(startTime)
	totalBytes := float64(sendCount * ONE_PACKET)
	fmt.Printf("Sent: throughput %.4v; %v packets in %v\n", totalBytes/elapsed.Seconds(), sendCount, elapsed)
	totalBytes = float64(recvCount * ONE_PACKET)
	fmt.Printf("Received: throughput %.4v; %v packets in %v\n", totalBytes/elapsed.Seconds(), recvCount, elapsed)
	done <- struct{}{}
}
