package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"
)

var rcvd *receivedBytes
var ackBuffer *rawPacket
var done chan interface{}

func init() {
	rcvd = &receivedBytes{
		b: make([]byte, 1500),
	}

	ackBuffer = &rawPacket{
		buf: make([]byte, 22),
	}

	done = make(chan interface{})
}

func Receiver(port string) error {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}

	rcvConn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return err
	}

	err = rcvConn.SetReadBuffer(SOCK_BUF_SIZE)
	if err != nil {
		fmt.Println("err setting sock rd buf sz", err)
	}

	// wait for first packet
	syn, fromAddr, err := RecvPacket(rcvConn)
	if err != nil {
		return err
	}

	// close and reopen
	rcvConn.Close()

	// dial connection to send ACKs
	rcvConn, err = net.DialUDP("udp4", addr, fromAddr)
	if err != nil {
		return err
	}

	go func() {
		fmt.Println("connected to ", fromAddr)

		// send first ack
		syn.RecvTime = Now()
		err := SendAck(rcvConn, syn)
		if err != nil {
			fmt.Println("synack", err)
		}
	}()

	//print on ctrl-c
	procExit := make(chan os.Signal, 1)
	signal.Notify(procExit, os.Interrupt)

	var r realPacketOps
	go receive(rcvConn, r)

	exitStats(procExit, done)

	return nil
}

func receive(conn *net.UDPConn, r packetOps) error {
	tot := 0
	lastTime := time.Now()
	for {
		err := doReceive(conn, r, &lastTime)
		if err != nil {
			fmt.Println(err)
			continue
		}

		tot += 1
	}
}

func doReceive(conn *net.UDPConn, r packetOps, lastTime *time.Time) error {
	r.Listen(conn, rcvd)
	*lastTime = time.Now()

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
