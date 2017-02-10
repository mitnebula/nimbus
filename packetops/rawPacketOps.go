package packetops

import (
	log "github.com/Sirupsen/logrus"
	"net"
)

func SendRaw(
	conn *net.UDPConn,
	p *RawPacket,
) error {
	if len(p.Buf) > 1472 {
		log.Panic("packet too big! ", len(p.Buf), " > 1472")
	}
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
