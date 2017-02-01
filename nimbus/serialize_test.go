package main

import (
	"bytes"
	"testing"
)

func TestEncode(t *testing.T) {
	p := Packet{VirtFid: 0, SeqNo: 0, Echo: 0, RecvTime: 0}
	buf, err := encode(p, 50)
	if err != nil {
		t.Error(err)
	}

	hdr := bytes.Repeat([]byte{0}, 22)
	pay := bytes.Repeat([]byte("a"), 28)
	expected := append(hdr, pay...)

	for i, v := range buf {
		if v != expected[i] {
			t.Error("mismatch", buf, expected)
		}
	}
}

func TestDecode(t *testing.T) {
	hdr := bytes.Repeat([]byte{0}, 22)
	pay := bytes.Repeat([]byte("a"), 28)
	in := append(hdr, pay...)

	expected := Packet{VirtFid: 0, SeqNo: 0, Echo: 0, RecvTime: 0}

	pkt, err := decode(in)
	if err != nil {
		t.Error(err)
	}

	if expected.VirtFid != pkt.VirtFid ||
		expected.SeqNo != pkt.SeqNo ||
		expected.Echo != pkt.Echo ||
		expected.RecvTime != pkt.RecvTime {
		t.Error("mismatch", pkt, expected)
	}
}
