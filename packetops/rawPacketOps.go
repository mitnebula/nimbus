package packetops

import (
	"net"
)

func SendRaw(
	conn *net.UDPConn,
	p *RawPacket,
) error {
	_, err := conn.Write(p.Buf)
	return err
}

func Listen(
	conn *net.UDPConn,
	res *RawPacket,
) error {
	_, addr, err := conn.ReadFromUDP(res.Buf)
	res.From = addr
	return err
}
