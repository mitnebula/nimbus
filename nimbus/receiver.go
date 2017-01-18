package main

import (
	"fmt"
	"net"
	"time"
)

var rcvd *receivedBytes
var ackBuffer *rawPacket

func init() {
	rcvd = &receivedBytes{
		b: make([]byte, 1500),
	}

	ackBuffer = &rawPacket{
		buf: make([]byte, 22),
	}
}

func Client(ip string, port string) error {
	conn, _, err := setupClientSock(ip, port)
	if err != nil {
		return err
	}

	go receive(conn)
	err = sendSyn(conn)
	if err != nil {
		return err
	}

	return nil
}

func setupClientSock(ip string, port string) (*net.UDPConn, *net.UDPAddr, error) {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		return nil, nil, err
	}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return nil, nil, err
	}

	return conn, addr, nil
}

func sendSyn(conn *net.UDPConn) error {
	syn := Packet{
		SeqNo:   42,
		VirtFid: 42,
		Echo:    Now(),
		Payload: "SYN",
	}

	err := r.SendAck(conn, syn)
	if err != nil {
		return err
	}

	return nil
}

func receive(conn *net.UDPConn) error {
	lastTime := time.Now()
	for {
		err := doReceive(conn, r, &lastTime)
		if err != nil {
			fmt.Println(err)
			continue
		}

		recvCount++
	}
}

func doReceive(conn *net.UDPConn, r packetOps, lastTime *time.Time) error {
	r.Listen(conn, rcvd)
	*lastTime = time.Now()
	if rcvd.err != nil {
		fmt.Println(rcvd.err)
		return rcvd.err
	}

	// copy header (first 22 bytes) to ack
	ack := rcvd.b[:22]
	copy(ackBuffer.buf, ack)

	makeAck(ackBuffer, lastTime.UnixNano())

	err := r.SendRaw(conn, ackBuffer)
	if err != nil {
		return err
	}

	return nil
}

func makeAck(ack *rawPacket, recvTime int64) {
	// write recvTime to bytes 14 - 21
	buf := ack.buf[14:]
	encodeInt64(recvTime, buf)
}
