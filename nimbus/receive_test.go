package main

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

type fakePacketOps struct{}

func (f fakePacketOps) Listen(
	conn *net.UDPConn,
	res *receivedBytes,
) {
	hdr := bytes.Repeat([]byte{0}, 22)
	pay := bytes.Repeat([]byte("a"), 28)
	in := append(hdr, pay...)
	copy(res.b, in)
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
