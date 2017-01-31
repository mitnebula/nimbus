package main

import (
	"fmt"
	"net"
	"time"

	"github.mit.edu/hari/packetops"
)

var rcvd *packetops.RawPacket
var ackBuffer *packetops.RawPacket

func init() {
	rcvd = &packetops.RawPacket{
		Buf: make([]byte, 1500),
	}

	ackBuffer = &packetops.RawPacket{
		Buf: make([]byte, 1500),
	}
}

func Client(ip string, port string) error {
	conn, _, err := packetops.SetupClientSock(ip, port)
	if err != nil {
		return err
	}

	go receive(conn)
	syn := Packet{
		SeqNo:   42,
		VirtFid: 42,
		Echo:    Now(),
		Payload: "SYN",
	}
	err = packetops.SendSyn(conn, &syn)
	if err != nil {
		return err
	}

	return nil
}

func Receiver(port string) error {
	conn, listenAddr, err := packetops.SetupListeningSock(port)
	if err != nil {
		return err
	}

	syn := Packet{}
	conn, err = packetops.ListenForSyn(conn, listenAddr, &syn)
	if err != nil {
		return err
	}

	go receive(conn)
	go func() {
		// send first ack
		syn.RecvTime = Now()
		err := packetops.SendAck(conn, &syn)
		if err != nil {
			fmt.Println("synack", err)
		}
	}()

	return nil
}

func receive(conn *net.UDPConn) error {
	lastTime := time.Now()
	for {
		err := doReceive(conn, &lastTime)
		if err != nil {
			fmt.Println(err)
			continue
		}

		recvCount++
	}
}

func doReceive(
	conn *net.UDPConn,
	lastTime *time.Time,
) error {
	err := packetops.Listen(conn, rcvd)
	*lastTime = time.Now()
	if err != nil {
		fmt.Println(err)
		return err
	}

	// copy header (first 22 bytes) to ack
	ack := rcvd.Buf[:22]
	copy(ackBuffer.Buf, ack)

	makeAck(ackBuffer, lastTime.UnixNano())

	err = packetops.SendRaw(conn, ackBuffer)
	if err != nil {
		return err
	}

	return nil
}

func makeAck(ack *packetops.RawPacket, recvTime int64) {
	// write recvTime to bytes 14 - 21
	buf := ack.Buf[14:]
	encodeInt64(recvTime, buf)
}
