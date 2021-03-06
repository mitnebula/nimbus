package main

import (
	"net"
	"sync"
	"time"

	"github.com/akshayknarayan/history"
	"github.com/akshayknarayan/udp/packetops"
	log "github.com/sirupsen/logrus"
)

const (
	ONE_PACKET = 1500.0 * 8.0
)

var flowRate float64
var flowRateLock sync.Mutex

var reportInterval time.Duration

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
	go output()

	startTime = time.Now()
	modeSwitchTime = startTime
	go send(conn)

	return nil
}

func Sender(ip string, port string) error {
	conn, toAddr, err := packetops.SetupClientSock(ip, port)
	if err != nil {
		log.Error("socket setup err: ", err)
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
		log.Error("synack exchange err: ", err)
	}

	xtcpData.checkXtcpSeq(syn.VirtFid, syn.SeqNo)

	log.Info("sender connected successfully.")

	go handleAck(conn, toAddr, rtt_history)
	go flowRateUpdater()
	go output()

	startTime = time.Now()
	modeSwitchTime = startTime
	go send(conn)

	return nil
}

func output() {
	for _ = range time.Tick(reportInterval) {
		rtt, _ := rtts.Latest()
		inTpt, _, _, err := ThroughputFromTimes(sendTimes, time.Now(), reportInterval)
		if err != nil {
			inTpt = 0
		}
		outTpt, _, _, err := ThroughputFromTimes(ackTimes, time.Now(), reportInterval)
		if err != nil {
			outTpt = 0
		}
		log.WithFields(log.Fields{
			"elapsed": time.Since(startTime),
			"in":      BpsToMbps(inTpt),
			"out":     BpsToMbps(outTpt),
			"rtt":     rtt.(time.Duration),
			"min":     min_rtt,
			"mode":    currMode,
		}).Info()

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

			sendTimes.UpdateDuration(rtt * 1000)
			ackTimes.UpdateDuration(rtt * 1000)
			zt_history.UpdateDuration(rtt * 1000)
			xt_history.UpdateDuration(rtt * 1000)
			zout_history.UpdateDuration(rtt * 1000)

			maxQd = min_rtt
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
			log.Error(err)
			continue
		}

		err = ack.Decode(pktBuf)
		if err != nil {
			log.Error(err)
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
