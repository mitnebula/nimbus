package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
)

type Packet struct {
	SeqNo    int    // packet sequence number
	VirtFid  int    // virtual flow packet is assigned to
	Echo     int64  // time at which packet was sent
	RecvTime int64  // time packet reached receiver
	Payload  string // payload (useless)
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
	var b bytes.Buffer

	enc := gob.NewEncoder(&b)
	err := enc.Encode(pkt)
	if err != nil {
		return err
	}

	pad := size - b.Len() - 40
	pkt.Payload = MakeBytes(pad)
	err = enc.Encode(pkt)
	if err != nil {
		return err
	}

	_, err = conn.Write(b.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func SendAck(
	conn *net.UDPConn,
	pkt Packet,
) error {
	var b bytes.Buffer

	err := gob.NewEncoder(&b).Encode(pkt)
	if err != nil {
		return err
	}

	_, err = conn.Write(b.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func RecvPacket(
	conn *net.UDPConn,
) (Packet, *net.UDPAddr, error) {
	buf := make([]byte, 1500)
	var pkt Packet

	read, addr, err := conn.ReadFromUDP(buf)
	if err != nil {
		return Packet{}, nil, err
	}

	if read == 0 {
		return Packet{}, nil, fmt.Errorf("read %d bytes instead of full packet", read)
	}

	err = gob.NewDecoder(bytes.NewReader(buf)).Decode(&pkt)
	if err != nil {
		return Packet{}, nil, err
	}

	return pkt, addr, nil
}
