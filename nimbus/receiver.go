package main

import (
	"fmt"
	"net"
)

var acks chan Packet
var recvd chan receivedBytes

var recv_seqnos map[int]int

func init() {
	acks = make(chan Packet, 10)
	recvd = make(chan receivedBytes, 10)
	recv_seqnos = make(map[int]int)
}

func Receiver(port string) error {
	done := make(chan interface{})

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
	go handlePacket(rcvConn, done)

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
		rcvd, err := Listen(conn)
		if err != nil {
			fmt.Println(err)
			continue
		}

		recvd <- rcvd
	}
}

func handlePacket(conn *net.UDPConn, done chan interface{}) {
	for rp := range recvd {
		pkt, fromAddr, err := Decode(rp)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if fromAddr.String() != conn.RemoteAddr().String() {
			// got packet from some other connection
			fmt.Println("got unknown packet", fromAddr, conn.RemoteAddr())
			return
		}

		ack, _ := makeAck(pkt, done)
		acks <- ack
	}
}

func makeAck(pkt Packet, done chan interface{}) (Packet, error) {
	err := error(nil)
	seq, ok := recv_seqnos[pkt.VirtFid]
	if seq != pkt.SeqNo-1 && ok {
		err = fmt.Errorf("drop %v %d %d", Now(), pkt.VirtFid, pkt.SeqNo-1)
		fmt.Println(err)
		done <- struct{}{}
	}

	recv_seqnos[pkt.VirtFid] = pkt.SeqNo
	pkt.RecvTime = Now()

	return pkt, err
}

func ackSender(conn *net.UDPConn) {
	for a := range acks {
		err := SendAck(conn, a)
		if err != nil {
			fmt.Println(err)
		}
	}
}
