package main

import (
	"fmt"
	"net"
	"sync"
)

type rawAck struct {
	buf []byte
	mut sync.Mutex
}

var rcvd []*receivedBytes
var recvd chan int // index of rcvd to read
var ackBuffer []*rawAck
var acks chan int // index of ackBuffer to read
var done chan interface{}

func init() {
	acks = make(chan int, 10)
	recvd = make(chan int, 10)

	rcvd = make([]*receivedBytes, 10)
	for i := 0; i < 10; i++ {
		rcvd[i] = &receivedBytes{
			b: make([]byte, 1500),
		}
	}

	ackBuffer = make([]*rawAck, 10)
	for i := 0; i < 10; i++ {
		ackBuffer[i] = &rawAck{
			buf: make([]byte, 22),
		}
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

	go ackSender(rcvConn)
	go handlePacket(rcvConn)

	go func() {
		fmt.Println("connected to ", fromAddr)

		// send first ack
		syn.RecvTime = Now()
		err := SendAck(rcvConn, syn)
		if err != nil {
			fmt.Println("synack", err)
		}
	}()

	go receive(rcvConn)

	<-done

	return nil
}

func receive(conn *net.UDPConn) error {
	curr := 0
	tot := 0
	for {
		rcvd[curr].mut.Lock()
		Listen(conn, rcvd[curr])

		select {
		case recvd <- curr:
		default:
			fmt.Println("recvd channel full, dropping packet!", curr, tot)
			panic(false)
		}

		curr = (curr + 1) % len(rcvd)
		tot += 1
	}
}

func handlePacket(conn *net.UDPConn) {
	for idx := range recvd {
		rp := rcvd[idx]
		if rp.err != nil {
			fmt.Println("socket", rp.err)
			break
		}

		// copy header (first 22 bytes) to ack
		hdr := rp.b[:22]

		ackBuffer[idx].mut.Lock()
		copy(ackBuffer[idx].buf, hdr)
		rp.mut.Unlock()

		// make ack without deserializing
		// directly write RecvTime to nimbus hdr
		makeAck(ackBuffer[idx])

		select {
		case acks <- idx:
		default:
			fmt.Println("ack channel full, dropping packet!")
		}
	}
	done <- struct{}{}
}

func makeAck(ack *rawAck) {
	recvTime := Now()

	// write recvTime to bytes 14 - 21
	buf := ack.buf[14:]
	encodeInt64(recvTime, buf)
}

func ackSender(conn *net.UDPConn) {
	for a := range acks {
		ack := ackBuffer[a]
		err := SendRaw(conn, ack)
		ack.mut.Unlock()
		if err != nil {
			fmt.Println(err)
			break
		}
	}
	done <- struct{}{}
}
