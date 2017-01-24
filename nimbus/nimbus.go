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
var runtime = flag.Duration("t", time.Duration(180)*time.Second, "runtime in seconds")
var numVirtualFlows = flag.Int("numFlows", 1, "number of virtual flows")
var estBandwidth = flag.Float64("estBandwidth", 24e6, "estimated bandwidth")

// TODO make a slow start-like startup
var initRate = flag.Float64("initRate", 10e6, "initial sending rate")

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

	est_bandwidth = *estBandwidth
	flowRate = *initRate

	xtcpData.numVirtualFlows = uint16(*numVirtualFlows)
	xtcpData.setXtcpCwnd(flowRate, time.Duration(150)*time.Millisecond)

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
