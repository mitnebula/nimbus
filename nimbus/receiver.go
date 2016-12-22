package main

import (
	"fmt"
	"net"
)

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

	err = handleRecv(rcvConn)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

func handleRecv(conn *net.UDPConn) error {
	seqnos := make(map[int]int)
	for {
		pkt, fromAddr, err := RecvPacket(conn)
		if err != nil {
			fmt.Println(err)
			continue
		}

		//fmt.Println("recvd", pkt.VirtFid, pkt.SeqNo)
		seq, ok := seqnos[pkt.VirtFid]
		if seq != pkt.SeqNo-1 && ok {
			fmt.Println("drop", Now(), pkt.VirtFid, pkt.SeqNo-1)
			panic("pkt drop")
		}
		seqnos[pkt.VirtFid] = pkt.SeqNo

		pkt.Rtt = Now() - pkt.Rtt

		err = SendAck(conn, fromAddr, pkt)
		if err != nil {
			fmt.Println(err)
		}
	}
}
