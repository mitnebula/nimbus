package main

import (
	"fmt"
	"net"
)

type Packet struct {
	SeqNo    uint32 // packet sequence number
	VirtFid  uint16 // virtual flow packet is assigned to
	Echo     int64  // time at which packet was sent
	RecvTime int64  // time packet reached receiver
	Payload  string // payload (useless)
}

type receivedBytes struct {
	b    []byte
	from *net.UDPAddr
	err  error
}

func PrintPacket(pkt Packet) string {
	return fmt.Sprintf(
		"{seq %d vfid %d echo %d recv %d size %d}",
		pkt.SeqNo,
		pkt.VirtFid,
		pkt.Echo,
		pkt.RecvTime,
		len(pkt.Payload),
	)
}

func SendPacket(
	conn *net.UDPConn,
	pkt Packet,
	size int,
) error {
	// ip header 20 bytes
	// udp header 8 bytes
	padTo := size - 28
	buf, err := encode(pkt, padTo)

	_, err = conn.Write(buf)
	if err != nil {
		return err
	}

	return nil
}

func SendAck(
	conn *net.UDPConn,
	pkt Packet,
) error {
	return SendPacket(conn, pkt, 28)
}

// helper function to receive and decode a packet
// use when decoding can be done synchronously
func RecvPacket(
	conn *net.UDPConn,
) (Packet, *net.UDPAddr, error) {
	rcvd := receivedBytes{b: make([]byte, 1500)}
	Listen(conn, &rcvd)
	if rcvd.err != nil {
		return Packet{}, nil, rcvd.err
	}

	return Decode(rcvd)
}

func Listen(
	conn *net.UDPConn,
	res *receivedBytes,
) {
	_, addr, err := conn.ReadFromUDP(res.b)
	res.from = addr
	res.err = err
}

func Decode(
	r receivedBytes,
) (Packet, *net.UDPAddr, error) {
	pkt, err := decode(r.b)
	if err != nil {
		return Packet{}, nil, err
	}

	return pkt, r.from, nil
}
