package main

import (
	"github.com/akshayknarayan/udp/packetops"
)

func setupReceiver() (syn packetops.Packet, counter *int64, offset int) {
	syn = &Packet{
		SeqNo:   42,
		VirtFid: 42,
		Echo:    Now(),
		Payload: "SYN",
	}

	counter = &recvCount
	offset = 14

	return
}
