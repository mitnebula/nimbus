package main

import (
	"flag"
	"fmt"
)

const SOCK_BUF_SIZE = 5000 * 1500

var ip = flag.String("ip", "127.0.0.1", "IP to connect to")
var port = flag.String("port", "42424", "Port to connect to/listen on")
var mode = flag.String("mode", "sender", "sender or receiver")

func main() {
	flag.Parse()

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
