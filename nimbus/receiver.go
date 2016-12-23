package main

import (
	"fmt"
	"net"
)

var recv_seqnos map[int]int

func init() {
	recv_seqnos = make(map[int]int)
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

	err = receive(rcvConn)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

func receive(conn *net.UDPConn) error {
	for {
		pkt, fromAddr, err := RecvPacket(conn)
		if err != nil {
			fmt.Println(err)
			continue
		}
		//fmt.Println("recvd", pkt.VirtFid, pkt.SeqNo)

		// second return value is error if drop detected
		ack, _ := handlePacket(pkt, fromAddr)

		err = SendAck(conn, fromAddr, ack)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func handlePacket(pkt Packet, fromAddr *net.UDPAddr) (Packet, error) {
	err := error(nil)
	seq, ok := recv_seqnos[pkt.VirtFid]
	if seq != pkt.SeqNo-1 && ok {
		err = fmt.Errorf("drop %v %d %d", Now(), pkt.VirtFid, pkt.SeqNo-1)
	}

	recv_seqnos[pkt.VirtFid] = pkt.SeqNo
	//fmt.Println(recv_seqnos)
	pkt.Rtt = Now() - pkt.Rtt

	return pkt, err
}
