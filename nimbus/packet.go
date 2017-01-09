package main

import (
	"bytes"
	"encoding/gob"
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

	pad := size - b.Len() - 54 // 52 = 20 bytes IP hdr + 8 bytes UDP hdr + 26 bytes gob encoding
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

// helper function to receive and decode a packet
// use when decoding can be done synchronously
func RecvPacket(
	conn *net.UDPConn,
) (Packet, *net.UDPAddr, error) {
	rcvd, err := Listen(conn)
	if err != nil {
		return Packet{}, nil, err
	}

	return Decode(rcvd)
}

func Listen(
	conn *net.UDPConn,
) (receivedBytes, error) {
	buf := make([]byte, 1500)
	read, addr, err := conn.ReadFromUDP(buf)
	if err != nil {
		return receivedBytes{}, err
	}

	if read == 0 {
		return receivedBytes{}, err
	}

	return receivedBytes{b: buf, from: addr}, nil
}

func Decode(
	rcvd receivedBytes,
) (Packet, *net.UDPAddr, error) {
	var pkt Packet
	buf := rcvd.b
	err := gob.NewDecoder(bytes.NewReader(buf)).Decode(&pkt)
	if err != nil {
		return Packet{}, nil, err
	}

	return pkt, rcvd.from, nil

}
