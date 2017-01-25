package main

import (
	"fmt"
	"net"
)

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
		Echo:    Now(),
		Payload: "SYN",
	}

	err := r.SendAck(conn, syn)
	if err != nil {
		return err
	}

	return nil
}

func setupListeningSock(port string) (*net.UDPConn, *net.UDPAddr, error) {
	// set up syn listening socket
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%s", port))
	if err != nil {
		return nil, nil, err
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, nil, err
	}

	return conn, addr, nil
}

// wait for the SYN
// send the synack
func listenForSyn(
	conn *net.UDPConn,
	listenAddr *net.UDPAddr,
) (Packet, *net.UDPConn, error) {
	syn, fromAddr, err := r.RecvPacket(conn)
	if err != nil {
		return Packet{}, nil, err
	}

	// close and reopen
	conn.Close()

	// dial connection to send ACKs
	newConn, err := net.DialUDP("udp4", listenAddr, fromAddr)
	if err != nil {
		return Packet{}, nil, err
	}

	fmt.Println("connected to ", fromAddr)
	return syn, newConn, nil
}

func synAckExchange(conn *net.UDPConn, expSrc *net.UDPAddr, rtt_history chan int64) error {
	syn := Packet{
		Echo:    Now(),
		Payload: "SYN",
	}

	err := r.SendAck(conn, syn)
	if err != nil {
		return err
	}

	ack, srcAddr, err := r.RecvPacket(conn)
	if err != nil {
		return err
	}
	if srcAddr.String() != expSrc.String() {
		return fmt.Errorf("got packet from unexpected src: %s; expected %s", srcAddr, expSrc)
	}

	delay := Now() - ack.Echo
	rtt_history <- delay

	return nil
}
