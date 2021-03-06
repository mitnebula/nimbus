package main

import (
	"github.com/akshayknarayan/udp/packetops"
)

type Packet struct {
	Echo     int64  // time at which packet was sent
	RecvTime int64  // time packet reached receiver
	Payload  string // payload (useless)
}

func (pkt *Packet) Encode(
	size int,
) (*packetops.RawPacket, error) {
	// ip header 20 bytes
	// udp header 8 bytes
	padTo := size - 28
	b, err := encode(*pkt, padTo)
	return &packetops.RawPacket{Buf: b}, err
}

func (pkt *Packet) Decode(
	r *packetops.RawPacket,
) error {
	p, err := decode(r.Buf)
	if err != nil {
		return err
	}

	pkt.Echo = p.Echo
	pkt.RecvTime = p.RecvTime

	return nil
}
