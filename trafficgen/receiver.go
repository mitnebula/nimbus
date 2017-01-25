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

func Receiver(port string) error {
	conn, listenAddr, err := setupListeningSock(port)
	if err != nil {
		return err
	}

	syn, conn, err := listenForSyn(conn, listenAddr)
	if err != nil {
		return err
	}

	go receive(conn)
	go func() {
		// send first ack
		syn.RecvTime = Now()
		err := r.SendAck(conn, syn)
		if err != nil {
			fmt.Println("synack", err)
		}
	}()

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

func doReceive(
	conn *net.UDPConn,
	r packetOps,
	lastTime *time.Time,
) error {
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
	// write recvTime to bytes 8 - 15
	buf := ack.buf[8:]
	encodeInt64(recvTime, buf)
}
