package receiver

import (
	"fmt"
	"net"
	"time"

	"github.mit.edu/hari/nimbus-cc/packetops"
)

var rcvd *packetops.RawPacket
var ackBuffer *packetops.RawPacket
var RecvCount *int64
var headerOffset int

func init() {
	rcvd = &packetops.RawPacket{
		Buf: make([]byte, 1500),
	}

	ackBuffer = &packetops.RawPacket{
		Buf: make([]byte, 1500),
	}
}

func Client(ip string, port string, syn packetops.Packet, rCount *int64, hdrOff int) error {
	RecvCount = rCount
	headerOffset = hdrOff

	conn, _, err := packetops.SetupClientSock(ip, port)
	if err != nil {
		return err
	}

	go receive(conn)
	err = packetops.SendSyn(conn, syn)
	if err != nil {
		return err
	}

	return nil
}

func Receiver(port string, syn packetops.Packet, rCount *int64, hdrOff int) error {
	RecvCount = rCount
	headerOffset = hdrOff

	conn, listenAddr, err := packetops.SetupListeningSock(port)
	if err != nil {
		return err
	}

	conn, err = packetops.ListenForSyn(conn, listenAddr, syn)
	if err != nil {
		return err
	}

	go receive(conn)
	go func() {
		// send first ack
		now := time.Now().UnixNano()
		rawSyn, err := syn.Encode(0)
		if err != nil {
			fmt.Println(err)
			return
		}

		makeAck(rawSyn, now)
		err = packetops.SendRaw(conn, rawSyn)
		if err != nil {
			fmt.Println(err)
			return
		}
	}()

	return nil
}

func receive(conn *net.UDPConn) error {
	lastTime := time.Now()
	for {
		err := doReceive(conn, &lastTime)
		if err != nil {
			fmt.Println(err)
			continue
		}

		(*RecvCount)++
	}
}

func doReceive(
	conn *net.UDPConn,
	lastTime *time.Time,
) error {
	err := packetops.Listen(conn, rcvd)
	*lastTime = time.Now()
	if err != nil {
		fmt.Println(err)
		return err
	}

	ack := rcvd.Buf[:headerOffset+8]
	copy(ackBuffer.Buf, ack)

	makeAck(ackBuffer, lastTime.UnixNano())

	err = packetops.SendRaw(conn, ackBuffer)
	if err != nil {
		return err
	}

	return nil
}

func makeAck(ack *packetops.RawPacket, recvTime int64) {
	buf := ack.Buf[headerOffset:]
	encodeInt64(recvTime, buf)
}
