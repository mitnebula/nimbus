package main

import (
	"github.mit.edu/hari/packetops"
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
