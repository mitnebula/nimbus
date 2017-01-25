package main

import (
	"net"
	"sync"
)

type Packet struct {
	Echo     int64  // time at which packet was sent
	RecvTime int64  // time packet reached receiver
	Payload  string // payload (useless)
}

func (pkt Packet) makeRaw(
	size int,
) (*rawPacket, error) {
	// ip header 20 bytes
	// udp header 8 bytes
	padTo := size - 28
	b, err := encode(pkt, padTo)
	return &rawPacket{buf: b}, err
}

type rawPacket struct {
	buf []byte
	mut sync.Mutex
}

type receivedBytes struct {
	b    []byte
	from *net.UDPAddr
	err  error
	mut  sync.Mutex
}

func Decode(
	r *receivedBytes,
) (Packet, *net.UDPAddr, error) {
	pkt, err := decode(r.b)
	if err != nil {
		return Packet{}, nil, err
	}

	return pkt, r.from, nil
}

type packetOps interface {
	SendPacket(conn *net.UDPConn, pkt Packet, size int) error
	SendAck(conn *net.UDPConn, pkt Packet) error
	SendRaw(conn *net.UDPConn, pkt *rawPacket) error

	RecvPacket(conn *net.UDPConn) (Packet, *net.UDPAddr, error)
	Listen(conn *net.UDPConn, res *receivedBytes)
}

type pktops struct{}

func (r pktops) SendPacket(
	conn *net.UDPConn,
	pkt Packet,
	size int,
) error {
	rp, err := pkt.makeRaw(size)

	_, err = conn.Write(rp.buf)
	if err != nil {
		return err
	}

	return nil
}

func (r pktops) SendAck(
	conn *net.UDPConn,
	pkt Packet,
) error {
	return r.SendPacket(conn, pkt, 28)
}

func (r pktops) SendRaw(
	conn *net.UDPConn,
	p *rawPacket,
) error {
	_, err := conn.Write(p.buf)
	return err
}

// helper function to receive and decode a packet
// use when decoding can be done synchronously
func (r pktops) RecvPacket(
	conn *net.UDPConn,
) (Packet, *net.UDPAddr, error) {
	rcvd := receivedBytes{b: make([]byte, 1500)}
	r.Listen(conn, &rcvd)
	if rcvd.err != nil {
		return Packet{}, nil, rcvd.err
	}

	return Decode(&rcvd)
}

func (r pktops) Listen(
	conn *net.UDPConn,
	res *receivedBytes,
) {
	_, addr, err := conn.ReadFromUDP(res.b)
	res.from = addr
	res.err = err
}
