package main

import (
	"bytes"
	"testing"
)

func TestEncode(t *testing.T) {
	p := Packet{Echo: 0, RecvTime: 0}
	buf, err := encode(p, 50)
	if err != nil {
		t.Error(err)
	}

	hdr := bytes.Repeat([]byte{0}, 16)
	pay := bytes.Repeat([]byte("p"), 34)
	expected := append(hdr, pay...)

	for i, v := range buf {
		if v != expected[i] {
			t.Error("mismatch", buf, expected)
		}
	}
}

func TestDecode(t *testing.T) {
	hdr := bytes.Repeat([]byte{0}, 16)
	pay := bytes.Repeat([]byte("p"), 28)
	in := append(hdr, pay...)

	expected := Packet{Echo: 0, RecvTime: 0}

	pkt, err := decode(in)
	if err != nil {
		t.Error(err)
	}

	if expected.Echo != pkt.Echo ||
		expected.RecvTime != pkt.RecvTime {
		t.Error("mismatch", pkt, expected)
	}
}

func TestEncodeRecvTime(t *testing.T) {
	hdr := bytes.Repeat([]byte{0}, 16)
	ack := rawPacket{buf: hdr}
	makeAck(&ack, Now())
	n := Now()

	dec, err := decode(ack.buf)
	if err != nil {
		t.Error(err)
	}

	diff := n - dec.RecvTime
	if diff < 0 || diff > 100000 {
		t.Error("encoded incorrectly", dec.RecvTime, n, diff, dec, ack.buf)
	}
}

// benchmark how much time it takes to modify the packet
func BenchmarkRecvTime(b *testing.B) {
	hdr := bytes.Repeat([]byte{0}, 16)
	ack := rawPacket{buf: hdr}
	for i := 0; i < b.N; i++ {
		makeAck(&ack, Now())
	}
}
