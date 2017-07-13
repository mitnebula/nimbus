package main

import (
	"github.com/akshayknarayan/udp/packetops"
)

func setupReceiver() (syn packetops.Packet, counter *int64, offset int) {
	syn = &Packet{
		Echo:    Now(),
		Payload: "SYN",
	}

	counter = &recvCount
	offset = 8

	return
}
