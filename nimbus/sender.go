package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	ONE_PACKET = 1500.0 * 8.0
)

var flowRate float64
var flowRateLock sync.Mutex

var min_rtt time.Duration

// Log is thread-safe
var rtts *Log
var sendTimes *TimedLog
var ackTimes *TimedLog

type Mode int

const (
	DELAY Mode = iota
	XTCP
	TEST_FROM_DELAY
	TEST_FROM_XTCP
)

var flowMode Mode

func init() {
	flowMode = DELAY

	flowRate = 90e6
	min_rtt = time.Duration(999) * time.Hour

	rtts = InitLog(1000)
	sendTimes = InitTimedLog(min_rtt)
	ackTimes = InitTimedLog(min_rtt)

	sendCount = 0
	recvCount = 0
}

func Server(port string) error {
	conn, addr, err := setupListeningSock(port)
	if err != nil {
		return err
	}

	rtt_history := make(chan int64)
	go rttUpdater(rtt_history)

	conn, err = listenForSyn(conn, addr)
	if err != nil {
		return err
	}

	go handleAck(conn, addr, rtt_history)
	//go flowRateUpdater()
	go output()

	startTime = time.Now()
	go send(conn)

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
func listenForSyn(conn *net.UDPConn, listenAddr *net.UDPAddr) (*net.UDPConn, error) {
	_, fromAddr, err := r.RecvPacket(conn)
	if err != nil {
		return nil, err
	}

	// close and reopen
	conn.Close()

	// dial connection to send ACKs
	newConn, err := net.DialUDP("udp4", listenAddr, fromAddr)
	if err != nil {
		return nil, err
	}

	fmt.Println("connected to ", fromAddr)
	return newConn, nil
}

func output() {
	for _ = range time.Tick(2 * time.Second) {
		rtt, _ := rtts.Latest()
		inTpt, _, _, err := ThroughputFromTimes(sendTimes, time.Now(), time.Duration(2)*time.Second)
		if err != nil {
			inTpt = 0
		}
		outTpt, _, _, err := ThroughputFromTimes(ackTimes, time.Now(), time.Duration(2)*time.Second)
		if err != nil {
			outTpt = 0
		}
		fmt.Printf("%v : %v %v %v %v\n", Now(), inTpt, outTpt, time.Duration(rtt.(durationLogVal)), min_rtt)
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
		}

		rtts.Add(durationLogVal(rtt))
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

func stampTime(pkt *rawPacket, t int64) {
	// write Echo to bytes 6 - 13
	buf := pkt.buf[6:13]
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
	seq, vfid := xtcpData.getNextSeq()

	pkt := Packet{
		SeqNo:   seq,
		VirtFid: vfid,
	}
	rp, err := pkt.makeRaw(1500)
	if err != nil {
		return err
	}

	stampTime(rp, Now())
	r.SendRaw(conn, rp)
	sendTimes.Add(time.Now(), pkt)
	return nil
}

func handleAck(
	conn *net.UDPConn,
	expSrc *net.UDPAddr,
	rtt_history chan int64,
) {
	pktBuf := &receivedBytes{b: make([]byte, 50)}
	for {
		r.Listen(conn, pktBuf)
		pkt, _, err := Decode(pktBuf)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// check for XTCP packet drop
		if ok := xtcpData.checkXtcpSeq(pkt.VirtFid, pkt.SeqNo); !ok {
			xtcpData.dropDetected(pkt.VirtFid)
		} else {
			xtcpData.increaseXtcpWind(pkt.VirtFid)
		}

		recvTime := time.Unix(0, pkt.RecvTime)
		ackTimes.Add(recvTime, pkt)
		recvCount++

		delay := pkt.RecvTime - pkt.Echo // one way delay
		rtt_history <- delay * 2         // multiply one-way delay by 2
	}
}
