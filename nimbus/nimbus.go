package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
)

const SOCK_BUF_SIZE = 4000 * 1500

var ip = flag.String("ip", "127.0.0.1", "IP to connect to")
var port = flag.String("port", "42424", "Port to connect to/listen on")
var mode = flag.String("mode", "sender", "sender or receiver")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")

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

	if *mode == "sender" || *mode == "s" {
		err := Sender(*ip, *port)
		if err != nil {
			fmt.Println(err)
		}
	} else if *mode == "receiver" || *mode == "r" {
		err := Receiver(*port)
		if err != nil {
			fmt.Println(err)
		}
	}
}
