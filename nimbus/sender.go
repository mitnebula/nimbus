package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/akshayknarayan/history"
	"github.mit.edu/hari/nimbus-cc/packetops"
)

const (
	ONE_PACKET = 1500.0 * 8.0
)

var flowRate float64
var flowRateLock sync.Mutex

var reportInterval int64

var min_rtt time.Duration

// Log is thread-safe
var rtts *history.QueueHistory
var sendTimes *history.UniqueHistory
var ackTimes *history.UniqueHistory

type Mode int

const (
	DELAY Mode = iota
	XTCP
)

var flowMode Mode

func init() {
	flowMode = DELAY
	currMode = "DELAY"

	flowRate = 0
	min_rtt = time.Duration(999) * time.Hour

	rtts = history.MakeQueueHistory(100)
	sendTimes = history.MakeUniqueHistory(min_rtt)
	ackTimes = history.MakeUniqueHistory(min_rtt)

	sendCount = 0
	recvCount = 0
}

func Server(port string) error {
	conn, addr, err := packetops.SetupListeningSock(port)
	if err != nil {
		return err
	}

	rtt_history := make(chan int64)
	go rttUpdater(rtt_history)

	syn := Packet{}
	conn, err = packetops.ListenForSyn(conn, addr, &syn)
	if err != nil {
		return err
	}

	go handleAck(conn, addr, rtt_history)
	go flowRateUpdater()
	go measurePeriod()
	go output()

	startTime = time.Now()
	modeSwitchTime = startTime
	go send(conn)

	return nil
}

func Sender(ip string, port string) error {
	conn, toAddr, err := packetops.SetupClientSock(ip, port)
	if err != nil {
		return fmt.Errorf("sock setup err: %v", err)
	}

	rtt_history := make(chan int64)
	go rttUpdater(rtt_history)

	seq, vfid := xtcpData.getNextSeqRR()
	syn := Packet{
		SeqNo:   seq,
		VirtFid: vfid,
		Echo:    Now(),
		Payload: "SYN",
	}

	err = packetops.SynAckExchange(conn, toAddr, &syn)
	if err != nil {
		return fmt.Errorf("synack exch err: %v", err)
	}

	xtcpData.checkXtcpSeq(syn.VirtFid, syn.SeqNo)

	fmt.Println("connected")

	go handleAck(conn, toAddr, rtt_history)
	go flowRateUpdater()
	go measurePeriod()
	go output()

	startTime = time.Now()
	modeSwitchTime = startTime
	go send(conn)

	return nil
}

func output() {
	for _ = range time.Tick(time.Duration(reportInterval) * time.Millisecond) {
		rtt, _ := rtts.Latest()
		inTpt, _, _, err := ThroughputFromTimes(sendTimes, time.Now(), time.Duration(reportInterval)*time.Millisecond)
		if err != nil {
			inTpt = 0
		}
		outTpt, _, _, err := ThroughputFromTimes(ackTimes, time.Now(), time.Duration(reportInterval)*time.Millisecond)
		if err != nil {
			outTpt = 0
		}
		fmt.Printf("[%v] : in=%.3fMbps out=%.3fMbps rtt=%.6vms min=%.6vms %s\n",
			NowPretty(),
			BpsToMbps(inTpt),
			BpsToMbps(outTpt),
			rtt.(time.Duration),
			min_rtt,
			currMode)

		if time.Now().After(endTime) {
			doExit()
		}
	}
}

// keep rtt up to date (from received acks)
func rttUpdater(rtt_history chan int64) {
	for t := range rtt_history {
		rtt := time.Duration(t) * time.Nanosecond
		if rtt < min_rtt {
			min_rtt = rtt

			sendTimes.UpdateDuration(rtt * 100)
			ackTimes.UpdateDuration(rtt * 100)
			zt_history.UpdateDuration(rtt * 100)
			xt_history.UpdateDuration(rtt * 100)

			maxQd = 2 * min_rtt
		}

		rtts.Add(rtt)
	}
}

// read the current flow rate and set the pacing channel appropriately
func flowPacer(pacing chan interface{}) {
	credit := float64(ONE_PACKET)
	lastTime := time.Now()

	for _ = range time.Tick(time.Duration(100) * time.Microsecond) {
		elapsed := time.Since(lastTime)
		lastTime = time.Now()

		credit += elapsed.Seconds() * flowRate
		if credit > 100*ONE_PACKET {
			credit = 100 * ONE_PACKET
		}

		for credit >= ONE_PACKET {
			pacing <- struct{}{}
			credit -= ONE_PACKET
		}
	}
}

func stampTime(pkt *packetops.RawPacket, t int64) {
	// write Echo to bytes 6 - 13
	buf := pkt.Buf[6:13]
	encodeInt64(t, buf)
}

func send(
	conn *net.UDPConn,
) error {
	pacing := make(chan interface{})
	go flowPacer(pacing)

	for {
		doSend(conn)
		sendCount++
		<-pacing
	}
}

func doSend(conn *net.UDPConn) error {
	seq, vfid := xtcpData.getNextSeqLottery()

	pkt := Packet{
		SeqNo:   seq,
		VirtFid: vfid,
	}
	rp, err := pkt.Encode(1500)
	if err != nil {
		return err
	}

	stampTime(rp, Now())
	packetops.SendRaw(conn, rp)
	sendTimes.Add(time.Now(), pkt)
	return nil
}

func handleAck(
	conn *net.UDPConn,
	expSrc *net.UDPAddr,
	rtt_history chan int64,
) {
	pktBuf := &packetops.RawPacket{Buf: make([]byte, 50)}
	var ack Packet
	for {
		err := packetops.Listen(conn, pktBuf)
		if err != nil {
			fmt.Println(err)
			continue
		}

		err = ack.Decode(pktBuf)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// check for XTCP packet drop
		if ok := xtcpData.checkXtcpSeq(ack.VirtFid, ack.SeqNo); !ok {
			xtcpData.dropDetected(ack.VirtFid)
		} else {
			xtcpData.increaseXtcpWind(ack.VirtFid)
		}

		recvTime := time.Unix(0, ack.RecvTime)
		ackTimes.Add(recvTime, ack)
		recvCount++

		delay := Now() - ack.Echo
		rtt_history <- delay
	}
}
