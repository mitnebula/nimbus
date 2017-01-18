package main

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

type fakePacketOps struct{}

func (f fakePacketOps) SendPacket(conn *net.UDPConn, pkt Packet, size int) error {
	return nil
}

func (f fakePacketOps) SendAck(conn *net.UDPConn, pkt Packet) error {
	return nil
}

func (f fakePacketOps) SendRaw(
	conn *net.UDPConn,
	p *rawPacket,
) error {
	pkt, err := decode(p.buf)
	if err != nil {
		return err
	}

	if pkt.SeqNo != 0 || pkt.VirtFid != 0 {
		return fmt.Errorf("invalid ack sent %v", pkt)
	}

	return nil
}

func (f fakePacketOps) RecvPacket(conn *net.UDPConn) (Packet, *net.UDPAddr, error) {
	rcvd := receivedBytes{b: make([]byte, 1500)}
	f.Listen(conn, &rcvd)
	if rcvd.err != nil {
		return Packet{}, nil, rcvd.err
	}

	return Decode(&rcvd)
}

func (f fakePacketOps) Listen(
	conn *net.UDPConn,
	res *receivedBytes,
) {
	hdr := bytes.Repeat([]byte{0}, 22)
	pay := bytes.Repeat([]byte("a"), 28)
	in := append(hdr, pay...)
	copy(res.b, in)

	addr, err := net.ResolveUDPAddr("udp4", ":40000")
	res.err = err
	res.from = addr
}

func BenchmarkReceiveLatency(b *testing.B) {
	var f fakePacketOps
	lastTime := time.Now()
	for i := 0; i < b.N; i++ {
		err := doReceive(nil, f, &lastTime)
		if err != nil {
			b.Fatal(err)
		}
	}
}
