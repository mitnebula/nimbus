package packetops

import (
	"net"
	"sync"
)

type RawPacket struct {
	Buf  []byte       // Bytes in the packet
	From *net.UDPAddr // if incoming, source address of packet
	Mut  sync.Mutex   // mutex
}

type Packet interface {
	Encode(pad int) (*RawPacket, error)
	Decode(r *RawPacket) error
}
