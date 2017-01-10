package main

import (
	"fmt"
	"net"
)

var rcvd receivedBytes
var recvd chan interface{}
var doneDecoding chan interface{}
var acks chan Packet

var done chan interface{}

func init() {
	acks = make(chan Packet, 100)
	recvd = make(chan interface{}, 100)
	rcvd = receivedBytes{b: make([]byte, 1500)}
	doneDecoding = make(chan interface{})

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
	pkt, fromAddr, err := RecvPacket(rcvConn)
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

	go ackSender(rcvConn)
	go handlePacket(rcvConn)

	go func() {
		fmt.Println("connected to ", fromAddr)

		// send first ack
		ack, _ := makeAck(pkt, done)
		acks <- ack
	}()

	go receive(rcvConn)

	<-done

	return nil
}

func receive(conn *net.UDPConn) error {
	for {
		Listen(conn, &rcvd)

		//recvd <- rcvd
		select {
		case recvd <- struct{}{}:
		default:
			fmt.Println("recvd channel full, dropping packet!")
		}

		<-doneDecoding
	}
}

func handlePacket(conn *net.UDPConn) {
	for _ = range recvd {
		rp := rcvd
		if rp.err != nil {
			fmt.Println("socket", rp.err)
			break
		}

		pkt, _, err := Decode(rp)
		doneDecoding <- struct{}{}
		if err != nil {
			fmt.Println("decode", err)
			continue
		}

		ack, _ := makeAck(pkt, done)

		//acks <- ack
		select {
		case acks <- ack:
		default:
			fmt.Println("ack channel full, dropping packet!")
		}
	}
	done <- struct{}{}
}

func makeAck(pkt Packet, done chan interface{}) (Packet, error) {
	pkt.RecvTime = Now()
	return pkt, nil
}

func ackSender(conn *net.UDPConn) {
	for a := range acks {
		err := SendAck(conn, a)
		if err != nil {
			fmt.Println(err)
			break
		}
	}
	done <- struct{}{}
}
